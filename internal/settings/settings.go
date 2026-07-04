// Package settings persists user configuration for the LibreTranslate
// translator app in an OS-specific config directory (SPEC §9).
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Shortcut option identifiers stored in Settings.Shortcut.
const (
	ShortcutCtrlEnter = "ctrl_enter" // Ctrl+Enter translates, Enter inserts newline (default)
	ShortcutEnter     = "enter"      // Enter translates, Ctrl+Enter inserts newline
)

// DefaultBaseURL is the LibreTranslate base URL used when none is configured.
const DefaultBaseURL = "https://libretranslate.com"

const (
	appName      = "LibreTranslateTranslator"
	configFile   = "settings.json"
	configDirEnv = "LIBRETRANSLATE_CONFIG_DIR"
	debugEnv     = "LIBRETRANSLATE_DEBUG"
)

// Settings is the full user-configurable state persisted to disk. Field names
// use camelCase JSON tags so the Wails-generated TS model is idiomatic.
type Settings struct {
	BaseURL         string `json:"baseUrl"`
	APIKey          string `json:"apiKey"`
	LiveTranslation bool   `json:"liveTranslation"`
	Shortcut        string `json:"shortcut"`
	DefaultToAuto   bool   `json:"defaultToAuto"`
	LastSourceLang  string `json:"lastSourceLang"`
	LastTargetLang  string `json:"lastTargetLang"`
	AutoCopy        bool   `json:"autoCopy"`
	Debug           bool   `json:"debug"`
}

// DefaultSettings returns the settings used on first run (SPEC §9.3):
// default Base URL, empty API key, Default-to-Auto on, Live Translation off.
func DefaultSettings() Settings {
	return Settings{
		BaseURL:         DefaultBaseURL,
		APIKey:          "",
		LiveTranslation: false,
		Shortcut:        ShortcutCtrlEnter,
		DefaultToAuto:   true,
		LastSourceLang:  "en",
		LastTargetLang:  "en",
		AutoCopy:        false,
		Debug:           false,
	}
}

// EnvDebug reports whether verbose debug logging was requested at launch via
// the LIBRETRANSLATE_DEBUG environment variable (e.g. "1", "true", "yes").
func EnvDebug() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(debugEnv))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// DebugEnabled reports whether verbose logging is active for the given
// settings (persisted toggle OR the launch env override).
func DebugEnabled(s Settings) bool {
	return s.Debug || EnvDebug()
}

// normalize fills empty/invalid fields with defaults and canonicalises values.
func (s Settings) normalize() Settings {
	out := s
	out.BaseURL = strings.TrimSpace(out.BaseURL)
	out.BaseURL = strings.TrimRight(out.BaseURL, "/")
	if out.BaseURL == "" {
		out.BaseURL = DefaultBaseURL
	}
	out.APIKey = strings.TrimSpace(out.APIKey)
	switch out.Shortcut {
	case ShortcutCtrlEnter, ShortcutEnter:
	default:
		out.Shortcut = ShortcutCtrlEnter
	}
	if out.LastSourceLang == "" {
		out.LastSourceLang = "en"
	}
	if out.LastTargetLang == "" {
		out.LastTargetLang = "en"
	}
	return out
}

// Service loads and persists Settings. It is safe for concurrent use.
// It implements the SettingsProvider interface expected by the translator
// package via its GetSettings method.
type Service struct {
	path string
	mu   sync.RWMutex
	cur  Settings
}

// NewService creates a Service backed by the OS config directory and loads
// existing settings (writing defaults when absent).
func NewService() (*Service, error) {
	dir, err := configDir()
	if err != nil {
		return nil, fmt.Errorf("determine config directory: %w", err)
	}
	s := &Service{path: filepath.Join(dir, configFile)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// NewServiceWithDir is a constructor used by tests to isolate the config file.
func NewServiceWithDir(dir string) (*Service, error) {
	s := &Service{path: filepath.Join(dir, configFile)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// configDir resolves the directory holding settings.json, honouring the
// LIBRETRANSLATE_CONFIG_DIR override (useful for tests), otherwise
// os.UserConfigDir() (SPEC §9.1).
func configDir() (string, error) {
	if override := os.Getenv(configDirEnv); override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	cfgDir := filepath.Join(base, appName)

	if EnvDebug() {
		log.Printf("[settings] config directort %v", cfgDir)
	}

	return cfgDir, nil
}

func (s *Service) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cur := DefaultSettings().normalize()
			s.mu.Lock()
			s.cur = cur
			s.mu.Unlock()
			if EnvDebug() {
				log.Printf("[settings] no config at %q; wrote defaults (baseURL=%s)", s.path, cur.BaseURL)
			}
			return s.writeLocked(cur)
		}
		return fmt.Errorf("read settings: %w", err)
	}
	var loaded Settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse settings %q: %w", s.path, err)
	}
	loaded = loaded.normalize()
	s.mu.Lock()
	s.cur = loaded
	s.mu.Unlock()
	if EnvDebug() {
		log.Printf("[settings] loaded from %s: baseURL=%s apiKey=%s liveTranslation=%v shortcut=%s defaultToAuto=%v debug=%v",
			s.path, loaded.BaseURL, apiKeyLabel(loaded.APIKey), loaded.LiveTranslation, loaded.Shortcut, loaded.DefaultToAuto, loaded.Debug)
	}
	return nil
}

// GetSettings returns a snapshot of the current settings.
func (s *Service) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

// Snapshot returns a non-sensitive one-line summary for debug logging.
func (s *Service) Snapshot() string {
	c := s.GetSettings()
	return fmt.Sprintf("baseURL=%s apiKey=%s liveTranslation=%v shortcut=%s defaultToAuto=%v debug=%v",
		c.BaseURL, apiKeyLabel(c.APIKey), c.LiveTranslation, c.Shortcut, c.DefaultToAuto, c.Debug)
}

// SaveSettings validates, persists and atomically replaces the current
// settings. Empty/invalid fields fall back to defaults before writing.
func (s *Service) SaveSettings(next Settings) error {
	normalized := next.normalize()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cur = normalized
	if DebugEnabled(normalized) {
		log.Printf("[settings] saving to %s: baseURL=%s apiKey=%s liveTranslation=%v shortcut=%s defaultToAuto=%v autoCopy=%v debug=%v",
			s.path, normalized.BaseURL, apiKeyLabel(normalized.APIKey), normalized.LiveTranslation, normalized.Shortcut, normalized.DefaultToAuto, normalized.AutoCopy, normalized.Debug)
	}
	return s.writeLocked(normalized)
}

// apiKeyLabel returns a non-sensitive label for logging.
func apiKeyLabel(key string) string {
	if strings.TrimSpace(key) == "" {
		return "<none>"
	}
	return "<set>"
}

func (s *Service) writeLocked(cur Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cur, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod settings: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}

package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	d := DefaultSettings()
	if d.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", d.BaseURL, DefaultBaseURL)
	}
	if d.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", d.APIKey)
	}
	if !d.DefaultToAuto {
		t.Error("DefaultToAuto should be true by default")
	}
	if d.LiveTranslation {
		t.Error("LiveTranslation should be false by default")
	}
	if d.Shortcut != ShortcutCtrlEnter {
		t.Errorf("Shortcut = %q, want %q", d.Shortcut, ShortcutCtrlEnter)
	}
}

func TestNewServiceCreatesDefaultsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	s, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("NewServiceWithDir: %v", err)
	}
	got := s.GetSettings()
	if got.BaseURL != DefaultBaseURL || !got.DefaultToAuto {
		t.Errorf("defaults not applied: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, configFile)); err != nil {
		t.Errorf("settings file not written on first run: %v", err)
	}
}

func TestSaveThenReload(t *testing.T) {
	dir := t.TempDir()
	s, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("NewServiceWithDir: %v", err)
	}
	want := Settings{
		BaseURL:         "https://lt.example.com",
		APIKey:          "secret",
		LiveTranslation: true,
		Shortcut:        ShortcutEnter,
		DefaultToAuto:   false,
		LastSourceLang:  "fr",
		LastTargetLang:  "ru",
		Debug:           true,
	}
	if err := s.SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	s2, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := s2.GetSettings()
	if got != want {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestSaveNormalizesInvalidValues(t *testing.T) {
	dir := t.TempDir()
	s, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("NewServiceWithDir: %v", err)
	}
	if err := s.SaveSettings(Settings{
		BaseURL:        "  https://lt.example.com/  ",
		Shortcut:       "bogus",
		LastTargetLang: "",
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got := s.GetSettings()
	if got.BaseURL != "https://lt.example.com" {
		t.Errorf("BaseURL not trimmed: %q", got.BaseURL)
	}
	if got.Shortcut != ShortcutCtrlEnter {
		t.Errorf("invalid Shortcut not defaulted: %q", got.Shortcut)
	}
	if got.LastTargetLang != "en" {
		t.Errorf("empty LastTargetLang not defaulted: %q", got.LastTargetLang)
	}
}

func TestSaveNormalizesSchemelessBaseURL(t *testing.T) {
	dir := t.TempDir()
	s, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("NewServiceWithDir: %v", err)
	}
	if err := s.SaveSettings(Settings{BaseURL: "lt.example.com/"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if got := s.GetSettings().BaseURL; got != "https://lt.example.com" {
		t.Errorf("schemeless BaseURL not normalized: %q", got)
	}
}

func TestUpdateSettingsMergesPatch(t *testing.T) {
	dir := t.TempDir()
	s, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("NewServiceWithDir: %v", err)
	}
	base := Settings{
		BaseURL:        "https://lt.example.com",
		APIKey:         "secret",
		LastSourceLang: "fr",
		LastTargetLang: "ru",
		AutoCopy:       true,
	}
	if err := s.SaveSettings(base); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	got, err := s.UpdateSettings(map[string]any{
		"lastSourceLang": "de",
		"lastTargetLang": "en",
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if got.LastSourceLang != "de" || got.LastTargetLang != "en" {
		t.Errorf("patched fields not applied: %+v", got)
	}
	if got.BaseURL != base.BaseURL || got.APIKey != "secret" || !got.AutoCopy {
		t.Errorf("unpatched fields overwritten: %+v", got)
	}

	// The merge must also survive a reload from disk.
	s2, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded := s2.GetSettings(); reloaded != got {
		t.Errorf("persisted mismatch:\n got  %+v\n want %+v", reloaded, got)
	}
}

func TestUpdateSettingsRejectsBadPatch(t *testing.T) {
	dir := t.TempDir()
	s, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("NewServiceWithDir: %v", err)
	}
	before := s.GetSettings()
	if _, err := s.UpdateSettings(map[string]any{"liveTranslation": "not-a-bool"}); err == nil {
		t.Error("expected error for type-mismatched patch, got nil")
	}
	if after := s.GetSettings(); after != before {
		t.Errorf("failed patch mutated settings:\n got  %+v\n want %+v", after, before)
	}
}

func TestCorruptFileFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFile)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("NewServiceWithDir should not fail on corrupt file: %v", err)
	}
	if got := s.GetSettings(); got.BaseURL != DefaultBaseURL || !got.DefaultToAuto {
		t.Errorf("expected defaults after corrupt load, got %+v", got)
	}
	if s.LoadWarning() == "" {
		t.Error("expected a non-empty load warning after corrupt file")
	}
	if _, err := os.Stat(path + ".corrupt"); err != nil {
		t.Errorf("corrupt file not backed up to %s: %v", path+".corrupt", err)
	}

	// The file at path is now valid defaults, so a fresh load is clean.
	s2, err := NewServiceWithDir(dir)
	if err != nil {
		t.Fatalf("reload after reset: %v", err)
	}
	if s2.LoadWarning() != "" {
		t.Errorf("second load should be clean, got warning: %q", s2.LoadWarning())
	}
}

func TestInMemoryServiceNeverPersists(t *testing.T) {
	s := NewInMemoryService("boom")
	if s.LoadWarning() != "boom" {
		t.Errorf("LoadWarning = %q, want %q", s.LoadWarning(), "boom")
	}
	if got := s.GetSettings().BaseURL; got != DefaultBaseURL {
		t.Errorf("in-memory service should hold defaults, got BaseURL=%q", got)
	}
	// SaveSettings must not panic or error despite having no path.
	if err := s.SaveSettings(Settings{BaseURL: "https://x.example"}); err != nil {
		t.Errorf("SaveSettings on in-memory service: %v", err)
	}
	if got := s.GetSettings().BaseURL; got != "https://x.example" {
		t.Errorf("in-memory update not reflected in memory: %q", got)
	}
}

func TestJSONTagsRoundTrip(t *testing.T) {
	s := Settings{BaseURL: "u", APIKey: "k", LiveTranslation: true, Shortcut: "enter",
		DefaultToAuto: true, LastSourceLang: "a", LastTargetLang: "b", Debug: true}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"baseUrl", "apiKey", "liveTranslation", "shortcut", "defaultToAuto", "lastSourceLang", "lastTargetLang", "debug"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing camelCase key %q in JSON: %v", k, string(data))
		}
	}
}

func TestEnvDebug(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv(debugEnv, v)
		if !EnvDebug() {
			t.Errorf("EnvDebug()=false for %q", v)
		}
	}
	for _, v := range []string{"", "0", "false", "no", "off", "maybe"} {
		t.Setenv(debugEnv, v)
		if EnvDebug() {
			t.Errorf("EnvDebug()=true for %q", v)
		}
	}
}

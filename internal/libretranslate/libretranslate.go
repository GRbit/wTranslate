// Package libretranslate is the pure-Go client for the LibreTranslate API
// (SPEC §4). It performs /languages and /translate requests with per-request
// timeouts and classifies errors for the UI (SPEC §7).
package libretranslate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"translator/internal/settings"
)

// requestTimeout is the per-request deadline (SPEC §7.1: ~10s). It is a var
// (not const) so tests can shorten it without waiting 10 seconds.
var requestTimeout = 10 * time.Second

const userAgent = "LibreTranslate-Desktop/1.0"

// SettingsProvider is satisfied by *settings.Service; kept as an interface so
// the package is testable with a fake provider (SPEC §8.2.4, §10.3).
type SettingsProvider interface {
	GetSettings() settings.Settings
}

// Language is one entry from GET /languages (SPEC §4.1.2).
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// TranslateRequest is the payload sent from the UI to Translate.
type TranslateRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"` // language code or "auto"
	Target string `json:"target"`
}

// DetectedLanguage is returned by /translate when source == "auto".
type DetectedLanguage struct {
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

// TranslateResponse mirrors the LibreTranslate /translate success body.
type TranslateResponse struct {
	TranslatedText   string            `json:"translatedText"`
	DetectedLanguage *DetectedLanguage `json:"detectedLanguage,omitempty"`
}

// APIError represents a non-2xx response from LibreTranslate (e.g. HTTP 400
// validation errors, 403 bad API key). Its message is shown to the user.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("LibreTranslate API error (HTTP %d): %s", e.Status, e.Message)
}

// Service calls the LibreTranslate API. The base URL and API key are read from
// the SettingsProvider on every call so settings changes take effect at once.
type Service struct {
	client   *http.Client
	settings SettingsProvider
}

// NewService builds a Service backed by the given settings provider.
func NewService(sp SettingsProvider) *Service {
	return &Service{
		client:   &http.Client{},
		settings: sp,
	}
}

// baseURL returns the normalised base URL for the current settings.
func (s *Service) baseURL() string {
	b := strings.TrimSpace(s.settings.GetSettings().BaseURL)
	b = strings.TrimRight(b, "/")
	if b == "" {
		b = settings.DefaultBaseURL
	}
	return b
}

func (s *Service) apiKey() string {
	return strings.TrimSpace(s.settings.GetSettings().APIKey)
}

// debug reports whether verbose logging is active (SPEC-add: debug option).
func (s *Service) debug() bool {
	return settings.DebugEnabled(s.settings.GetSettings())
}

// logf logs only when debug is enabled. The API key is never logged.
func (s *Service) logf(format string, args ...any) {
	if s.debug() {
		log.Printf("[libretranslate] "+format, args...)
	}
}

// apiKeyLabel returns a non-sensitive marker for logs.
func apiKeyLabel(key string) string {
	if strings.TrimSpace(key) == "" {
		return "<none>"
	}
	return "<set>"
}

// GetLanguages calls GET {BaseURL}/languages (SPEC §4.1.1).
func (s *Service) GetLanguages() ([]Language, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	fullURL := s.baseURL() + "/languages"
	s.logf("GET %s apiKey=%s", fullURL, apiKeyLabel(s.apiKey()))
	start := time.Now()

	// Some gated instances require the API key even for /languages; a GET has
	// no body, so it goes in the query string. Only the bare URL is logged.
	requestURL := fullURL
	if key := s.apiKey(); key != "" {
		requestURL += "?api_key=" + url.QueryEscape(key)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		s.logf("GET %s: build request failed: %v", fullURL, err)
		return nil, fmt.Errorf("invalid Base URL: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logf("GET %s: request failed after %s: %v", fullURL, time.Since(start), err)
		return nil, s.classifyErr(ctx, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logf("GET %s: read body failed: %v", fullURL, err)
		return nil, fmt.Errorf("reading /languages response: %w", err)
	}
	s.logf("GET %s -> %d (%s), %d bytes", fullURL, resp.StatusCode, time.Since(start), len(body))

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Status: resp.StatusCode, Message: extractErrorMessage(body)}
	}

	var langs []Language
	if err := json.Unmarshal(body, &langs); err != nil {
		return nil, fmt.Errorf("unexpected /languages response (HTTP %d): %s", resp.StatusCode, snippet(body))
	}
	s.logf("GET %s: parsed %d languages", fullURL, len(langs))
	return langs, nil
}

// Translate calls POST {BaseURL}/translate with a JSON body (SPEC §4.2).
func (s *Service) Translate(r TranslateRequest) (TranslateResponse, error) {
	if strings.TrimSpace(r.Q) == "" {
		return TranslateResponse{}, nil
	}
	if r.Target == "" {
		return TranslateResponse{}, errors.New("target language is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	payload := map[string]string{
		"q":      r.Q,
		"source": r.Source,
		"target": r.Target,
		"format": "text",
	}
	if key := s.apiKey(); key != "" {
		payload["api_key"] = key
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("encode request: %w", err)
	}

	fullURL := s.baseURL() + "/translate"
	s.logf("POST %s source=%s target=%s format=text qLen=%d apiKey=%s",
		fullURL, r.Source, r.Target, len(r.Q), apiKeyLabel(s.apiKey()))
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(body))
	if err != nil {
		s.logf("POST %s: build request failed: %v", fullURL, err)
		return TranslateResponse{}, fmt.Errorf("invalid Base URL: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logf("POST %s: request failed after %s: %v", fullURL, time.Since(start), err)
		return TranslateResponse{}, s.classifyErr(ctx, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logf("POST %s: read body failed: %v", fullURL, err)
		return TranslateResponse{}, fmt.Errorf("reading /translate response: %w", err)
	}
	s.logf("POST %s -> %d (%s), %d bytes", fullURL, resp.StatusCode, time.Since(start), len(respBody))

	if resp.StatusCode != http.StatusOK {
		return TranslateResponse{}, &APIError{Status: resp.StatusCode, Message: extractErrorMessage(respBody)}
	}

	var out TranslateResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return TranslateResponse{}, fmt.Errorf("unexpected /translate response (HTTP %d): %s", resp.StatusCode, snippet(respBody))
	}
	if out.TranslatedText == "" {
		return TranslateResponse{}, fmt.Errorf("empty translation in response (HTTP %d): %s", resp.StatusCode, snippet(respBody))
	}
	s.logf("POST %s: translatedLen=%d detected=%v", fullURL, len(out.TranslatedText), detectedLabel(out.DetectedLanguage))
	s.logf("resp %v", string(respBody))
	return out, nil
}

// detectedLabel renders the detected-language info for logs (or "<none>").
func detectedLabel(d *DetectedLanguage) string {
	if d == nil {
		return "<none>"
	}
	return fmt.Sprintf("%s(%.0f%%)", d.Language, d.Confidence)
}

// classifyErr converts a transport error into a user-friendly message per
// SPEC §7.1/§7.2 (timeout vs. unreachable/wrong URL).
func (s *Service) classifyErr(ctx context.Context, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("Server timeout: LibreTranslate did not respond within %s. Check the Base URL/API key in Settings.", requestTimeout)
	}
	// DNS / connection refused / wrong host → "cannot connect".
	var netErr net.Error
	if errors.As(err, &netErr) || isURLError(err) {
		return fmt.Errorf("Cannot connect to %s (%v). Check the LibreTranslate Base URL in Settings.", s.baseURL(), unwrapReason(err))
	}
	return fmt.Errorf("Network error contacting %s: %v. Check the Base URL in Settings.", s.baseURL(), err)
}

func isURLError(err error) bool {
	var ue *url.Error
	return errors.As(err, &ue)
}

func unwrapReason(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// extractErrorMessage parses a LibreTranslate error body of the forms
// `{"error":"msg"}` or `{"error":{"message":"msg"}}`. Falls back to a body
// snippet for non-JSON (e.g. HTML 404 from a wrong Base URL) per SPEC §7.2.
func extractErrorMessage(body []byte) string {
	var raw struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err == nil && len(raw.Error) > 0 {
		var s string
		if json.Unmarshal(raw.Error, &s) == nil {
			return strings.TrimSpace(s)
		}
		var obj struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(raw.Error, &obj) == nil && obj.Message != "" {
			return obj.Message
		}
		return strings.TrimSpace(string(raw.Error))
	}
	return "unexpected response: " + snippet(body)
}

func snippet(body []byte) string {
	const max = 200
	t := strings.TrimSpace(string(body))
	t = strings.ReplaceAll(t, "\n", " ")
	if len(t) > max {
		// Back up to a rune boundary so the cut never splits a UTF-8 sequence.
		cut := max
		for cut > 0 && !utf8.RuneStart(t[cut]) {
			cut--
		}
		return t[:cut] + "…"
	}
	return t
}

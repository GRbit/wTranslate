package libretranslate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"translator/internal/settings"
)

type fakeSettings struct{ s settings.Settings }

func (f fakeSettings) GetSettings() settings.Settings { return f.s }

func newSvc(t *testing.T, baseURL, apiKey string) *Service {
	t.Helper()
	return NewService(fakeSettings{s: settings.Settings{BaseURL: baseURL, APIKey: apiKey}})
}

func readJSON(t *testing.T, r io.Reader) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return m
}

func TestGetLanguagesSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/languages" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"code":"en","name":"English"},{"code":"ru","name":"Russian"}]`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	langs, err := svc.GetLanguages()
	if err != nil {
		t.Fatalf("GetLanguages: %v", err)
	}
	if len(langs) != 2 || langs[0].Code != "en" || langs[1].Name != "Russian" {
		t.Errorf("unexpected languages: %+v", langs)
	}
}

func TestGetFrontendSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/frontend/settings" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("api_key"); got != "k" {
			t.Errorf("api_key query = %q, want %q", got, "k")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"charLimit":2000,"keyRequired":false}`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "k")
	fs, err := svc.GetFrontendSettings()
	if err != nil {
		t.Fatalf("GetFrontendSettings: %v", err)
	}
	if fs.CharLimit != 2000 {
		t.Errorf("CharLimit = %d, want 2000", fs.CharLimit)
	}
}

func TestGetFrontendSettingsUnlimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"charLimit":-1}`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	fs, err := svc.GetFrontendSettings()
	if err != nil {
		t.Fatalf("GetFrontendSettings: %v", err)
	}
	if fs.CharLimit != -1 {
		t.Errorf("CharLimit = %d, want -1 (unlimited)", fs.CharLimit)
	}
}

func TestTranslateWithAutoDetect(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/translate" {
			t.Errorf("path = %q, want /translate", r.URL.Path)
		}
		gotBody = readJSON(t, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translatedText":"Bonjour","detectedLanguage":{"language":"fr","confidence":90.0}}`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	res, err := svc.Translate(TranslateRequest{Q: "Hello", Source: "auto", Target: "ru"})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if res.TranslatedText != "Bonjour" {
		t.Errorf("translatedText = %q", res.TranslatedText)
	}
	if res.DetectedLanguage == nil || res.DetectedLanguage.Language != "fr" || res.DetectedLanguage.Confidence != 90.0 {
		t.Errorf("detectedLanguage = %+v", res.DetectedLanguage)
	}
	if gotBody["source"] != "auto" || gotBody["target"] != "ru" || gotBody["format"] != "text" {
		t.Errorf("request body wrong: %+v", gotBody)
	}
	if _, ok := gotBody["api_key"]; ok {
		t.Error("api_key should not be sent when empty")
	}
}

func TestTranslateWithExplicitSourceAndAPIKey(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody = readJSON(t, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translatedText":"Hola"}`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "secret-key")
	res, err := svc.Translate(TranslateRequest{Q: "Hello", Source: "en", Target: "es"})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if res.TranslatedText != "Hola" {
		t.Errorf("translatedText = %q", res.TranslatedText)
	}
	if res.DetectedLanguage != nil {
		t.Errorf("detectedLanguage should be nil for explicit source, got %+v", res.DetectedLanguage)
	}
	if gotBody["api_key"] != "secret-key" {
		t.Errorf("api_key not forwarded: %+v", gotBody)
	}
}

func TestTranslateEmptyInputSkipsRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	res, err := svc.Translate(TranslateRequest{Q: "   ", Source: "auto", Target: "ru"})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if res.TranslatedText != "" {
		t.Errorf("expected empty response, got %q", res.TranslatedText)
	}
	if called {
		t.Error("server should not be called for empty input")
	}
}

func TestTranslateMissingTarget(t *testing.T) {
	svc := newSvc(t, "https://example.com", "")
	if _, err := svc.Translate(TranslateRequest{Q: "Hi", Source: "auto"}); err == nil {
		t.Error("expected error for missing target language")
	}
}

func TestTranslateAPIErrorString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"Invalid target language"}`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	_, err := svc.Translate(TranslateRequest{Q: "Hi", Source: "auto", Target: "zzz"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 400 || !strings.Contains(apiErr.Message, "Invalid target language") {
		t.Errorf("unexpected APIError: %+v", apiErr)
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("error message should mention HTTP 400: %v", err)
	}
}

func TestTranslateAPIErrorObjectForm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"API key required"}}`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	_, err := svc.Translate(TranslateRequest{Q: "Hi", Source: "auto", Target: "ru"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 403 || !strings.Contains(apiErr.Message, "API key required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrongURLReturnsAPIErrorFor404HTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<html><body>Not Found</body></html>`))
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	_, err := svc.GetLanguages()
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 404 {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
	if !strings.Contains(apiErr.Message, "unexpected response") {
		t.Errorf("404 message should mention unexpected response: %v", apiErr)
	}
}

func TestNetworkErrorCannotConnect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // free the port → connection refused

	svc := newSvc(t, srv.URL, "")
	_, err := svc.GetLanguages()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Cannot connect") {
		t.Errorf("expected 'Cannot connect' message, got: %v", err)
	}
}

func TestTimeoutError(t *testing.T) {
	orig := requestTimeout
	requestTimeout = 40 * time.Millisecond
	t.Cleanup(func() { requestTimeout = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	svc := newSvc(t, srv.URL, "")
	_, err := svc.Translate(TranslateRequest{Q: "Hi", Source: "auto", Target: "ru"})
	if err == nil || !strings.Contains(err.Error(), "Server timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestDebugLoggingEmitsNetworkDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translatedText":"Hola","detectedLanguage":{"language":"es","confidence":99.0}}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origOut) })

	svc := NewService(fakeSettings{s: settings.Settings{BaseURL: srv.URL, Debug: true}})
	_, err := svc.Translate(TranslateRequest{Q: "Hello", Source: "auto", Target: "es"})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"POST", "/translate", "source=auto", "target=es", "-> 200", "detected=es(99%)", "apiKey=<none>"} {
		if !strings.Contains(out, want) {
			t.Errorf("debug log missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestDebugDisabledByDefaultIsQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"translatedText":"Hola"}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origOut) })

	svc := newSvc(t, srv.URL, "") // Debug: false, no env
	_, _ = svc.Translate(TranslateRequest{Q: "Hello", Source: "en", Target: "es"})
	if strings.TrimSpace(buf.String()) != "" {
		t.Errorf("expected no debug logs when debug disabled, got:\n%s", buf.String())
	}
}

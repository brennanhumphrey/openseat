package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

// ===================
// Mock email sender for testing
// ===================

type MockEmailSender struct {
	Sent []struct {
		To      string
		Subject string
		Body    string
	}
	ShouldError bool
}

func (m *MockEmailSender) Send(to, subject, body string) error {
	if m.ShouldError {
		return fmt.Errorf("mock email error")
	}
	m.Sent = append(m.Sent, struct {
		To      string
		Subject string
		Body    string
	}{to, subject, body})
	return nil
}

// ===================
// Helper to create temp config files
// ===================

func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	return tmpfile.Name()
}

// ===================
// loadConfig tests
// ===================

func TestLoadConfig_Valid(t *testing.T) {
	path := createTempConfig(t, `{
		"crns": ["12345", "67890"],
		"email": "test@example.com",
		"checkInterval": 60
	}`)
	defer os.Remove(path)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.CRNs) != 2 {
		t.Errorf("expected 2 CRNs, got %d", len(cfg.CRNs))
	}
	if cfg.CheckInterval != 60 {
		t.Errorf("expected checkInterval 60, got %d", cfg.CheckInterval)
	}
}

func TestLoadConfig_AppliesDefaults(t *testing.T) {
	path := createTempConfig(t, `{"crns": ["12345"]}`)
	defer os.Remove(path)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CheckInterval != 30 {
		t.Errorf("expected default checkInterval 30, got %d", cfg.CheckInterval)
	}
	if cfg.Campus != "0" {
		t.Errorf("expected default campus '0', got '%s'", cfg.Campus)
	}
	if cfg.Term != "202601" {
		t.Errorf("expected default term '202601', got '%s'", cfg.Term)
	}
	if cfg.BaseURL != DefaultTimetableURL {
		t.Errorf("expected default BaseURL, got '%s'", cfg.BaseURL)
	}
}

func TestLoadConfig_ErrorNoCRNs(t *testing.T) {
	path := createTempConfig(t, `{"email": "test@example.com"}`)
	defer os.Remove(path)

	_, err := loadConfig(path)
	if err == nil {
		t.Error("expected error for missing CRNs")
	}
}

func TestLoadConfig_ErrorFileNotFound(t *testing.T) {
	_, err := loadConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfig_ErrorInvalidJSON(t *testing.T) {
	path := createTempConfig(t, `{not valid json`)
	defer os.Remove(path)

	_, err := loadConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ===================
// buildPayload tests
// ===================

func TestBuildPayload_IncludesCRN(t *testing.T) {
	cfg := Config{Campus: "0", Term: "202601"}
	payload := cfg.buildPayload("12345", false)

	if got := payload.Get("crn"); got != "12345" {
		t.Errorf("crn = %q, want %q", got, "12345")
	}
}

func TestBuildPayload_IncludesTermAndCampus(t *testing.T) {
	cfg := Config{Campus: "1", Term: "202509"}
	payload := cfg.buildPayload("99999", false)

	if got := payload.Get("CAMPUS"); got != "1" {
		t.Errorf("CAMPUS = %q, want %q", got, "1")
	}
	if got := payload.Get("TERMYEAR"); got != "202509" {
		t.Errorf("TERMYEAR = %q, want %q", got, "202509")
	}
}

func TestBuildPayload_OpenOnlyFalse(t *testing.T) {
	cfg := Config{Campus: "0", Term: "202601"}
	payload := cfg.buildPayload("12345", false)

	if got := payload.Get("open_only"); got != "" {
		t.Errorf("open_only = %q, want empty", got)
	}
}

func TestBuildPayload_OpenOnlyTrue(t *testing.T) {
	cfg := Config{Campus: "0", Term: "202601"}
	payload := cfg.buildPayload("12345", true)

	if got := payload.Get("open_only"); got != "on" {
		t.Errorf("open_only = %q, want %q", got, "on")
	}
}

// ===================
// fetchDocument tests
// ===================

func TestFetchDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><div class="dataentrytable">content</div></html>`))
	}))
	defer server.Close()

	doc, err := fetchDocument(server.URL, url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if text := doc.Find(".dataentrytable").Text(); text != "content" {
		t.Errorf("got %q, want %q", text, "content")
	}
}

func TestFetchDocument_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchDocument(server.URL, url.Values{})
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestFetchDocument_NetworkError(t *testing.T) {
	_, err := fetchDocument("http://localhost:99999", url.Values{})
	if err == nil {
		t.Error("expected error for connection refused")
	}
}

// ===================
// checkSectionOpen tests
// ===================

func TestCheckSectionOpen_SeatAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's requesting open_only
		r.ParseForm()
		if r.FormValue("open_only") != "on" {
			t.Error("expected open_only=on in request")
		}
		w.Write([]byte(`<table class="dataentrytable"><tr><td>12345</td></tr></table>`))
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Campus: "0", Term: "202601"}
	open, err := cfg.checkSectionOpen("12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !open {
		t.Error("expected open=true when CRN is in results")
	}
}

func TestCheckSectionOpen_NoSeatAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty table (no matching CRN)
		w.Write([]byte(`<table class="dataentrytable"></table>`))
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Campus: "0", Term: "202601"}
	open, err := cfg.checkSectionOpen("12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if open {
		t.Error("expected open=false when CRN not in results")
	}
}

func TestCheckSectionOpen_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Campus: "0", Term: "202601"}
	_, err := cfg.checkSectionOpen("12345")
	if err == nil {
		t.Error("expected error for server failure")
	}
}

// ===================
// getCourseName tests
// ===================

func TestGetCourseName_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<table class="dataentrytable">
				<tr><td>12345</td><td>001</td><td>Intro to Testing</td></tr>
			</table>
		`))
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Campus: "0", Term: "202601"}
	name, err := cfg.getCourseName("12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "Intro to Testing" {
		t.Errorf("got %q, want %q", name, "Intro to Testing")
	}
}

func TestGetCourseName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<table class="dataentrytable"></table>`))
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Campus: "0", Term: "202601"}
	_, err := cfg.getCourseName("99999")
	if err == nil {
		t.Error("expected error for CRN not found")
	}
}

// ===================
// ResendEmailSender tests
// ===================

func TestResendEmailSender_NoAPIKey(t *testing.T) {
	sender := &ResendEmailSender{APIKey: ""}
	err := sender.Send("to@example.com", "Subject", "Body")
	if err == nil {
		t.Error("expected error when API key is empty")
	}
}

// ===================
// Integration-style test for Run (optional)
// ===================

func TestRun_InvalidConfigPath(t *testing.T) {
	err := Run(RunOptions{ConfigPath: "/nonexistent/config.json"})
	if err == nil {
		t.Error("expected error for invalid config path")
	}
}

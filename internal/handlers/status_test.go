package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"formfling/internal/config"
)

func TestStatusHandler_UsesConfiguredRedirect(t *testing.T) {
	handler := NewStatusHandler(&config.Config{
		StatusTemplate:    "../../web/templates/status_template.html",
		StatusRedirectURL: "https://example.com/contact?source=form&result=done",
	})
	req := httptest.NewRequest(http.MethodGet, "/status?type=error&redirect=javascript:alert(1)", nil)
	req.Header.Set("Referer", "https://attacker.example/phishing")
	recorder := httptest.NewRecorder()

	handler.Handle(recorder, req)
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(body, `href="https://example.com/contact?source=form&amp;result=done"`) {
		t.Error("status page did not use the configured return URL")
	}
	if strings.Contains(body, "javascript:alert") || strings.Contains(body, "attacker.example") {
		t.Error("status page included a request-supplied redirect")
	}
}

func TestStatusHandler_WithoutConfiguredRedirect(t *testing.T) {
	handler := NewStatusHandler(&config.Config{
		StatusTemplate: "../../web/templates/status_template.html",
	})
	req := httptest.NewRequest(http.MethodGet, "/status?redirect=https://attacker.example", nil)
	recorder := httptest.NewRecorder()

	handler.Handle(recorder, req)
	body := recorder.Body.String()
	if strings.Contains(body, "attacker.example") || strings.Contains(body, `id="return-link"`) {
		t.Error("status page rendered a return URL without configuration")
	}
}

func TestStatusHandler_RateLimitedMessage(t *testing.T) {
	handler := NewStatusHandler(&config.Config{StatusTemplate: "../../web/templates/status_template.html"})
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, httptest.NewRequest(http.MethodGet, "/status?type=rate_limited", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Message Limit Reached") {
		t.Fatalf("rate limit status page: HTTP %d, body %q", recorder.Code, recorder.Body.String())
	}
}

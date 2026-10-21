package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"formfling/internal/config"
	"formfling/internal/models"
	"formfling/internal/services"
)

func newTestSubmitHandler(email services.EmailSender) *SubmitHandler {
	return NewSubmitHandler(email, config.RateLimitSettings{MaxEmails: 3, Window: 24 * time.Hour}, "https://example.com/contact?source=form#old-section")
}

// Mock email service for testing
type mockEmailService struct {
	shouldFail bool
	calls      int
}

func (m *mockEmailService) SendEmail(formData models.FormData, origin string) error {
	m.calls++
	if m.shouldFail {
		return errors.New("mock email service error")
	}
	return nil
}

func TestSubmitHandler_RejectsOversizedBodies(t *testing.T) {
	tests := []struct {
		name          string
		contentType   string
		body          string
		unknownLength bool
		acceptJSON    bool
	}{
		{
			name:          "JSON without known length",
			contentType:   "application/json",
			body:          strings.Repeat("x", int(config.MaxRequestBodyBytes)+1),
			unknownLength: true,
			acceptJSON:    true,
		},
		{
			name:        "URL encoded with known length",
			contentType: "application/x-www-form-urlencoded",
			body:        "message=" + strings.Repeat("x", int(config.MaxRequestBodyBytes)),
		},
		{
			name:          "URL encoded without known length",
			contentType:   "application/x-www-form-urlencoded",
			body:          "message=" + strings.Repeat("x", int(config.MaxRequestBodyBytes)),
			unknownLength: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := &mockEmailService{}
			handler := newTestSubmitHandler(email)
			req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			if tt.unknownLength {
				req.ContentLength = -1
			}
			if tt.acceptJSON {
				req.Header.Set("Accept", "application/json")
			}
			recorder := httptest.NewRecorder()

			handler.Handle(recorder, req)
			wantStatus := http.StatusSeeOther
			if tt.acceptJSON {
				wantStatus = http.StatusRequestEntityTooLarge
			}
			if recorder.Code != wantStatus {
				t.Errorf("expected %d, got %d", wantStatus, recorder.Code)
			}
			if !tt.acceptJSON && recorder.Header().Get("Location") != "https://example.com/contact?source=form#form-error" {
				t.Errorf("unexpected browser redirect: %q", recorder.Header().Get("Location"))
			}
			if email.calls != 0 {
				t.Error("oversized request reached the email service")
			}
		})
	}
}

func TestSubmitHandler_MultipartBodyLimit(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted", true: "rejected"}[oversized], func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			writer.WriteField("name", "Alice")
			writer.WriteField("email", "alice@example.com")
			message := strings.Repeat("valid message ", 30)
			if oversized {
				message = strings.Repeat("x", int(config.MaxRequestBodyBytes))
			}
			writer.WriteField("message", message)
			writer.Close()

			email := &mockEmailService{}
			handler := newTestSubmitHandler(email)
			req := httptest.NewRequest(http.MethodPost, "/submit", &body)
			req.ContentLength = -1
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("Accept", "application/json")
			recorder := httptest.NewRecorder()

			handler.Handle(recorder, req)
			if oversized {
				if recorder.Code != http.StatusRequestEntityTooLarge || email.calls != 0 {
					t.Errorf("expected 413 without email, got status %d and %d email calls", recorder.Code, email.calls)
				}
			} else if recorder.Code != http.StatusOK || email.calls != 1 {
				t.Errorf("expected successful email, got status %d and %d email calls", recorder.Code, email.calls)
			}
		})
	}
}

// Ensure mockEmailService implements EmailSender
var _ services.EmailSender = (*mockEmailService)(nil)

func TestSubmitHandler_RedirectMode(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Test form submission without AJAX headers (should redirect)
	formData := url.Values{
		"name":    {"John Doe"},
		"email":   {"john@example.com"},
		"message": {"This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters."},
	}

	req, err := http.NewRequest("POST", "/submit", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://example.com/contact")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should redirect (303 status)
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Expected status 303, got %v", status)
	}

	// Should have location header with success status
	location := rr.Header().Get("Location")
	if location != "https://example.com/contact?source=form#form-success" {
		t.Errorf("Expected success redirect, got %s", location)
	}
}

func TestSubmitHandler_AjaxMode(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	formData := url.Values{
		"name":    {"John Doe"},
		"email":   {"john@example.com"},
		"message": {"This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters."},
	}

	req, err := http.NewRequest("POST", "/submit", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest") // AJAX indicator

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should return JSON (200 status)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v", status)
	}

	// Should have JSON content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type application/json, got %s", contentType)
	}

	// Should contain success message
	responseBody := rr.Body.String()
	if !strings.Contains(responseBody, "message sent") {
		t.Errorf("Expected response to contain 'message sent', got %s", responseBody)
	}
}

func TestSubmitHandler_JSONRequest_Success(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Create JSON request body
	formData := models.FormData{
		Name:    "John Doe",
		Email:   "john@example.com",
		Message: "This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters.",
	}

	jsonData, err := json.Marshal(formData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/submit", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should return JSON (200 status) because of JSON content type
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %v", status)
	}

	// Should have JSON content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type application/json, got %s", contentType)
	}

	// Parse response
	var response models.Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Could not unmarshal response: %v", err)
	}

	if response.Status != "message sent" {
		t.Errorf("Expected status 'message sent', got %s", response.Status)
	}
}

func TestSubmitHandler_HoneypotBrowserFormDoesNotSendOrUseLimit(t *testing.T) {
	email := &mockEmailService{}
	handler := NewSubmitHandler(email, config.RateLimitSettings{MaxEmails: 1, Window: time.Hour}, "https://example.com/contact")

	spam := url.Values{"company_website": {"https://spam.example"}}
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(spam.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "198.51.100.1:1234"
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "https://example.com/contact#form-success" || email.calls != 0 {
		t.Fatalf("honeypot response: HTTP %d, location %q, email calls %d", recorder.Code, recorder.Header().Get("Location"), email.calls)
	}

	valid := url.Values{
		"name":            {"Alice"},
		"email":           {"alice@example.com"},
		"message":         {strings.Repeat("valid message ", 30)},
		"company_website": {"   "},
	}
	req = httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(valid.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "198.51.100.1:1234"
	recorder = httptest.NewRecorder()
	handler.Handle(recorder, req)
	if recorder.Code != http.StatusSeeOther || email.calls != 1 {
		t.Fatalf("valid follow-up: HTTP %d, email calls %d", recorder.Code, email.calls)
	}
}

func TestSubmitHandler_HoneypotJSONDoesNotSend(t *testing.T) {
	email := &mockEmailService{}
	handler := newTestSubmitHandler(email)
	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(`{"company_website":"https://spam.example"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	var response models.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || response.Status != "message sent" || email.calls != 0 {
		t.Fatalf("honeypot response: HTTP %d, body %q, email calls %d", recorder.Code, recorder.Body.String(), email.calls)
	}
}

func TestSubmitHandler_JSONRequest_ValidationError(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Create JSON request body with invalid data
	formData := models.FormData{
		Name:    "",
		Email:   "invalid-email",
		Message: "short",
	}

	jsonData, err := json.Marshal(formData)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/submit", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should return error status
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %v", status)
	}

	// Parse response
	var response models.Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Could not unmarshal response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got %s", response.Status)
	}
}

func TestSubmitHandler_JSONRequest_MessageMinimumLength(t *testing.T) {
	for _, tt := range []struct {
		name       string
		length     int
		wantStatus int
		wantError  string
	}{
		{"63 characters", 63, http.StatusBadRequest, "message must be at least 64 characters"},
		{"64 characters", 64, http.StatusOK, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			emailService := &mockEmailService{}
			handler := newTestSubmitHandler(emailService)
			body, err := json.Marshal(models.FormData{
				Name:    "John Doe",
				Email:   "john@example.com",
				Message: strings.Repeat("a", tt.length),
			})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			handler.Handle(recorder, req)

			var response models.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != tt.wantStatus || response.Error != tt.wantError {
				t.Fatalf("HTTP %d, response %+v; want HTTP %d and error %q", recorder.Code, response, tt.wantStatus, tt.wantError)
			}
			wantCalls := 0
			if tt.wantStatus == http.StatusOK {
				wantCalls = 1
			}
			if emailService.calls != wantCalls {
				t.Fatalf("email calls = %d, want %d", emailService.calls, wantCalls)
			}
		})
	}
}

func TestSubmitHandler_JSONRequest_ParseError(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Create invalid JSON
	invalidJSON := `{"name": "John", "email": }`

	req, err := http.NewRequest("POST", "/submit", strings.NewReader(invalidJSON))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should return error status
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %v", status)
	}

	// Parse response
	var response models.Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Could not unmarshal response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got %s", response.Status)
	}

	if response.Error != "failed to parse JSON" {
		t.Errorf("Expected error 'failed to parse JSON', got %s", response.Error)
	}
}

func TestSubmitHandler_IgnoresCustomRedirect(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Test with custom redirect URL
	formData := url.Values{
		"name":      {"John Doe"},
		"email":     {"john@example.com"},
		"message":   {"This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters."},
		"_redirect": {"https://example.com/thank-you"},
	}

	req, err := http.NewRequest("POST", "/submit", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://attacker.example/phishing")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// A request-supplied redirect cannot replace the configured destination.
	location := rr.Header().Get("Location")
	if location != "https://example.com/contact?source=form#form-success" {
		t.Errorf("Expected configured redirect, got %s", location)
	}
}

func TestSubmitHandler_ValidationError(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Test with invalid data (missing required fields)
	formData := url.Values{
		"name":    {""},
		"email":   {"invalid-email"},
		"message": {"short"},
	}

	req, err := http.NewRequest("POST", "/submit", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://example.com/contact")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should redirect with error status
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Expected status 303, got %v", status)
	}

	location := rr.Header().Get("Location")
	if location != "https://example.com/contact?source=form#form-error" {
		t.Errorf("Expected error redirect, got %s", location)
	}
}

func TestSubmitHandler_AjaxValidationError(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Test AJAX request with invalid data
	formData := url.Values{
		"name":    {""},
		"email":   {"invalid-email"},
		"message": {"short"},
	}

	req, err := http.NewRequest("POST", "/submit", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should return JSON error
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %v", status)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type application/json, got %s", contentType)
	}

	responseBody := rr.Body.String()
	if !strings.Contains(responseBody, "error") {
		t.Errorf("Expected response to contain error, got %s", responseBody)
	}
}

func TestSubmitHandler_MethodNotAllowed(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Test GET request (should fail)
	req, err := http.NewRequest("GET", "/submit", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should redirect with error status
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Expected status 303, got %v", status)
	}

	location := rr.Header().Get("Location")
	// The configured destination is used even without Referer.
	if location != "https://example.com/contact?source=form#form-error" {
		t.Errorf("Expected configured error redirect, got %s", location)
	}
}

func TestSubmitHandler_EmailServiceError(t *testing.T) {
	// Mock email service that fails
	emailService := &mockEmailService{shouldFail: true}
	handler := newTestSubmitHandler(emailService)

	formData := url.Values{
		"name":    {"John Doe"},
		"email":   {"john@example.com"},
		"message": {"This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters. This is a valid message with enough characters."},
	}

	req, err := http.NewRequest("POST", "/submit", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest") // AJAX mode

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should return error status
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %v", status)
	}

	// Should return JSON error
	var response models.Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Could not unmarshal response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got %s", response.Status)
	}

	if response.Error != "failed to send email" {
		t.Errorf("Expected error 'failed to send email', got %s", response.Error)
	}
}

func TestIsAjaxRequest(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name: "XMLHttpRequest header",
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
			},
			expected: true,
		},
		{
			name: "JSON content type",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expected: true,
		},
		{
			name: "JSON content type with charset",
			headers: map[string]string{
				"Content-Type": "application/json; charset=utf-8",
			},
			expected: true,
		},
		{
			name: "JSON accept header",
			headers: map[string]string{
				"Accept": "application/json",
			},
			expected: true,
		},
		{
			name: "Mixed accept header with JSON",
			headers: map[string]string{
				"Accept": "text/html,application/json,*/*",
			},
			expected: true,
		},
		{
			name: "Regular form request",
			headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			expected: false,
		},
		{
			name:     "No special headers",
			headers:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/submit", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := handler.isAjaxRequest(req)
			if result != tt.expected {
				t.Errorf("isAjaxRequest() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSubmitHandler_ParseFormError(t *testing.T) {
	emailService := &mockEmailService{}
	handler := newTestSubmitHandler(emailService)

	// Create a request with malformed form data
	req, err := http.NewRequest("POST", "/submit", strings.NewReader("%"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Should redirect with error status
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Expected status 303, got %v", status)
	}

	location := rr.Header().Get("Location")
	// The configured destination is used even without Referer.
	if location != "https://example.com/contact?source=form#form-error" {
		t.Errorf("Expected configured error redirect, got %s", location)
	}
}

func TestSubmitHandler_RateLimitsSuccessfulEmailsPerIP(t *testing.T) {
	email := &mockEmailService{}
	handler := newTestSubmitHandler(email)
	form := url.Values{
		"name":    {"John Doe"},
		"email":   {"john@example.com"},
		"message": {strings.Repeat("valid message ", 30)},
	}
	submit := func(ip string, jsonResponse bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
		req.RemoteAddr = ip + ":1234"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if jsonResponse {
			req.Header.Set("Accept", "application/json")
		}
		recorder := httptest.NewRecorder()
		handler.Handle(recorder, req)
		return recorder
	}

	for i := 0; i < 3; i++ {
		if got := submit("198.51.100.1", true).Code; got != http.StatusOK {
			t.Fatalf("submission %d got HTTP %d", i+1, got)
		}
	}
	recorder := submit("198.51.100.1", true)
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") == "" || email.calls != 3 {
		t.Fatalf("fourth submission: HTTP %d, retry-after %q, email calls %d", recorder.Code, recorder.Header().Get("Retry-After"), email.calls)
	}
	if got := submit("198.51.100.1", false).Header().Get("Location"); got != "https://example.com/contact?source=form#form-rate-limited" {
		t.Fatalf("browser submission redirected to %q", got)
	}
	if got := submit("198.51.100.2", true).Code; got != http.StatusOK || email.calls != 4 {
		t.Fatalf("second visitor got HTTP %d with %d email calls", got, email.calls)
	}
}

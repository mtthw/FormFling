package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalPort := os.Getenv("PORT")
	originalSMTPHost := os.Getenv("SMTP_HOST")
	originalOrigins := os.Getenv("ALLOWED_ORIGINS")
	originalStatusTemplate := os.Getenv("STATUS_TEMPLATE")
	originalStatusRedirect := os.Getenv("STATUS_REDIRECT_URL")
	originalRateLimitMax := os.Getenv("RATE_LIMIT_MAX_EMAILS")
	originalRateLimitWindow := os.Getenv("RATE_LIMIT_WINDOW")

	// Clean up after test
	defer func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("SMTP_HOST", originalSMTPHost)
		os.Setenv("ALLOWED_ORIGINS", originalOrigins)
		os.Setenv("STATUS_TEMPLATE", originalStatusTemplate)
		os.Setenv("STATUS_REDIRECT_URL", originalStatusRedirect)
		os.Setenv("RATE_LIMIT_MAX_EMAILS", originalRateLimitMax)
		os.Setenv("RATE_LIMIT_WINDOW", originalRateLimitWindow)
	}()

	// Test default values
	os.Unsetenv("PORT")
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("ALLOWED_ORIGINS")
	os.Unsetenv("STATUS_TEMPLATE")
	os.Unsetenv("STATUS_REDIRECT_URL")
	os.Unsetenv("RATE_LIMIT_MAX_EMAILS")
	os.Unsetenv("RATE_LIMIT_WINDOW")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", cfg.Port)
	}

	if cfg.SMTPHost != "smtp.gmail.com" {
		t.Errorf("Expected smtp.gmail.com, got %s", cfg.SMTPHost)
	}

	if cfg.SMTPPort != 587 {
		t.Errorf("Expected SMTP port 587, got %d", cfg.SMTPPort)
	}

	if cfg.StatusTemplate != "./web/templates/status_template.html" {
		t.Errorf("Expected status template './web/templates/status_template.html', got %s", cfg.StatusTemplate)
	}
	if cfg.StatusRedirectURL != "" {
		t.Errorf("Expected no default status redirect, got %s", cfg.StatusRedirectURL)
	}
	if cfg.RateLimitMaxEmails != "3" || cfg.RateLimitWindow != "24h" {
		t.Errorf("Expected default rate limit 3 per 24h, got %s per %s", cfg.RateLimitMaxEmails, cfg.RateLimitWindow)
	}

	// Test custom values
	os.Setenv("PORT", "3000")
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("ALLOWED_ORIGINS", "https://example.com,https://test.com")
	os.Setenv("STATUS_TEMPLATE", "/custom/status_template.html")
	os.Setenv("STATUS_REDIRECT_URL", "https://example.com/contact")
	os.Setenv("RATE_LIMIT_MAX_EMAILS", "5")
	os.Setenv("RATE_LIMIT_WINDOW", "12h")

	cfg = Load()

	if cfg.Port != "3000" {
		t.Errorf("Expected port 3000, got %s", cfg.Port)
	}

	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("Expected smtp.example.com, got %s", cfg.SMTPHost)
	}

	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("Expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
	}

	if cfg.AllowedOrigins[0] != "https://example.com" {
		t.Errorf("Expected https://example.com, got %s", cfg.AllowedOrigins[0])
	}

	if cfg.StatusTemplate != "/custom/status_template.html" {
		t.Errorf("Expected status template '/custom/status_template.html', got %s", cfg.StatusTemplate)
	}
	if cfg.StatusRedirectURL != "https://example.com/contact" {
		t.Errorf("Expected configured status redirect, got %s", cfg.StatusRedirectURL)
	}
	if cfg.RateLimitMaxEmails != "5" || cfg.RateLimitWindow != "12h" {
		t.Errorf("Expected configured rate limit 5 per 12h, got %s per %s", cfg.RateLimitMaxEmails, cfg.RateLimitWindow)
	}
}

func TestValidateStatusRedirectURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"", true},
		{"https://example.com/contact?from=form", true},
		{"http://localhost:3000/contact", true},
		{"javascript:alert(1)", false},
		{"//example.com/contact", false},
		{"https:example.com/contact", false},
		{"https://user@example.com/contact", false},
	}
	for _, tt := range tests {
		err := ValidateStatusRedirectURL(tt.url)
		if (err == nil) != tt.valid {
			t.Errorf("ValidateStatusRedirectURL(%q) error = %v, want valid = %v", tt.url, err, tt.valid)
		}
	}
}

func TestGetEnv(t *testing.T) {
	result := getEnv("NONEXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got %s", result)
	}

	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result = getEnv("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got %s", result)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	result := getEnvAsInt("NONEXISTENT_VAR", 123)
	if result != 123 {
		t.Errorf("Expected 123, got %d", result)
	}

	os.Setenv("TEST_INT_VAR", "456")
	defer os.Unsetenv("TEST_INT_VAR")

	result = getEnvAsInt("TEST_INT_VAR", 123)
	if result != 456 {
		t.Errorf("Expected 456, got %d", result)
	}

	// Test invalid int
	os.Setenv("TEST_INVALID_INT", "not_a_number")
	defer os.Unsetenv("TEST_INVALID_INT")

	result = getEnvAsInt("TEST_INVALID_INT", 789)
	if result != 789 {
		t.Errorf("Expected 789 (default), got %d", result)
	}
}

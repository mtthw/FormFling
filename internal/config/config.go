package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Timezone           string
	Port               string
	SMTPHost           string
	SMTPPort           int
	SMTPUsername       string
	SMTPPassword       string
	FromEmail          string
	FromName           string
	ToEmail            string
	ToName             string
	AllowedOrigins     []string
	FormTitle          string
	EmailTemplate      string
	StatusTemplate     string
	StatusRedirectURL  string
	TrustedProxyCIDRs  string
	RateLimitMaxEmails string
	RateLimitWindow    string
}

func Load() *Config {
	config := &Config{
		Timezone:           getEnv("TZ", "UTC"),
		Port:               getEnv("PORT", "8080"),
		SMTPHost:           getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:           getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername:       getEnv("SMTP_USERNAME", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
		FromEmail:          getEnv("FROM_EMAIL", ""),
		FromName:           getEnv("FROM_NAME", "FormFling"),
		ToEmail:            getEnv("TO_EMAIL", ""),
		ToName:             getEnv("TO_NAME", ""),
		FormTitle:          getEnv("FORM_TITLE", "Contact Me"),
		EmailTemplate:      getEnv("EMAIL_TEMPLATE", "./web/templates/email_template.html"),
		StatusTemplate:     getEnv("STATUS_TEMPLATE", "./web/templates/status_template.html"),
		StatusRedirectURL:  getEnv("STATUS_REDIRECT_URL", ""),
		TrustedProxyCIDRs:  getEnv("TRUSTED_PROXY_CIDRS", ""),
		RateLimitMaxEmails: getEnv("RATE_LIMIT_MAX_EMAILS", "3"),
		RateLimitWindow:    getEnv("RATE_LIMIT_WINDOW", "24h"),
	}

	// Parse allowed origins
	allowedOriginsStr := getEnv("ALLOWED_ORIGINS", "*")
	if allowedOriginsStr != "*" && allowedOriginsStr != "" {
		config.AllowedOrigins = strings.Split(allowedOriginsStr, ",")
		for i := range config.AllowedOrigins {
			config.AllowedOrigins[i] = strings.TrimSpace(config.AllowedOrigins[i])
		}
	}

	return config
}

// ValidateStatusRedirectURL rejects URL schemes that could execute script in a browser.
func ValidateStatusRedirectURL(raw string) error {
	if raw == "" {
		return nil
	}

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("STATUS_REDIRECT_URL must be an absolute http(s) URL without user info")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

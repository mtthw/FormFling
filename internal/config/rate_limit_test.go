package config

import (
	"testing"
	"time"
)

func TestParseRateLimitSettings(t *testing.T) {
	settings, err := ParseRateLimitSettings("3", "24h")
	if err != nil || settings.MaxEmails != 3 || settings.Window != 24*time.Hour {
		t.Fatalf("unexpected settings %+v, error %v", settings, err)
	}
	for _, test := range []struct{ max, window string }{
		{"0", "24h"},
		{"-1", "24h"},
		{"abc", "24h"},
		{"3", "0s"},
		{"3", "-1h"},
		{"3", "tomorrow"},
	} {
		if _, err := ParseRateLimitSettings(test.max, test.window); err == nil {
			t.Errorf("expected invalid rate limit for %q, %q", test.max, test.window)
		}
	}
}

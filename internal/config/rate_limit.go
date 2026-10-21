package config

import (
	"fmt"
	"strconv"
	"time"
)

type RateLimitSettings struct {
	MaxEmails int
	Window    time.Duration
}

func ParseRateLimitSettings(maxEmailsRaw, windowRaw string) (RateLimitSettings, error) {
	maxEmails, err := strconv.Atoi(maxEmailsRaw)
	if err != nil || maxEmails <= 0 {
		return RateLimitSettings{}, fmt.Errorf("RATE_LIMIT_MAX_EMAILS must be a positive integer")
	}
	window, err := time.ParseDuration(windowRaw)
	if err != nil || window <= 0 {
		return RateLimitSettings{}, fmt.Errorf("RATE_LIMIT_WINDOW must be a positive duration (for example, 24h)")
	}
	return RateLimitSettings{MaxEmails: maxEmails, Window: window}, nil
}

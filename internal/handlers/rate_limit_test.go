package handlers

import (
	"sync"
	"testing"
	"time"

	"formfling/internal/config"
)

func TestEmailRateLimiterRollingWindow(t *testing.T) {
	current := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	limiter := newEmailRateLimiter(config.RateLimitSettings{MaxEmails: 3, Window: 24 * time.Hour})
	limiter.now = func() time.Time { return current }

	for i := 0; i < 3; i++ {
		complete, retryAfter := limiter.reserve("visitor")
		if complete == nil || retryAfter != 0 {
			t.Fatalf("send %d should have been allowed", i+1)
		}
		complete(true)
		current = current.Add(time.Hour)
	}

	if complete, retryAfter := limiter.reserve("visitor"); complete != nil || retryAfter != 21*time.Hour {
		t.Fatalf("fourth send: complete = %v, retry after = %v", complete != nil, retryAfter)
	}
	if complete, _ := limiter.reserve("another visitor"); complete == nil {
		t.Fatal("another visitor should have an independent allowance")
	} else {
		complete(false)
	}

	current = current.Add(21 * time.Hour)
	if complete, _ := limiter.reserve("visitor"); complete == nil {
		t.Fatal("oldest send should have expired after 24 hours")
	} else {
		complete(true)
	}
}

func TestEmailRateLimiterFailedAndConcurrentSends(t *testing.T) {
	limiter := newEmailRateLimiter(config.RateLimitSettings{MaxEmails: 3, Window: 24 * time.Hour})
	var wg sync.WaitGroup
	results := make(chan func(bool), 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			complete, _ := limiter.reserve("visitor")
			if complete != nil {
				results <- complete
			}
		}()
	}
	wg.Wait()
	close(results)
	var completions []func(bool)
	for complete := range results {
		completions = append(completions, complete)
	}
	if len(completions) != 3 {
		t.Fatalf("expected 3 reservations, got %d", len(completions))
	}

	completions[0](false)
	if complete, _ := limiter.reserve("visitor"); complete == nil {
		t.Fatal("failed send should release its slot")
	} else {
		complete(true)
	}
	for _, complete := range completions[1:] {
		complete(true)
	}
	if complete, _ := limiter.reserve("visitor"); complete != nil {
		t.Fatal("three successful sends should fill the window")
	}
}

func TestEmailRateLimiterCustomSettings(t *testing.T) {
	current := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	limiter := newEmailRateLimiter(config.RateLimitSettings{MaxEmails: 2, Window: time.Hour})
	limiter.now = func() time.Time { return current }
	for i := 0; i < 2; i++ {
		complete, _ := limiter.reserve("visitor")
		if complete == nil {
			t.Fatal("send should be allowed")
		}
		complete(true)
	}
	if complete, retryAfter := limiter.reserve("visitor"); complete != nil || retryAfter != time.Hour {
		t.Fatalf("third send: allowed = %v, retry after = %v", complete != nil, retryAfter)
	}
	current = current.Add(time.Hour)
	if complete, _ := limiter.reserve("visitor"); complete == nil {
		t.Fatal("custom window should have expired")
	} else {
		complete(false)
	}
}

package handlers

import (
	"sync"
	"time"

	"formfling/internal/config"
)

type emailLimitEntry struct {
	sent     []time.Time
	inFlight int
}

// emailRateLimiter counts successful sends and reserves slots for sends in progress.
type emailRateLimiter struct {
	mu        sync.Mutex
	entries   map[string]*emailLimitEntry
	now       func() time.Time
	lastSweep time.Time
	settings  config.RateLimitSettings
}

func newEmailRateLimiter(settings config.RateLimitSettings) *emailRateLimiter {
	return &emailRateLimiter{
		entries:  make(map[string]*emailLimitEntry),
		now:      time.Now,
		settings: settings,
	}
}

// reserve returns a completion function, or the time until another slot may open.
func (l *emailRateLimiter) reserve(key string) (func(bool), time.Duration) {
	l.mu.Lock()
	now := l.now()
	if now.Sub(l.lastSweep) >= time.Hour {
		for existingKey, existingEntry := range l.entries {
			pruneSent(existingEntry, now, l.settings.Window)
			if len(existingEntry.sent) == 0 && existingEntry.inFlight == 0 {
				delete(l.entries, existingKey)
			}
		}
		l.lastSweep = now
	}
	entry := l.entries[key]
	if entry == nil {
		entry = &emailLimitEntry{}
		l.entries[key] = entry
	}

	pruneSent(entry, now, l.settings.Window)

	if len(entry.sent)+entry.inFlight >= l.settings.MaxEmails {
		retryAfter := time.Second
		if len(entry.sent) == l.settings.MaxEmails {
			oldest := entry.sent[0]
			for _, sentAt := range entry.sent[1:] {
				if sentAt.Before(oldest) {
					oldest = sentAt
				}
			}
			retryAfter = oldest.Add(l.settings.Window).Sub(now)
		}
		l.mu.Unlock()
		return nil, retryAfter
	}
	entry.inFlight++
	l.mu.Unlock()

	var once sync.Once
	return func(sent bool) {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			entry.inFlight--
			if sent {
				entry.sent = append(entry.sent, l.now())
			} else if len(entry.sent) == 0 && entry.inFlight == 0 {
				delete(l.entries, key)
			}
		})
	}, 0
}

func pruneSent(entry *emailLimitEntry, now time.Time, window time.Duration) {
	kept := entry.sent[:0]
	for _, sentAt := range entry.sent {
		if now.Sub(sentAt) < window {
			kept = append(kept, sentAt)
		}
	}
	entry.sent = kept
}

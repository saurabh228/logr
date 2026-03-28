package suppress

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/saurabh/logr/internal/parser"
)

var digitRe = regexp.MustCompile(`\d+`)

// normalizeMessage replaces all digit sequences with "#" so that
// "processed 42 items" and "processed 99 items" produce the same fingerprint.
func normalizeMessage(msg string) string {
	return digitRe.ReplaceAllString(msg, "#")
}

// fingerprint returns a stable hash for the entry based on its
// normalised message, service, and level.
func fingerprint(entry parser.LogEntry) string {
	norm := normalizeMessage(entry.Message)
	raw := fmt.Sprintf("%s|%s|%s", norm, entry.Service, entry.Level)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:8])
}

// Suppressor deduplicates repeated log patterns within a time window.
type Suppressor struct {
	ttl      time.Duration
	mu       sync.Mutex
	lastSeen map[string]time.Time
}

// New creates a Suppressor with the given TTL.
// A TTL of zero disables suppression (ShouldSuppress always returns false).
func New(ttl time.Duration) *Suppressor {
	return &Suppressor{
		ttl:      ttl,
		lastSeen: make(map[string]time.Time),
	}
}

// ShouldSuppress returns true if an entry with the same fingerprint was seen
// within the configured TTL window, meaning this entry should be dropped.
func (s *Suppressor) ShouldSuppress(entry parser.LogEntry) bool {
	if s.ttl == 0 {
		return false
	}

	fp := fingerprint(entry)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if last, ok := s.lastSeen[fp]; ok {
		if now.Sub(last) < s.ttl {
			return true
		}
	}

	s.lastSeen[fp] = now
	return false
}

// Purge removes fingerprints older than the TTL to prevent unbounded growth.
func (s *Suppressor) Purge() {
	if s.ttl == 0 {
		return
	}

	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for fp, t := range s.lastSeen {
		if now.Sub(t) >= s.ttl {
			delete(s.lastSeen, fp)
		}
	}
}

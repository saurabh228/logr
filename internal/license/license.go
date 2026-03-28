package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	gumroadVerifyURL = "https://api.gumroad.com/v2/licenses/verify"
	productPermalink = "logr"
	cacheTTL         = 30 * 24 * time.Hour // 30 days
)

// cacheEntry is persisted to ~/.logr/license as JSON.
type cacheEntry struct {
	Key        string    `json:"key"`
	VerifiedAt time.Time `json:"verified_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// cachePath returns the path to the license cache file.
func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".logr")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create logr directory: %w", err)
	}
	return filepath.Join(dir, "license"), nil
}

// loadCache reads and parses the cached license entry.
func loadCache() (*cacheEntry, error) {
	p, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// saveCache writes a cacheEntry to disk.
func saveCache(entry cacheEntry) error {
	p, err := cachePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// gumroadResponse is the minimal subset of the Gumroad verify response we care about.
type gumroadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// callGumroad contacts the Gumroad API to verify a license key.
func callGumroad(key string) error {
	resp, err := http.PostForm(gumroadVerifyURL, url.Values{
		"product_permalink": {productPermalink},
		"license_key":       {key},
	})
	if err != nil {
		return fmt.Errorf("license verification request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cannot read verification response: %w", err)
	}

	var gr gumroadResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return fmt.Errorf("cannot parse verification response: %w", err)
	}

	if !gr.Success {
		msg := gr.Message
		if msg == "" {
			msg = "license verification failed"
		}
		return errors.New(msg)
	}
	return nil
}

// Verify checks that key is a valid logr license.
//
// If the LOGR_DEV=1 environment variable is set, verification is skipped.
// A successful verification is cached for 30 days so repeated runs do not
// require a network round-trip.
func Verify(key string) error {
	// Development bypass.
	if os.Getenv("LOGR_DEV") == "1" {
		return nil
	}

	if key == "" {
		return errors.New("no license key provided — run: logr license <your-key>")
	}

	// Check the cache first.
	cache, err := loadCache()
	if err == nil && cache != nil && cache.Key == key && time.Now().Before(cache.ExpiresAt) {
		return nil
	}

	// Cache miss or expired — call Gumroad.
	if err := callGumroad(key); err != nil {
		return err
	}

	now := time.Now()
	_ = saveCache(cacheEntry{
		Key:        key,
		VerifiedAt: now,
		ExpiresAt:  now.Add(cacheTTL),
	})

	return nil
}

// KeyFromCache returns the license key stored in the local cache (if any).
func KeyFromCache() string {
	cache, err := loadCache()
	if err != nil || cache == nil {
		return ""
	}
	return cache.Key
}

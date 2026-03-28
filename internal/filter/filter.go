package filter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/saurabh/logr/internal/hier"
	"github.com/saurabh/logr/internal/parser"
	"github.com/saurabh/logr/internal/suppress"
)

// levelOrder maps level names to a numeric rank for comparison.
var levelOrder = map[string]int{
	"trace":   0,
	"debug":   0,
	"info":    1,
	"warn":    2,
	"warning": 2,
	"error":   3,
	"err":     3,
	"fatal":   4,
	"panic":   4,
}

// Config holds all filter settings.
type Config struct {
	MinLevel      string        `toml:"min_level"`
	IncludeFields []string      `toml:"include_fields"`
	ExcludeFields []string      `toml:"exclude_fields"`
	HierPatterns  []string      `toml:"hier_patterns"`
	SuppressTTL   time.Duration `toml:"suppress_ttl"`
	Services      []string      `toml:"services"`
	Keys          []string      `toml:"keys"` // output key projection
}

// Engine applies a Config to log entries.
type Engine struct {
	cfg        Config
	suppressor *suppress.Suppressor
}

// New creates a new filter Engine.
func New(cfg Config) *Engine {
	return &Engine{
		cfg:        cfg,
		suppressor: suppress.New(cfg.SuppressTTL),
	}
}

// Pass returns true if the entry should be included in the output.
func (e *Engine) Pass(entry parser.LogEntry) bool {
	// Raw entries bypass level/field/hier filters.
	if entry.Level == "raw" {
		return true
	}

	// Stage 1: Level filter.
	if e.cfg.MinLevel != "" {
		minRank, ok := levelOrder[strings.ToLower(e.cfg.MinLevel)]
		if !ok {
			minRank = 0
		}
		entryRank, ok := levelOrder[strings.ToLower(entry.Level)]
		if !ok {
			entryRank = 1
		}
		if entryRank < minRank {
			return false
		}
	}

	// Stage 2: Service filter.
	if len(e.cfg.Services) > 0 {
		matched := false
		for _, svc := range e.cfg.Services {
			if strings.EqualFold(strings.TrimSpace(svc), entry.Service) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Stage 3: Field includes — entry must match at least one pattern.
	if len(e.cfg.IncludeFields) > 0 {
		if !matchesAnyField(e.cfg.IncludeFields, entry) {
			return false
		}
	}

	// Stage 4: Field excludes — entry must not match any pattern.
	if len(e.cfg.ExcludeFields) > 0 {
		if matchesAnyField(e.cfg.ExcludeFields, entry) {
			return false
		}
	}

	// Stage 5: Hier filter — entry must match at least one pattern.
	if len(e.cfg.HierPatterns) > 0 {
		if entry.HierPath == "" {
			return false
		}
		if !hier.MatchAny(e.cfg.HierPatterns, entry.HierPath) {
			return false
		}
	}

	// Stage 6: TTL suppression.
	if e.suppressor.ShouldSuppress(entry) {
		return false
	}

	return true
}

// matchesAnyField checks whether a log entry matches any of the given
// field patterns ("field=value" or "field=*").
func matchesAnyField(patterns []string, entry parser.LogEntry) bool {
	for _, pat := range patterns {
		if matchField(pat, entry) {
			return true
		}
	}
	return false
}

// matchField evaluates a single "field=value" pattern against an entry.
func matchField(pattern string, entry parser.LogEntry) bool {
	idx := strings.Index(pattern, "=")
	if idx < 0 {
		// Plain field existence check.
		_, ok := entry.Fields[pattern]
		return ok
	}

	fieldName := pattern[:idx]
	wantVal := pattern[idx+1:]

	var actualVal string
	switch strings.ToLower(fieldName) {
	case "level", "lvl", "severity":
		actualVal = entry.Level
	case "service", "svc", "app", "component":
		actualVal = entry.Service
	case "msg", "message", "event":
		actualVal = entry.Message
	case "hier_path":
		actualVal = entry.HierPath
	default:
		v, ok := entry.Fields[fieldName]
		if !ok {
			return false
		}
		actualVal = valueToString(v)
	}

	if wantVal == "*" {
		return actualVal != ""
	}
	return strings.EqualFold(actualVal, wantVal)
}

// valueToString converts an arbitrary JSON-decoded value to a string.
func valueToString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return fmt.Sprintf("%g", s)
	case bool:
		return fmt.Sprintf("%t", s)
	case json.Number:
		return s.String()
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

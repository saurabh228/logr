package parser

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// LogEntry is the central struct representing a parsed log line.
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Service   string
	HierPath  string         // dot-path like "payment.txn.charge"
	Fields    map[string]any
	Raw       []byte
}

var timestampKeys = []string{"timestamp", "ts", "time", "@timestamp"}
var levelKeys = []string{"level", "lvl", "severity"}
var messageKeys = []string{"msg", "message", "event"}
var serviceKeys = []string{"service", "svc", "app", "component"}

// hierKeys deliberately does NOT include "path" (commonly an HTTP request path)
// or "trace" (commonly a trace/correlation ID). Only "hier_path" is reserved.
// Users can specify a custom field via ParseOptions.HierField.
var hierKeys = []string{"hier_path"}

// Options allows callers to override which JSON field names are used for
// the service name and hier path. Empty strings mean "use defaults".
type Options struct {
	ServiceField string // e.g. "name", "logger", "app_name"
	HierField    string // e.g. "path", "trace", "module"
}

// Parse parses a single log line using default field name detection.
func Parse(line []byte) LogEntry {
	return ParseWith(line, Options{})
}

// ParseWith parses a single log line with custom field name overrides.
func ParseWith(line []byte, opts Options) LogEntry {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return LogEntry{Raw: line, Level: "raw"}
	}

	var rawMap map[string]any
	if err := json.Unmarshal([]byte(trimmed), &rawMap); err != nil {
		return LogEntry{Raw: line, Level: "raw"}
	}

	entry := LogEntry{
		Fields: make(map[string]any),
		Raw:    line,
	}

	consumed := make(map[string]bool)

	// Extract timestamp.
	for _, k := range timestampKeys {
		if v, ok := rawMap[k]; ok {
			entry.Timestamp = parseTime(v)
			consumed[k] = true
			break
		}
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Extract level.
	for _, k := range levelKeys {
		if v, ok := rawMap[k]; ok {
			entry.Level = strings.ToLower(anyToString(v))
			consumed[k] = true
			break
		}
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	// Normalise numeric levels (pino: 10=trace,20=debug,30=info,40=warn,50=error,60=fatal).
	entry.Level = normalizeLevel(entry.Level)

	// Extract message.
	for _, k := range messageKeys {
		if v, ok := rawMap[k]; ok {
			entry.Message = anyToString(v)
			consumed[k] = true
			break
		}
	}

	// Extract service — check user-specified field first, then defaults.
	svcKeys := serviceKeys
	if opts.ServiceField != "" {
		svcKeys = append([]string{opts.ServiceField}, serviceKeys...)
	}
	for _, k := range svcKeys {
		if v, ok := rawMap[k]; ok {
			entry.Service = anyToString(v)
			consumed[k] = true
			break
		}
	}

	// Extract hier path — check user-specified field first, then defaults.
	hKeys := hierKeys
	if opts.HierField != "" {
		hKeys = append([]string{opts.HierField}, hierKeys...)
	}
	for _, k := range hKeys {
		if v, ok := rawMap[k]; ok {
			entry.HierPath = anyToString(v)
			consumed[k] = true
			break
		}
	}

	// Everything else goes to Fields.
	for k, v := range rawMap {
		if !consumed[k] {
			entry.Fields[k] = v
		}
	}

	return entry
}

// normalizeLevel converts numeric pino-style levels to their string names.
func normalizeLevel(level string) string {
	switch level {
	case "10":
		return "trace"
	case "20":
		return "debug"
	case "30":
		return "info"
	case "40":
		return "warn"
	case "50":
		return "error"
	case "60":
		return "fatal"
	}
	return level
}

// anyToString converts a JSON scalar to a string.
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case bool:
		if s {
			return "true"
		}
		return "false"
	case json.Number:
		return s.String()
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// parseTime converts a JSON value to a time.Time.
// Supported formats: RFC3339 string, Unix int64, Unix float64.
func parseTime(v any) time.Time {
	switch val := v.(type) {
	case string:
		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.999999999",
			"2006-01-02 15:04:05",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, val); err == nil {
				return t
			}
		}
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return time.Unix(i, 0)
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			sec := int64(f)
			nsec := int64((f - float64(sec)) * 1e9)
			return time.Unix(sec, nsec)
		}
	case float64:
		sec := int64(val)
		nsec := int64((val - float64(sec)) * 1e9)
		return time.Unix(sec, nsec)
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return time.Unix(i, 0)
		}
		if f, err := val.Float64(); err == nil {
			sec := int64(f)
			nsec := int64((f - float64(sec)) * 1e9)
			return time.Unix(sec, nsec)
		}
	}
	return time.Time{}
}

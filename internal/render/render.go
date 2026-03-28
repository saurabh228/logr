package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/saurabh/logr/internal/parser"
)

// levelColors maps log level names to color attributes.
var levelColors = map[string]*color.Color{
	"trace": color.New(color.FgHiBlack),
	"debug": color.New(color.FgCyan),
	"info":  color.New(color.FgGreen),
	"warn":  color.New(color.FgYellow),
	"warning": color.New(color.FgYellow),
	"error": color.New(color.FgRed),
	"err":   color.New(color.FgRed),
	"fatal": color.New(color.FgMagenta, color.Bold),
	"panic": color.New(color.FgMagenta, color.Bold),
}

var (
	dimColor       = color.New(color.Faint)
	timestampColor = color.New(color.FgHiBlack)
	serviceColor   = color.New(color.FgBlue, color.Bold)
	fieldKeyColor  = color.New(color.FgHiBlack)
)

// Options controls rendering behaviour.
type Options struct {
	NoColor bool
	JSON    bool
	Keys    []string // if set, only these extra field keys are shown
}

// wantedKeys returns a set of the requested keys (lowercased) for O(1) lookup.
// An empty set means "show all".
func wantedKeys(keys []string) map[string]bool {
	if len(keys) == 0 {
		return nil
	}
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[strings.ToLower(k)] = true
	}
	return m
}

// Render writes a human-readable (or JSON) representation of entry to w.
func Render(entry parser.LogEntry, w io.Writer, opts Options) {
	if opts.NoColor {
		color.NoColor = true
	}

	if opts.JSON {
		renderJSON(entry, w, opts)
		return
	}

	if entry.Level == "raw" {
		dimColor.Fprintln(w, strings.TrimRight(string(entry.Raw), "\n"))
		return
	}

	renderPretty(entry, w, opts)
}

// renderPretty writes a colorised single-line representation.
//
// Format: [TS] LEVEL  service-name  message  key=val · key=val  #hier.path
func renderPretty(entry parser.LogEntry, w io.Writer, opts Options) {
	// Timestamp — dim grey in brackets.
	ts := timestampColor.Sprintf("[%s]", entry.Timestamp.Format("15:04:05.000"))

	// Level — 5 chars wide, colored.
	levelStr := strings.ToUpper(entry.Level)
	c, ok := levelColors[strings.ToLower(entry.Level)]
	if !ok {
		c = color.New(color.Reset)
	}
	levelPadded := c.Sprintf("%-5s", levelStr)

	// Service — bold blue, capped at 20 chars to keep lines tight.
	svc := ""
	if entry.Service != "" {
		name := entry.Service
		if len(name) > 20 {
			name = name[:19] + "…"
		}
		svc = serviceColor.Sprintf("  %-20s", name)
	}

	// Message — level color.
	msg := c.Sprint(entry.Message)

	// Fields — sorted, separated by dim "·", keys dimmed, values plain.
	wanted := wantedKeys(opts.Keys)
	var fieldParts []string
	fieldKeys := make([]string, 0, len(entry.Fields))
	for k := range entry.Fields {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)
	for _, k := range fieldKeys {
		if wanted != nil && !wanted[strings.ToLower(k)] {
			continue
		}
		v := entry.Fields[k]
		fieldParts = append(fieldParts, fmt.Sprintf("%s=%s",
			fieldKeyColor.Sprint(k),
			formatFieldValue(v),
		))
	}
	fieldsStr := ""
	if len(fieldParts) > 0 {
		sep := dimColor.Sprint(" · ")
		fieldsStr = "  " + strings.Join(fieldParts, sep)
	}

	// HierPath — moved to end as a dim tag so it doesn't break flow.
	hierStr := ""
	if entry.HierPath != "" {
		hierStr = dimColor.Sprintf("  #%s", entry.HierPath)
	}

	fmt.Fprintf(w, "%s %s%s  %s%s%s\n", ts, levelPadded, svc, msg, fieldsStr, hierStr)
}

// renderJSON writes the entry as compact JSON, respecting opts.Keys if set.
func renderJSON(entry parser.LogEntry, w io.Writer, opts Options) {
	wanted := wantedKeys(opts.Keys)

	include := func(key string) bool {
		return wanted == nil || wanted[key]
	}

	out := map[string]any{}
	if include("timestamp") || include("ts") || include("time") {
		out["timestamp"] = entry.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")
	}
	if include("level") || include("lvl") {
		out["level"] = entry.Level
	}
	if include("message") || include("msg") || include("event") {
		out["message"] = entry.Message
	}
	if entry.Service != "" && (include("service") || include("svc")) {
		out["service"] = entry.Service
	}
	if entry.HierPath != "" && (include("hier_path") || include("hier") || include("path")) {
		out["hier_path"] = entry.HierPath
	}
	for k, v := range entry.Fields {
		if include(strings.ToLower(k)) {
			out[k] = v
		}
	}
	if entry.Level == "raw" {
		out["raw"] = strings.TrimRight(string(entry.Raw), "\n")
	}
	b, _ := json.Marshal(out)
	fmt.Fprintln(w, string(b))
}

// formatFieldValue formats a field value for terminal display.
func formatFieldValue(v any) string {
	if v == nil {
		return "<nil>"
	}
	switch s := v.(type) {
	case string:
		if strings.ContainsAny(s, " \t") {
			return fmt.Sprintf("%q", s)
		}
		return s
	case float64:
		return fmt.Sprintf("%g", s)
	case bool:
		return fmt.Sprintf("%t", s)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

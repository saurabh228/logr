package hier

import "strings"

// Match reports whether the dot-separated path matches the given pattern.
//
// Pattern rules:
//   - "*"  matches exactly one segment
//   - "**" matches zero or more segments (greedy)
//
// Examples:
//
//	Match("payment.*",  "payment.charge")        → true
//	Match("payment.*",  "payment.charge.retry")  → false
//	Match("payment.**", "payment.charge.retry")  → true
//	Match("payment.**", "payment")               → true  (** matches zero)
func Match(pattern, path string) bool {
	patParts := splitDot(pattern)
	pathParts := splitDot(path)
	return matchParts(patParts, pathParts)
}

// MatchAny reports whether path matches any of the given patterns.
func MatchAny(patterns []string, path string) bool {
	for _, p := range patterns {
		if Match(p, path) {
			return true
		}
	}
	return false
}

func splitDot(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}

// matchParts is a recursive matcher over pre-split segments.
func matchParts(pat, path []string) bool {
	// Both exhausted → match.
	if len(pat) == 0 && len(path) == 0 {
		return true
	}

	// Pattern exhausted but path still has segments → no match.
	if len(pat) == 0 {
		return false
	}

	// ** can consume zero or more path segments.
	if pat[0] == "**" {
		// Try consuming zero path segments (skip ** in pattern).
		if matchParts(pat[1:], path) {
			return true
		}
		// Try consuming one or more path segments.
		for i := 1; i <= len(path); i++ {
			if matchParts(pat[1:], path[i:]) {
				return true
			}
		}
		return false
	}

	// Path is exhausted but pattern still has non-** tokens → no match.
	if len(path) == 0 {
		return false
	}

	// * matches exactly one segment.
	if pat[0] == "*" || pat[0] == path[0] {
		return matchParts(pat[1:], path[1:])
	}

	return false
}

package handler

import (
	"net/http"
	"strings"
)

// searchQuery returns the trimmed, lower-cased ?q= parameter ("" means no search).
func searchQuery(r *http.Request) string {
	return strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
}

// matchesAny reports whether q (lower-cased) is a substring of any field, ignoring case.
// An empty q matches everything.
func matchesAny(q string, fields ...string) bool {
	if q == "" {
		return true
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

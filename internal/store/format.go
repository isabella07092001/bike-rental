package store

import "time"

// formatTime formats a time for error messages in the shop's local time.
func formatTime(t time.Time) string {
	return t.Local().Format("02.01.2006 15:04")
}

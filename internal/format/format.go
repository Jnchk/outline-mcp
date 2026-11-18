package format

import (
	"strings"
	"time"
)

// Row renders one pipe-separated record; empty cells become "-".
func Row(cells ...string) string {
	for i, c := range cells {
		c = strings.Join(strings.Fields(c), " ")
		if c == "" {
			c = "-"
		}
		cells[i] = c
	}
	return strings.Join(cells, " | ")
}

func Truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// Clip keeps newlines and says how much was dropped.
func Clip(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "\n… (truncated, " + itoa(len(r)) + " chars total)"
}

// When renders an ISO timestamp as UTC without seconds.
func When(s string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format("2006-01-02 15:04Z")
		}
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

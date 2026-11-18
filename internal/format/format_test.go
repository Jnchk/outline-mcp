package format

import (
	"strings"
	"testing"
)

func TestRow(t *testing.T) {
	if got := Row("a", "", "c"); got != "a | - | c" {
		t.Fatalf("Row = %q", got)
	}
	if got := Row("a\nb", "c"); got != "a b | c" {
		t.Fatalf("Row newline = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("登录失败在单点登录", 4); got != "登录失败…" {
		t.Fatalf("Truncate = %q", got)
	}
}

func TestClip(t *testing.T) {
	long := strings.Repeat("字", 30)
	got := Clip(long, 5)
	if !strings.HasPrefix(got, "字字字字字") || !strings.Contains(got, "30 chars total") {
		t.Fatalf("Clip = %q", got)
	}
	if got := Clip("short", 10); got != "short" {
		t.Fatalf("Clip short = %q", got)
	}
}

func TestWhen(t *testing.T) {
	if got := When("2026-11-18T01:00:00Z"); got != "2026-11-18 01:00Z" {
		t.Fatalf("When = %q", got)
	}
	if got := When("junk"); got != "junk" {
		t.Fatalf("When junk = %q", got)
	}
}

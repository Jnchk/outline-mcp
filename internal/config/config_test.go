package config

import (
	"strings"
	"testing"
	"time"
)

func cleanEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"OUTLINE_URL", "OUTLINE_TOKEN", "OUTLINE_INSECURE", "OUTLINE_READ_ONLY",
		"OUTLINE_TIMEOUT", "OUTLINE_DEFAULT_LIMIT", "OUTLINE_MAX_UPLOAD",
		"MCP_TRANSPORT", "MCP_ADDR", "MCP_PATH", "MCP_TOKEN",
	} {
		t.Setenv(k, "")
	}
	t.Setenv("OUTLINE_URL", "https://outline.example.com")
	t.Setenv("OUTLINE_TOKEN", "secret")
}

func TestFromEnvDefaults(t *testing.T) {
	cleanEnv(t)
	c, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != "127.0.0.1:8940" || c.Path != "/mcp" || c.DefaultLimit != 25 {
		t.Fatalf("defaults: %#v", c)
	}
	if c.Timeout != 45*time.Second || c.MaxUpload != 10<<20 {
		t.Fatalf("timeouts: %#v", c)
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"missing token", map[string]string{"OUTLINE_TOKEN": ""}, "OUTLINE_TOKEN"},
		{"missing url", map[string]string{"OUTLINE_URL": ""}, "OUTLINE_URL"},
		{"bad url", map[string]string{"OUTLINE_URL": "ftp://h"}, "http(s)"},
		{"bad transport", map[string]string{"MCP_TRANSPORT": "sse"}, "MCP_TRANSPORT"},
		{"non-loopback without token", map[string]string{"MCP_TRANSPORT": "http", "MCP_ADDR": "0.0.0.0:1"}, "MCP_TOKEN"},
		{"non-loopback with token", map[string]string{"MCP_TRANSPORT": "http", "MCP_ADDR": "0.0.0.0:1", "MCP_TOKEN": "t"}, ""},
		{"path without slash", map[string]string{"MCP_TRANSPORT": "http", "MCP_PATH": "mcp"}, "MCP_PATH"},
		{"bad limit", map[string]string{"OUTLINE_DEFAULT_LIMIT": "101"}, "OUTLINE_DEFAULT_LIMIT"},
		{"bad timeout", map[string]string{"OUTLINE_TIMEOUT": "45"}, "OUTLINE_TIMEOUT"},
		{"bad upload size", map[string]string{"OUTLINE_MAX_UPLOAD": "0"}, "OUTLINE_MAX_UPLOAD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleanEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			_, err := FromEnv()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"OUTLINE_INSECURE":      "notabool",
		"OUTLINE_READ_ONLY":     "notabool",
		"OUTLINE_TIMEOUT":       "45",
		"OUTLINE_DEFAULT_LIMIT": "x",
		"OUTLINE_MAX_UPLOAD":    "x",
	}
	for key, value := range cases {
		t.Run(key, func(t *testing.T) {
			cleanEnv(t)
			t.Setenv(key, value)
			if _, err := FromEnv(); err == nil {
				t.Fatalf("%s=%q accepted", key, value)
			}
		})
	}
}

func TestFlagsAndPlainHTTP(t *testing.T) {
	cleanEnv(t)
	t.Setenv("OUTLINE_URL", "http://h")
	t.Setenv("OUTLINE_INSECURE", "true")
	t.Setenv("OUTLINE_READ_ONLY", "true")
	c, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if !c.IsPlainHTTP() || !c.Insecure || !c.ReadOnly {
		t.Fatalf("flags: %#v", c)
	}
}

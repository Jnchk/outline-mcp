package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	URL          string
	Token        string
	Insecure     bool
	ReadOnly     bool
	Timeout      time.Duration
	DefaultLimit int
	MaxUpload    int64

	Transport string
	Addr      string
	Path      string
	MCPToken  string
}

func FromEnv() (*Config, error) {
	c := &Config{
		URL:       strings.TrimRight(os.Getenv("OUTLINE_URL"), "/"),
		Token:     os.Getenv("OUTLINE_TOKEN"),
		Transport: env("MCP_TRANSPORT", "stdio"),
		Addr:      env("MCP_ADDR", "127.0.0.1:8940"),
		Path:      env("MCP_PATH", "/mcp"),
		MCPToken:  os.Getenv("MCP_TOKEN"),
	}
	var err error
	if c.Insecure, err = envBool("OUTLINE_INSECURE"); err != nil {
		return nil, err
	}
	if c.ReadOnly, err = envBool("OUTLINE_READ_ONLY"); err != nil {
		return nil, err
	}
	if c.Timeout, err = envDur("OUTLINE_TIMEOUT", 45*time.Second); err != nil {
		return nil, err
	}
	if c.DefaultLimit, err = envInt("OUTLINE_DEFAULT_LIMIT", 25); err != nil {
		return nil, err
	}
	n, err := envInt("OUTLINE_MAX_UPLOAD", 10<<20)
	if err != nil {
		return nil, err
	}
	c.MaxUpload = int64(n)
	return c, c.Validate()
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("OUTLINE_URL is required")
	}
	u, err := url.Parse(c.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("OUTLINE_URL must be an absolute http(s) URL")
	}
	if c.Token == "" {
		return fmt.Errorf("OUTLINE_TOKEN is required")
	}
	if c.DefaultLimit < 1 || c.DefaultLimit > 100 {
		return fmt.Errorf("OUTLINE_DEFAULT_LIMIT must be 1..100")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("OUTLINE_TIMEOUT must be positive")
	}
	if c.MaxUpload <= 0 {
		return fmt.Errorf("OUTLINE_MAX_UPLOAD must be positive")
	}
	switch c.Transport {
	case "stdio":
	case "http":
		if !strings.HasPrefix(c.Path, "/") {
			return fmt.Errorf("MCP_PATH must start with /")
		}
		if c.MCPToken == "" && !loopback(c.Addr) {
			return fmt.Errorf("MCP_ADDR %q is not loopback: set MCP_TOKEN", c.Addr)
		}
	default:
		return fmt.Errorf("MCP_TRANSPORT must be stdio or http, got %q", c.Transport)
	}
	return nil
}

func (c *Config) IsPlainHTTP() bool { return strings.HasPrefix(c.URL, "http://") }

func loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return b, nil
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}

func envDur(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

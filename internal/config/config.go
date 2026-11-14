package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration for Huginn.
type Config struct {
	Concurrency     int
	ExtractionDepth int
	Lang            string
	RequestPaceMS   int
	ExitIdleAfter   time.Duration
	HTTPAddr        string
	DataDir         string
	Debug           bool
	Proxies         []string
	DefaultOutput   string
	DefaultFormat   string
}

// LoadFromEnv loads configuration with sensible defaults.
func LoadFromEnv() Config {
	cfg := Config{
		Concurrency:     envInt("HUGINN_CONCURRENCY", 1),
		ExtractionDepth: envInt("HUGINN_DEPTH", 10),
		Lang:            envString("HUGINN_LANG", "en"),
		RequestPaceMS:   envInt("HUGINN_REQUEST_PACE_MS", 500),
		ExitIdleAfter:   envDuration("HUGINN_EXIT_IDLE", 2*time.Minute),
		HTTPAddr:        envString("HUGINN_HTTP_ADDR", ":8080"),
		DataDir:         envString("HUGINN_DATA_DIR", "./data"),
		Debug:           envBool("HUGINN_DEBUG", false),
		DefaultOutput:   envString("HUGINN_OUTPUT", "leads.json"),
		DefaultFormat:   envString("HUGINN_FORMAT", "json"),
	}
	if p := os.Getenv("HUGINN_PROXIES"); p != "" {
		cfg.Proxies = splitCSV(p)
	}
	return cfg
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

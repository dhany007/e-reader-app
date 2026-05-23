package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	DeepSeekAPIKey    string
	DeepSeekModel     string
	SessionSecret     string
	AdminUsername     string
	AdminPasswordHash string
	Port              string
	DataDir           string
	StorageDir        string
	PythonBin         string
	ParserScript      string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DeepSeekAPIKey:    os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekModel:     envOr("DEEPSEEK_MODEL", "deepseek-chat"),
		SessionSecret:     os.Getenv("SESSION_SECRET"),
		AdminUsername:     os.Getenv("ADMIN_USERNAME"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
		Port:              envOr("PORT", "8080"),
		DataDir:           envOr("DATA_DIR", "./data"),
		StorageDir:        envOr("STORAGE_DIR", "./storage"),
		PythonBin:         envOr("PYTHON_BIN", "python3"),
		ParserScript:      envOr("PARSER_SCRIPT", "parser/extract.py"),
	}

	if !filepath.IsAbs(cfg.ParserScript) {
		if abs, err := filepath.Abs(cfg.ParserScript); err == nil {
			cfg.ParserScript = abs
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DEEPSEEK_API_KEY":    c.DeepSeekAPIKey,
		"SESSION_SECRET":      c.SessionSecret,
		"ADMIN_USERNAME":      c.AdminUsername,
		"ADMIN_PASSWORD_HASH": c.AdminPasswordHash,
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("missing required env var: %s", key)
		}
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

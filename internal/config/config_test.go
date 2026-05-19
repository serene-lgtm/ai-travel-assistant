package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigPrefersConfigAPIKey(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "env-key")

	cfg, err := LoadConfig(writeTestConfig(t, "config-key"))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Deepseek.APIKey != "config-key" {
		t.Fatalf("Deepseek.APIKey = %q, want %q", cfg.Deepseek.APIKey, "config-key")
	}
}

func TestLoadConfigFallsBackToEnvWhenAPIKeyEmpty(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "env-key")

	cfg, err := LoadConfig(writeTestConfig(t, ""))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Deepseek.APIKey != "env-key" {
		t.Fatalf("Deepseek.APIKey = %q, want %q", cfg.Deepseek.APIKey, "env-key")
	}
}

func TestLoadConfigFallsBackToEnvWhenAPIKeyPlaceholder(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "env-key")

	cfg, err := LoadConfig(writeTestConfig(t, deepseekAPIKeyPlaceholder))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Deepseek.APIKey != "env-key" {
		t.Fatalf("Deepseek.APIKey = %q, want %q", cfg.Deepseek.APIKey, "env-key")
	}
}

func TestLoadConfigFallsBackToEnvWhenAPIKeyWhitespace(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "env-key")

	cfg, err := LoadConfig(writeTestConfig(t, "   "))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Deepseek.APIKey != "env-key" {
		t.Fatalf("Deepseek.APIKey = %q, want %q", cfg.Deepseek.APIKey, "env-key")
	}
}

func TestLoadConfigReturnsErrorWhenNoAPIKeyProvided(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "")

	_, err := LoadConfig(writeTestConfig(t, ""))
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want non-nil")
	}
}

func writeTestConfig(t *testing.T, apiKey string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "deepseek": {
    "api_key": "` + apiKey + `",
    "model": "deepseek-chat",
    "base_url": "https://api.deepseek.com/v1/chat/completions"
  },
  "mongo": {
    "uri": "mongodb://admin:password@localhost:27017",
    "database": "travel_inspiration",
    "timeout": 10000000000
  },
  "rag": {
    "enabled": false,
    "max_wiki_docs": 3,
    "max_context_chars": 2000
  }
}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}

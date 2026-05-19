package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config stores global configuration loaded from config.json.
type Config struct {
	Deepseek  DeepseekConfig  `json:"deepseek"`
	Mongo     MongoConfig     `json:"mongo"`
	Session   SessionConfig   `json:"session"`
	Wikipedia WikipediaConfig `json:"wikipedia"`
	RAG       RAGConfig       `json:"rag"`
}

type DeepseekConfig struct {
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
}

type MongoConfig struct {
	URI      string        `json:"uri"`
	Database string        `json:"database"`
	Timeout  time.Duration `json:"timeout"`
}

type SessionConfig struct {
	MaxMessageReserved int `json:"max_message_reserved"`
}

type WikipediaConfig struct {
	Language  string `json:"language"`
	UserAgent string `json:"user_agent"`
	Proxy     string `json:"proxy"`
}

type RAGConfig struct {
	Enabled         bool `json:"enabled"`
	MaxWikiDocs     int  `json:"max_wiki_docs"`
	MaxContextChars int  `json:"max_context_chars"`
}

type TokenConfig struct {
	MaxToken       int     `json:"max_token"`
	WarningPercent float64 `json:"warning_percent"`
}

var config *Config

const deepseekAPIKeyPlaceholder = "偷api_key的替我挡灾"

// Global returns the configuration that was loaded at init.
func Global() *Config {
	if config == nil {
		panic("config not initialized")
	}
	return config
}

func init() {
	path, err := resolveConfigPath()
	if err != nil {
		if isTestBinary() {
			return
		}
		panic(fmt.Sprintf("resolve config path: %v", err))
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		if isTestBinary() {
			return
		}
		panic(fmt.Sprintf("load config: %v", err))
	}
	config = cfg
}

// LoadConfig reads config.json and returns the parsed configuration.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	cfg.Deepseek.APIKey = resolveDeepseekAPIKey(cfg.Deepseek.APIKey)
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// resolveConfigPath finds config.json by walking upward or using CONFIG_PATH.
func resolveConfigPath() (string, error) {
	if custom := os.Getenv("CONFIG_PATH"); custom != "" {
		return custom, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get workdir: %w", err)
	}

	for {
		candidate := filepath.Join(wd, "config.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return "", fmt.Errorf("config.json not found; set CONFIG_PATH")
}

func (cfg *Config) applyDefaults() {
	if cfg.Mongo.URI == "" {
		cfg.Mongo.URI = "mongodb://admin:password@localhost:27017"
	}
	if cfg.Mongo.Database == "" {
		cfg.Mongo.Database = "travel_inspiration"
	}
	if cfg.Mongo.Timeout <= 0 {
		cfg.Mongo.Timeout = 10 * time.Second
	}
	if cfg.Wikipedia.Language == "" {
		cfg.Wikipedia.Language = "zh"
	}
	if cfg.RAG.MaxWikiDocs <= 0 {
		cfg.RAG.MaxWikiDocs = 3
	}
	if cfg.RAG.MaxContextChars <= 0 {
		cfg.RAG.MaxContextChars = 2000
	}
}

func (cfg *Config) validate() error {
	if cfg.Deepseek.APIKey == "" {
		return fmt.Errorf("deepseek api key is missing; provide deepseek.api_key in config.json or DEEPSEEK_API_KEY")
	}
	if cfg.Deepseek.Model == "" {
		return fmt.Errorf("config deepseek.model is empty")
	}
	if cfg.Mongo.URI == "" {
		return fmt.Errorf("config mongo.uri is empty")
	}
	if cfg.Mongo.Database == "" {
		return fmt.Errorf("config mongo.database is empty")
	}
	if cfg.Mongo.Timeout <= 0 {
		return fmt.Errorf("config mongo.timeout must be > 0")
	}
	if cfg.RAG.MaxWikiDocs <= 0 {
		return fmt.Errorf("config rag.max_wiki_docs must be > 0")
	}
	if cfg.RAG.MaxContextChars <= 0 {
		return fmt.Errorf("config rag.max_context_chars must be > 0")
	}
	return nil
}

func resolveDeepseekAPIKey(configValue string) string {
	key := strings.TrimSpace(configValue)
	if key != "" && key != deepseekAPIKeyPlaceholder {
		return key
	}

	return strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
}

func isTestBinary() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}

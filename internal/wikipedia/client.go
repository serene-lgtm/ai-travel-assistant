package wikipedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://%s.wikipedia.org/api/rest_v1"
	defaultLang      = "zh"
	defaultUserAgent = "ai-reading-assistant/1.0"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

type Summary struct {
	Title       string `json:"title"`
	Extract     string `json:"extract"`
	Description string `json:"description"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
		Mobile struct {
			Page string `json:"page"`
		} `json:"mobile"`
	} `json:"content_urls"`
}

type Option func(*clientConfig) error

type clientConfig struct {
	lang       string
	baseURL    string
	timeout    time.Duration
	userAgent  string
	proxy      string
	httpClient *http.Client
}

func NewClient(opts ...Option) (*Client, error) {
	cfg := clientConfig{
		lang:      defaultLang,
		timeout:   10 * time.Second,
		userAgent: defaultUserAgent,
	}

	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if cfg.proxy != "" {
			proxyURL, err := url.Parse(cfg.proxy)
			if err != nil {
				return nil, fmt.Errorf("parse wikipedia proxy: %w", err)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}
		httpClient = &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		}
	}

	baseURL := cfg.baseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf(defaultBaseURL, cfg.lang)
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		userAgent:  cfg.userAgent,
	}, nil
}

func WithLanguage(lang string) Option {
	return func(cfg *clientConfig) error {
		lang = strings.TrimSpace(lang)
		if lang == "" {
			return fmt.Errorf("wikipedia language is required")
		}
		cfg.lang = lang
		return nil
	}
}

func WithBaseURL(baseURL string) Option {
	return func(cfg *clientConfig) error {
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			return fmt.Errorf("wikipedia baseURL is required")
		}
		cfg.baseURL = baseURL
		return nil
	}
}

func WithUserAgent(userAgent string) Option {
	return func(cfg *clientConfig) error {
		userAgent = strings.TrimSpace(userAgent)
		if userAgent == "" {
			return fmt.Errorf("wikipedia userAgent is required")
		}
		cfg.userAgent = userAgent
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(cfg *clientConfig) error {
		if timeout <= 0 {
			return fmt.Errorf("wikipedia timeout must be > 0")
		}
		cfg.timeout = timeout
		return nil
	}
}

func WithProxy(proxy string) Option {
	return func(cfg *clientConfig) error {
		cfg.proxy = strings.TrimSpace(proxy)
		return nil
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(cfg *clientConfig) error {
		if httpClient == nil {
			return fmt.Errorf("wikipedia http client is nil")
		}
		cfg.httpClient = httpClient
		return nil
	}
}

func (c *Client) GetSummary(ctx context.Context, keyword string) (*Summary, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("wikipedia keyword is required")
	}

	apiURL := c.baseURL + "/page/summary/" + url.PathEscape(keyword)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build wikipedia request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call wikipedia summary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("wikipedia summary error: status %d body %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	var summary Summary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("decode wikipedia summary: %w", err)
	}
	return &summary, nil
}

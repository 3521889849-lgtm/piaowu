package llm

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

func FromEnv() Config {
	timeout := 15 * time.Second
	if v := strings.TrimSpace(os.Getenv("ASSISTANT_LLM_TIMEOUT_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 300 {
			timeout = time.Duration(n) * time.Second
		}
	}

	return Config{
		Provider: strings.TrimSpace(os.Getenv("ASSISTANT_LLM_PROVIDER")),
		BaseURL:  strings.TrimSpace(os.Getenv("ASSISTANT_LLM_BASE_URL")),
		APIKey:   strings.TrimSpace(os.Getenv("ASSISTANT_LLM_API_KEY")),
		Model:    strings.TrimSpace(os.Getenv("ASSISTANT_LLM_MODEL")),
		Timeout:  timeout,
	}
}

func NewFromConfig(cfg Config) (Client, error) {
	p := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if p == "" || p == "mock" {
		return NewMock(), nil
	}
	if p == "openai" || p == "openai_compat" {
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.openai.com"
		}
		if cfg.Model == "" {
			cfg.Model = "gpt-4o-mini"
		}
		if cfg.Timeout == 0 {
			cfg.Timeout = 15 * time.Second
		}
		return NewOpenAICompat(cfg)
	}
	if p == "qwen" || p == "dashscope" || p == "wenxin" || p == "ernie" {
		if cfg.BaseURL == "" || cfg.Model == "" {
			return nil, errors.New("Provider=" + cfg.Provider + " 需要配置 ASSISTANT_LLM_BASE_URL 与 ASSISTANT_LLM_MODEL（OpenAI 兼容接口）")
		}
		if cfg.Timeout == 0 {
			cfg.Timeout = 15 * time.Second
		}
		return NewOpenAICompat(cfg)
	}
	return nil, errors.New("未知LLM Provider: " + cfg.Provider)
}

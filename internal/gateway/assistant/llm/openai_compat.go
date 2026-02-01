package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type openAICompatClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAICompat(cfg Config) (Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("BaseURL为空")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("APIKey为空")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, errors.New("Model为空")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return &openAICompatClient{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaChatRequest struct {
	Model    string      `json:"model"`
	Messages []oaMessage `json:"messages"`
}

type oaChatResponse struct {
	Choices []struct {
		Message oaMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *openAICompatClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	userMsg := strings.TrimSpace(req.UserMessage)
	if userMsg == "" {
		return ChatResponse{}, errors.New("UserMessage为空")
	}

	system := "你是票务系统的智能助手。优先使用给定的上下文回答；不要编造不存在的订单/车次/规则。"
	if strings.TrimSpace(req.Context) != "" {
		system += "\n\n上下文：\n" + strings.TrimSpace(req.Context)
	}

	body, err := json.Marshal(oaChatRequest{
		Model: c.model,
		Messages: []oaMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: userMsg},
		},
	})
	if err != nil {
		return ChatResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, err
	}

	var decoded oaChatResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ChatResponse{}, err
	}
	if decoded.Error != nil && strings.TrimSpace(decoded.Error.Message) != "" {
		return ChatResponse{}, errors.New(decoded.Error.Message)
	}
	if len(decoded.Choices) == 0 {
		return ChatResponse{}, errors.New("LLM响应为空")
	}
	text := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if text == "" {
		return ChatResponse{}, errors.New("LLM响应内容为空")
	}
	return ChatResponse{Text: text}, nil
}


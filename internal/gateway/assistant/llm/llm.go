package llm

import (
	"context"
	"strings"
)

type ChatRequest struct {
	UserMessage string
	Context     string
}

type ChatResponse struct {
	Text string
}

type Client interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type mockClient struct{}

func NewMock() Client {
	return &mockClient{}
}

func (m *mockClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	_ = ctx
	msg := strings.TrimSpace(req.UserMessage)
	if msg == "" {
		return ChatResponse{Text: "请输入问题。"}, nil
	}
	if strings.TrimSpace(req.Context) != "" {
		return ChatResponse{Text: "结合知识库信息，我的建议是：\n" + strings.TrimSpace(req.Context)}, nil
	}
	return ChatResponse{Text: "我可以帮你：\n1) 查车次/余票\n2) 查车次详情\n3) 查订单列表\n你可以直接说：查 2026-02-03 从北京到上海 的车次。"}, nil
}


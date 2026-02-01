/*
 * Assistant 核心引擎
 *
 * 功能说明：
 * - 协调 NLU、RAG、LLM 与 Tools 模块
 * - 实现 Chat 主流程控制
 */
package assistant

import (
	"context"
	"errors"
	"example_shop/common/db"
	"example_shop/internal/gateway/assistant/llm"
	"example_shop/internal/gateway/assistant/nlu"
	"example_shop/internal/gateway/assistant/rag"
	"example_shop/internal/gateway/http/dto"
	"example_shop/kitex_gen/orderapi/orderservice"
	"example_shop/kitex_gen/ticketapi/ticketservice"
	"example_shop/kitex_gen/userapi/userservice"
	"strings"
)

// ChatInput 对话输入参数
type ChatInput struct {
	UserID  string
	Message string
	Mode    string // "plan" 或 "auto"
	UseRAG  bool   // 是否开启知识库召回
	Debug   bool
}

// ChatOutput 对话输出结果
type ChatOutput struct {
	Reply      string                   // AI 回复文本
	Intent     *dto.AssistantIntent     // 识别出的意图（调试用）
	ToolName   string                   // 触发的工具名称（如有）
	ToolData   any                      // 工具执行结果数据
	RAGSources []dto.AssistantRAGSnippet // 引用到的知识库片段
}

// Engine 助手引擎
// 聚合了 LLM, NLU, RAG, Tools 和 RPC 客户端
type Engine struct {
	llm          llm.Client
	nlu          nlu.Recognizer
	rag          rag.Store
	tools        *ToolRegistry
	userClient   userservice.Client
	ticketClient ticketservice.Client
	orderClient  orderservice.Client
}

// NewEngine 初始化引擎
// 自动加载 LLM 配置，若失败则降级为 Mock
func NewEngine(userClient userservice.Client, ticketClient ticketservice.Client, orderClient orderservice.Client) *Engine {
	llmClient, err := llm.NewFromConfig(llm.FromEnv())
	if err != nil {
		llmClient = llm.NewMock()
	}
	return &Engine{
		llm:          llmClient,
		nlu:          nlu.NewRuleBased(),          // 使用规则 NLU
		rag:          rag.NewMySQLStore(db.ReadDB), // 使用 MySQL 知识库
		tools:        NewToolRegistry(ticketClient, orderClient),
		userClient:   userClient,
		ticketClient: ticketClient,
		orderClient:  orderClient,
	}
}

// Chat 处理一次对话请求
// 核心流程：
// 1. NLU 意图识别
// 2. RAG 知识检索 (如果开启)
// 3. 意图路由：
//    - 命中工具意图 (查票/查订单)：
//      - 检查槽位是否齐全
//      - Mode="plan": 仅返回规划
//      - Mode="auto": 执行工具并返回结果
//    - 未命中工具意图：
//      - 拼接 RAG 上下文
//      - 调用 LLM 生成闲聊/问答回复
func (e *Engine) Chat(ctx context.Context, in ChatInput) (ChatOutput, error) {
	msg := strings.TrimSpace(in.Message)
	if msg == "" {
		return ChatOutput{}, errors.New("message不能为空")
	}

	// 1. 意图识别
	intent := e.nlu.Recognize(ctx, msg)
	out := ChatOutput{
		Intent: intent,
	}

	// 2. 知识库召回 (RAG)
	var ragSnips []rag.Snippet
	if in.UseRAG {
		ragSnips, _ = e.rag.Search(ctx, rag.SearchQuery{Query: msg, Limit: 3})
		out.RAGSources = make([]dto.AssistantRAGSnippet, 0, len(ragSnips))
		for _, s := range ragSnips {
			out.RAGSources = append(out.RAGSources, dto.AssistantRAGSnippet{
				DocKey:  s.DocKey,
				Title:   s.Title,
				Content: s.Content,
			})
		}
	}

	// 3. 意图处理路由
	switch intent.Name {
	// 命中工具类意图
	case nlu.IntentSearchTrain, nlu.IntentTrainDetail, nlu.IntentListOrders:
		toolName := map[string]string{
			nlu.IntentSearchTrain: "SearchTrain",
			nlu.IntentTrainDetail: "TrainDetail",
			nlu.IntentListOrders:  "ListOrders",
		}[intent.Name]

		tool, ok := e.tools.Get(toolName)
		if !ok {
			out.Reply = "当前不支持该操作。"
			return out, nil
		}
		
		// 检查必填参数是否齐全
		missing := MissingSlots(tool.RequiredSlots(), intent.Slots)
		
		// Plan 模式：只规划，不执行
		if strings.EqualFold(strings.TrimSpace(in.Mode), "plan") {
			if len(missing) > 0 {
				out.Reply = "规划结果：将调用 " + toolName + "。缺少参数：" + strings.Join(missing, ", ")
				return out, nil
			}
			out.Reply = "规划结果：将调用 " + toolName + "。参数已齐全，可直接执行。"
			return out, nil
		}
		
		// Auto 模式：参数不齐提示补全
		if len(missing) > 0 {
			out.Reply = "我可以帮你执行 " + toolName + "，但还缺少参数：" + strings.Join(missing, ", ")
			return out, nil
		}

		// 执行工具
		data, err := tool.Execute(ctx, ToolInput{UserID: in.UserID, Slots: intent.Slots})
		if err != nil {
			return ChatOutput{}, err
		}
		out.ToolName = toolName
		out.ToolData = data
		out.Reply = "已为你执行 " + toolName + "。"
		return out, nil

	// 未命中工具，走 LLM 闲聊/问答
	default:
		prompt := llm.ChatRequest{
			UserMessage: msg,
			Context:     rag.SnippetsToText(ragSnips), // 注入 RAG 上下文
		}
		resp, err := e.llm.Chat(ctx, prompt)
		if err != nil {
			// LLM 失败兜底回复
			out.Reply = "我可以帮你查票、查车次详情或查询订单。你可以直接说：查 2026-02-03 从北京到上海 的车次。"
			return out, nil
		}
		out.Reply = resp.Text
		return out, nil
	}
}

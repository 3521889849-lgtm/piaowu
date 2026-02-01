/*
 * Assistant HTTP 处理器
 *
 * 功能说明：
 * - 处理智能助手相关的HTTP请求
 * - 管理知识库的增删改查
 * - 确保知识库数据库表的初始化
 */
package assistant

import (
	"context"
	"errors"
	"example_shop/common/db"
	assistantlogic "example_shop/internal/gateway/assistant"
	"example_shop/internal/gateway/http/dto"
	"example_shop/internal/gateway/http/middleware"
	"example_shop/internal/model"
	"example_shop/kitex_gen/orderapi/orderservice"
	"example_shop/kitex_gen/ticketapi/ticketservice"
	"example_shop/kitex_gen/userapi/userservice"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

// Handler 智能助手处理器
type Handler struct {
	engine *assistantlogic.Engine
}

var (
	// kbMigrateOnce 确保知识库表只迁移一次
	kbMigrateOnce sync.Once
	// kbMigrateErr 记录迁移过程中的错误
	kbMigrateErr  error
)

// ensureKnowledgeTable 确保知识库表已创建
// 懒加载模式：首次请求时自动检查并创建表
func ensureKnowledgeTable(ctx context.Context) error {
	if db.MysqlDB == nil {
		return errors.New("MySQL未初始化")
	}
	kbMigrateOnce.Do(func() {
		kbMigrateErr = db.MysqlDB.WithContext(ctx).AutoMigrate(&model.KnowledgeDoc{})
	})
	return kbMigrateErr
}

// New 创建 Assistant 处理器实例
func New(userClient userservice.Client, ticketClient ticketservice.Client, orderClient orderservice.Client) *Handler {
	return &Handler{
		engine: assistantlogic.NewEngine(userClient, ticketClient, orderClient),
	}
}

// Chat 智能对话接口
//
// 路由: POST /api/v1/assistant/chat
//
// 功能：
// 1. 接收用户输入的消息
// 2. 调用 NLU 识别意图
// 3. (可选) 检索 RAG 知识库增强上下文
// 4. (可选) 执行工具调用（如查票、查订单）
// 5. 调用 LLM 生成自然语言回复
func (h *Handler) Chat(ctx context.Context, c *app.RequestContext) {
	var req dto.AssistantChatHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	// 必须登录才能使用助手（因为很多工具依赖 UserID）
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}

	res, err := h.engine.Chat(ctx, assistantlogic.ChatInput{
		UserID:  userID,
		Message: req.Message,
		Mode:    req.Mode,    // "plan"=只规划不执行, "auto"=自动执行
		UseRAG:  req.UseRAG,  // 是否启用知识库增强
		Debug:   req.Debug,
	})
	if err != nil {
		c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: err.Error()})
		return
	}

	c.JSON(200, dto.AssistantChatHTTPResp{
		Code:      200,
		Msg:       "success",
		Reply:     res.Reply,
		Intent:    res.Intent,
		ToolName:  res.ToolName,
		ToolData:  res.ToolData,
		RAGSources: res.RAGSources,
	})
}

// KBUpsert 知识库文档写入/更新接口
//
// 路由: POST /api/v1/assistant/kb/upsert
//
// 功能：
// - 根据 DocKey 唯一键判断是插入还是更新
// - 用于运营后台或人工录入知识文档
func (h *Handler) KBUpsert(ctx context.Context, c *app.RequestContext) {
	var req dto.AssistantKBUpsertHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}
	if err := ensureKnowledgeTable(ctx); err != nil {
		c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: err.Error()})
		return
	}

	doc := model.KnowledgeDoc{
		DocKey:  req.DocKey,
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
	}

	var existing model.KnowledgeDoc
	// 先查后写，模拟 Upsert 逻辑
	err := db.MysqlDB.WithContext(ctx).Where("doc_key = ?", req.DocKey).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: err.Error()})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.MysqlDB.WithContext(ctx).Create(&doc).Error; err != nil {
			c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: err.Error()})
			return
		}
		c.JSON(200, dto.BaseHTTPResp{Code: 200, Msg: "success"})
		return
	}

	existing.Title = doc.Title
	existing.Content = doc.Content
	existing.Tags = doc.Tags
	if err := db.MysqlDB.WithContext(ctx).Save(&existing).Error; err != nil {
		c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: err.Error()})
		return
	}
	c.JSON(200, dto.BaseHTTPResp{Code: 200, Msg: "success"})
}

// KBSearch 知识库文档搜索接口
//
// 路由: POST /api/v1/assistant/kb/search
//
// 功能：
// - 简单的关键词模糊搜索 (LIKE %query%)
// - 检索 Title, Content, Tags 字段
// - 用于验证召回效果或管理后台查询
func (h *Handler) KBSearch(ctx context.Context, c *app.RequestContext) {
	var req dto.AssistantKBSearchHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}
	if err := ensureKnowledgeTable(ctx); err != nil {
		c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: err.Error()})
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	var rows []model.KnowledgeDoc
	// 简单的多字段 LIKE 搜索
	if err := db.ReadDB().WithContext(ctx).
		Where("title LIKE ? OR content LIKE ? OR tags LIKE ?", "%"+req.Query+"%", "%"+req.Query+"%", "%"+req.Query+"%").
		Order("updated_at DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: err.Error()})
		return
	}

	items := make([]dto.AssistantKBSearchItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, dto.AssistantKBSearchItem{
			DocKey:  r.DocKey,
			Title:   r.Title,
			Content: r.Content,
			Tags:    r.Tags,
		})
	}
	c.JSON(200, dto.AssistantKBSearchHTTPResp{Code: 200, Msg: "success", Items: items})
}

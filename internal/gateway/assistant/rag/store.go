/*
 * RAG 知识检索模块
 *
 * 功能说明：
 * - 提供知识库的检索接口
 * - 实现基于 MySQL 的关键词召回（轻量级方案）
 * - 辅助将检索结果拼接为 LLM 上下文
 */
package rag

import (
	"context"
	"errors"
	"example_shop/internal/model"
	"strings"

	"gorm.io/gorm"
)

// SearchQuery 检索请求
type SearchQuery struct {
	Query string // 搜索关键词
	Limit int    // 返回条数限制
}

// Snippet 检索到的知识片段
type Snippet struct {
	DocKey  string
	Title   string
	Content string
}

// Store 知识库存储接口
type Store interface {
	// Search 根据 query 检索相关文档片段
	Search(ctx context.Context, q SearchQuery) ([]Snippet, error)
}

// mysqlStore 基于 MySQL 的轻量实现
type mysqlStore struct {
	db func() *gorm.DB
}

// NewMySQLStore 创建 MySQL 知识库实例
func NewMySQLStore(dbFn func() *gorm.DB) Store {
	return &mysqlStore{db: dbFn}
}

// Search 执行 MySQL LIKE 搜索
// 逻辑：
// 1. 匹配 title, content, tags 任意字段
// 2. 按 updated_at 倒序排列（优先用最新的知识）
// 3. 对长文本进行截断，避免 Token 消耗过大
func (s *mysqlStore) Search(ctx context.Context, q SearchQuery) ([]Snippet, error) {
	if s.db == nil {
		return nil, errors.New("db未初始化")
	}
	gdb := s.db()
	if gdb == nil {
		return nil, errors.New("db未连接")
	}
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return nil, nil
	}
	limit := q.Limit
	if limit <= 0 || limit > 10 {
		limit = 3
	}

	var rows []model.KnowledgeDoc
	// 简单的关键词匹配：Title OR Content OR Tags
	tx := gdb.WithContext(ctx).Model(&model.KnowledgeDoc{}).
		Where("title LIKE ? OR content LIKE ? OR tags LIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%").
		Order("updated_at DESC").
		Limit(limit).
		Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}

	out := make([]Snippet, 0, len(rows))
	for _, r := range rows {
		content := strings.TrimSpace(r.Content)
		// 截断过长的内容（TopK 召回后需要控制上下文长度）
		if len([]rune(content)) > 280 {
			rs := []rune(content)
			content = string(rs[:280]) + "…"
		}
		out = append(out, Snippet{
			DocKey:  r.DocKey,
			Title:   r.Title,
			Content: content,
		})
	}
	return out, nil
}

// SnippetsToText 将检索到的片段拼接为纯文本
// 格式：
// 标题1
// 内容1
//
// 标题2
// 内容2
func SnippetsToText(snips []Snippet) string {
	if len(snips) == 0 {
		return ""
	}
	var b strings.Builder
	for i, s := range snips {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if strings.TrimSpace(s.Title) != "" {
			b.WriteString(s.Title)
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimSpace(s.Content))
	}
	return b.String()
}

package model

import (
	"time"

	"gorm.io/gorm"
)

type KnowledgeDoc struct {
	ID        uint64         `gorm:"column:id;type:BIGINT UNSIGNED;primaryKey;autoIncrement" json:"id"`
	DocKey    string         `gorm:"column:doc_key;type:VARCHAR(128);NOT NULL;uniqueIndex" json:"doc_key"`
	Title     string         `gorm:"column:title;type:VARCHAR(255);NOT NULL;index:idx_kb_title" json:"title"`
	Content   string         `gorm:"column:content;type:LONGTEXT;NOT NULL" json:"content"`
	Tags      string         `gorm:"column:tags;type:VARCHAR(255);NOT NULL;default:'';index:idx_kb_tags" json:"tags"`
	CreatedAt time.Time      `gorm:"column:created_at;type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:DATETIME;index" json:"deleted_at,omitempty"`
}


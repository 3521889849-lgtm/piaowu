package audit

import (
	"time"

	"gorm.io/gorm"
)

// AuditFlowConfig 审核流配置表 (audit_flow_configs)
type AuditFlowConfig struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement;comment:流配置主键ID"`
	BizType      string         `gorm:"type:varchar(32);not null;index;comment:业务类型"`
	FlowName     string         `gorm:"type:varchar(64);not null;comment:审核流名称"`
	ConfigJSON   string         `gorm:"type:json;not null;comment:流配置详情(JSON)"`
	Status       int8           `gorm:"type:tinyint;default:1;comment:状态(1:启用,0:禁用)"`
	CreatedAt    time.Time      `gorm:"comment:创建时间"`
	UpdatedAt    time.Time      `gorm:"comment:更新时间"`
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (AuditFlowConfig) TableName() string {
	return "audit_flow_configs"
}

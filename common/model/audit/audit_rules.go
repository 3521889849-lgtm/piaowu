package audit

import (
	"time"

	"gorm.io/gorm"
)

// AuditRule 审核规则明细表 (audit_rules)
type AuditRule struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement;comment:规则主键ID"`
	FlowId      uint64         `gorm:"not null;index;comment:关联流配置ID"`
	RuleName    string         `gorm:"type:varchar(64);not null;comment:规则名称"`
	RuleType    string         `gorm:"type:varchar(32);not null;comment:规则类型"`
	Expression  string         `gorm:"type:text;not null;comment:规则表达式"`
	Priority    int            `gorm:"type:int;default:0;comment:优先级"`
	Status      int8           `gorm:"type:tinyint;default:1;comment:状态(1:启用,0:禁用)"`
	CreatedAt   time.Time      `gorm:"comment:创建时间"`
	UpdatedAt   time.Time      `gorm:"comment:更新时间"`
	DeletedAt   gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (AuditRule) TableName() string {
	return "audit_rules"
}

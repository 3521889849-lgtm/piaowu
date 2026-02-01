package audit

import (
	"time"

	"gorm.io/gorm"
)

// RuleConfig 规则配置表
type RuleConfig struct {
	ID         int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	BizType    string         `gorm:"type:varchar(32);index;not null;comment:业务类型(ticket_order/merchant_apply)" json:"biz_type"`
	RuleName   string         `gorm:"type:varchar(64);not null;comment:规则名称" json:"rule_name"`
	Expression string         `gorm:"type:text;not null;comment:规则表达式(JSON)" json:"expression"`
	Action     string         `gorm:"type:varchar(32);not null;comment:执行动作(Pass/Reject/Review)" json:"action"`
	Priority   int            `gorm:"type:int;default:0;comment:优先级(值越大越优先)" json:"priority"`
	Status     int            `gorm:"type:tinyint;default:1;comment:状态(1:启用,0:禁用)" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (RuleConfig) TableName() string {
	return "rule_configs"
}


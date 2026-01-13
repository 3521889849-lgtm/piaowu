package model

import "time"

const TableNameSysOperLog = "sys_oper_log"

// SysOperLog 系统操作日志表-管理端所有手动操作记录
type SysOperLog struct {
	ID          uint64    `gorm:"column:id;type:BIGINT UNSIGNED;primaryKey;autoIncrement;comment:日志主键ID" json:"id"`
	OperType    string    `gorm:"column:oper_type;type:VARCHAR(30);not null;index:idx_oper_type;comment:操作类型" json:"oper_type"`
	OperAdminID uint64    `gorm:"column:oper_admin_id;type:BIGINT UNSIGNED;not null;index:idx_oper_admin_id;comment:操作管理员ID" json:"oper_admin_id"`
	OperContent string    `gorm:"column:oper_content;type:VARCHAR(512);not null;comment:操作内容" json:"oper_content"`
	BusinessID  uint64    `gorm:"column:business_id;type:BIGINT UNSIGNED;not null;index:idx_business_id;comment:业务ID" json:"business_id"`
	OperIP      string    `gorm:"column:oper_ip;type:VARCHAR(50);not null;comment:操作IP地址" json:"oper_ip"`
	CreatedAt   time.Time `gorm:"column:created_at;type:DATETIME;not null;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`

	Admin *SysAdmin `gorm:"foreignKey:OperAdminID;references:ID" json:"admin,omitempty"`
}

func (SysOperLog) TableName() string { return TableNameSysOperLog }

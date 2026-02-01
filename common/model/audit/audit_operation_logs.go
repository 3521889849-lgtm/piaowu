package audit

import (
	"time"

	"gorm.io/gorm"
)

// 4. 审核操作日志表（audit_operation_logs）- 追溯审核全流程操作
type AuditOperationLog struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement;comment:操作日志主键ID"`
	AuditMainId     uint64         `gorm:"not null;index:idx_audit_main_id;comment:关联审核主表ID"`
	OperatorId      uint64         `gorm:"not null;comment:操作人ID"`
	OperatorName    string         `gorm:"type:varchar(50);not null;comment:操作人名称"`
	OperationType   int8           `gorm:"type:tinyint;not null;comment:操作类型（1=提交, 2=接收, 3=通过, 4=驳回, 5=撤销）"`
	OperationRemark string         `gorm:"type:text;comment:操作备注"`
	CreatedAt       time.Time      `gorm:"index;comment:操作时间"`
	UpdatedAt       time.Time      `gorm:"comment:更新时间"`
	DeletedAt       gorm.DeletedAt `gorm:"softDelete:delete_at;comment:软删除时间"`
}

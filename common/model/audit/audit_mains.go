package audit

import (
	"time"

	"gorm.io/gorm"
)

// 1. 审核主表（audit_mains）- 统一管理所有审核任务
type AuditMain struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement;comment:审核任务主键ID"`
	BusinessType   int8           `gorm:"type:tinyint;not null;index:idx_business_type_status;comment:审核业务类型（1=车票订单, 2=酒店订单, 3=酒店入驻）"`
	BusinessId     uint64         `gorm:"not null;index:idx_business_id;comment:关联业务主键ID（如车票订单ID、酒店ID）"`
	AuditStatus    int8           `gorm:"type:tinyint;not null;default:1;index:idx_business_type_status;comment:审核状态（1=待审核, 2=审核中, 3=通过, 4=驳回, 5=撤销）"`
	SubmitUserId   uint64         `gorm:"not null;comment:提交人ID"`
	SubmitUserName string         `gorm:"type:varchar(50);not null;comment:提交人名称"`
	AuditUserId    uint64         `gorm:"default:0;comment:审核人ID"`
	AuditUserName  string         `gorm:"type:varchar(50);default:'';comment:审核人名称"`
	AuditRemark    string         `gorm:"type:text;comment:审核备注"`
	AuditTime      *time.Time     `gorm:"comment:审核完成时间"`
	Extra          string         `gorm:"type:json;comment:扩展字段（JSON格式）"`
	CreatedAt      time.Time      `gorm:"comment:提交时间"`
	UpdatedAt      time.Time      `gorm:"comment:更新时间"`
	DeletedAt      gorm.DeletedAt `gorm:"softDelete:delete_at;index;comment:软删除时间"`
}


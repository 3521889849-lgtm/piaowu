package Review

import (
	"time"

	"gorm.io/gorm"
)

// AuditGroupRelation 审核人员与分组关联表（负载均衡数据源）
type AuditGroupRelation struct {
	gorm.Model
	GroupID     uint      `gorm:"column:group_id;type:bigint;not null;index:idx_group_id;comment:关联分组ID"`
	AuditorID   string    `gorm:"column:auditor_id;type:varchar(64);not null;index:idx_auditor_id;comment:审核人员ID"`
	AuditorName string    `gorm:"column:auditor_name;type:varchar(64);not null;comment:审核人员名称"`
	LoadCount   int       `gorm:"column:load_count;type:int;not null;default:0;index:idx_load_count;comment:当前待处理工单量"`
	Status      int8      `gorm:"column:status;type:tinyint;not null;index:idx_status;comment:关联状态：0=禁用,1=启用"`
	CreateTime  time.Time `gorm:"column:create_time;type:datetime;not null;comment:关联创建时间"`
	UpdateTime  time.Time `gorm:"column:update_time;type:datetime;not null;comment:关联更新时间"`
}

// TableName 指定审核人员关联表表名
func (AuditGroupRelation) TableName() string {
	return "audit_group_relation"
}

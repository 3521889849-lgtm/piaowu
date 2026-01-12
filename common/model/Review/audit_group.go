package Review

import (
	"time"

	"gorm.io/gorm"
)

// AuditGroup 审核人员分组表（用于工单自动分配）
type AuditGroup struct {
	gorm.Model
	GroupName    string    `gorm:"column:group_name;type:varchar(64);not null;comment:分组名称"`
	GroupType    int8      `gorm:"column:group_type;type:tinyint;not null;index:idx_group_type;comment:分组对应业务类型：1=车票相关审核,2=酒店相关审核"`
	AuditorLevel int8      `gorm:"column:auditor_level;type:tinyint;not null;index:idx_auditor_level;comment:分组审核等级：1=一级审核,2=二级审核"`
	Status       int8      `gorm:"column:status;type:tinyint;not null;index:idx_status;comment:分组状态：0=禁用,1=启用"`
	CreateTime   time.Time `gorm:"column:create_time;type:datetime;not null;comment:分组创建时间"`
	UpdateTime   time.Time `gorm:"column:update_time;type:datetime;not null;comment:分组更新时间"`
	Remark       string    `gorm:"column:remark;type:varchar(512);comment:分组备注"`
}

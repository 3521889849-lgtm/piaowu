package Review

import (
	"time"

	"gorm.io/gorm"
)

// AuditOrderDetail 审核工单详情表（记录审核全流程轨迹）
type AuditOrderDetail struct {
	gorm.Model
	AuditOrderID uint      `gorm:"column:audit_order_id;type:bigint;not null;index:idx_audit_order_id;comment:关联审核工单ID"`
	AuditOrderNo string    `gorm:"column:audit_order_no;type:varchar(64);not null;comment:关联审核工单编号"`
	AuditorLevel int8      `gorm:"column:auditor_level;type:tinyint;not null;comment:本次审核等级：1=一级审核,2=二级审核"`
	AuditorID    string    `gorm:"column:auditor_id;type:varchar(64);not null;index:idx_auditor_id;comment:本次审核人ID"`
	AuditorName  string    `gorm:"column:auditor_name;type:varchar(64);not null;comment:本次审核人名称"`
	AuditResult  int8      `gorm:"column:audit_result;type:tinyint;not null;comment:本次审核结果：0=待审核,1=审核通过,2=审核驳回,3=流转至下一级"`
	AuditOpinion string    `gorm:"column:audit_opinion;type:text;comment:本次审核意见"`
	AuditTime    time.Time `gorm:"column:audit_time;type:datetime;not null;index:idx_audit_time;comment:本次审核完成时间"`
	Remark       string    `gorm:"column:remark;type:varchar(512);comment:本次审核备注"`
}

// TableName 指定审核工单详情表表名
func (AuditOrderDetail) TableName() string {
	return "audit_order_detail"
}

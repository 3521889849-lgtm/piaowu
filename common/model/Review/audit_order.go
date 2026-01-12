package Review

import (
	"time"

	"gorm.io/gorm"
)

// AuditOrder 审核工单主表（GORM结构体）
type AuditOrder struct {
	gorm.Model
	OrderNo            string     `gorm:"column:order_no;type:varchar(64);not null;uniqueIndex:uk_order_no;comment:工单编号"`
	OrderType          int8       `gorm:"column:order_type;type:tinyint;not null;comment:工单分类类型：1=车票异常订单,2=车票退票,3=车票改签,4=酒店入驻,5=酒店订单异常,6=酒店退款"`
	OrderTypeName      string     `gorm:"column:order_type_name;type:varchar(32);not null;comment:工单分类名称"`
	InitiatorType      int8       `gorm:"column:initiator_type;type:tinyint;not null;comment:发起方类型：1=系统自动触发,2=运营人员手动发起,3=商家申请,4=用户申请"`
	InitiatorID        string     `gorm:"column:initiator_id;type:varchar(64);comment:发起方ID"`
	InitiatorName      string     `gorm:"column:initiator_name;type:varchar(64);comment:发起方名称"`
	AuditorLevel       int8       `gorm:"column:auditor_level;type:tinyint;not null;comment:审核等级：1=一级审核,2=二级审核"`
	CurrentAuditorID   string     `gorm:"column:current_auditor_id;type:varchar(64);comment:当前审核人ID"`
	CurrentAuditorName string     `gorm:"column:current_auditor_name;type:varchar(64);comment:当前审核人名称"`
	Priority           int8       `gorm:"column:priority;type:tinyint;not null;comment:工单优先级：1=高,2=中,3=低"`
	Status             int8       `gorm:"column:status;type:tinyint;not null;index:idx_status;comment:工单状态：0=待审核,1=审核中,2=审核通过,3=审核驳回,4=已撤销"`
	BusinessID         string     `gorm:"column:business_id;type:varchar(64);not null;index:idx_business_id;comment:关联业务单据ID"`
	BusinessType       int8       `gorm:"column:business_type;type:tinyint;not null;comment:关联业务类型：1=车票订单,2=酒店订单,3=酒店入驻申请,4=退款申请单"`
	Content            string     `gorm:"column:content;type:text;comment:工单核心内容"`
	AttachmentURL      string     `gorm:"column:attachment_url;type:varchar(512);comment:附件地址，多个用逗号分隔"`
	AssignTime         *time.Time `gorm:"column:assign_time;type:datetime;comment:工单分配时间"`
	AuditFinishTime    *time.Time `gorm:"column:audit_finish_time;type:datetime;comment:审核完成时间"`
	CancelTime         *time.Time `gorm:"column:cancel_time;type:datetime;comment:工单撤销时间"`
	TimeoutRemindTime  *time.Time `gorm:"column:timeout_remind_time;type:datetime;comment:超时提醒时间"`
	Remark             string     `gorm:"column:remark;type:varchar(512);comment:备注信息"`
	OriginalOrderID    *uint      `gorm:"column:original_order_id;type:bigint;comment:关联原工单ID（驳回重审用）"`
}

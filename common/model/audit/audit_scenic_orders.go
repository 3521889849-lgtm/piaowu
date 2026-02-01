package audit

import (
	"time"

	"gorm.io/gorm"
)

// 旅游门票审核明细表（audit_scenic_orders）- 旅游门票专属审核信息
type AuditScenicOrder struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement;comment:门票审核明细主键ID"`
	AuditMainId    uint64         `gorm:"not null;index:idx_audit_main_id;comment:关联审核主表ID"`
	ScenicOrderId  uint64         `gorm:"not null;index:idx_scenic_order_id;comment:关联门票订单ID"`
	TicketType     int8           `gorm:"type:tinyint;not null;comment:门票类型（1=景区门票, 2=演出票, 3=展览票, 4=游乐园票）"`
	ScenicName     string         `gorm:"type:varchar(100);not null;comment:景区/场馆名称"`
	ScenicAddress  string         `gorm:"type:varchar(200);not null;comment:景区/场馆地址"`
	TicketName     string         `gorm:"type:varchar(100);not null;comment:门票名称（成人票/儿童票/学生票等）"`
	VisitDate      time.Time      `gorm:"not null;comment:游玩日期"`
	VisitTime      string         `gorm:"type:varchar(50);comment:游玩时间段"`
	TicketQuantity int            `gorm:"not null;comment:门票数量"`
	UnitPrice      float64        `gorm:"type:decimal(10,2);not null;comment:单价（元）"`
	OrderAmount    float64        `gorm:"type:decimal(10,2);not null;comment:订单总金额（元）"`
	ContactName    string         `gorm:"type:varchar(50);not null;comment:联系人姓名"`
	ContactPhone   string         `gorm:"type:varchar(20);not null;comment:联系人电话（脱敏存储）"`
	ContactIdCard  string         `gorm:"type:varchar(18);comment:联系人身份证号（脱敏存储）"`
	ApplyReason    string         `gorm:"type:varchar(255);not null;comment:审核申请原因"`
	ScenicExtra    string         `gorm:"type:json;comment:门票扩展字段"`
	CreatedAt      time.Time      `gorm:"comment:创建时间"`
	UpdatedAt      time.Time      `gorm:"comment:更新时间"`
	DeletedAt      gorm.DeletedAt `gorm:"softDelete:delete_at;index;comment:软删除时间"`
}

func (AuditScenicOrder) TableName() string {
	return "audit_scenic_orders"
}

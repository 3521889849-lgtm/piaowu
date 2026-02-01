package audit

import (
	"time"

	"gorm.io/gorm"
)

// 2. 车票审核明细表（audit_ticket_orders）- 车票专属审核信息
type AuditTicketOrder struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement;comment:车票审核明细主键ID"`
	AuditMainId      uint64         `gorm:"not null;index:idx_audit_main_id;comment:关联审核主表ID"`
	TicketOrderId    uint64         `gorm:"not null;index:idx_ticket_order_id;comment:关联车票订单ID"`
	TicketType       int8           `gorm:"type:tinyint;not null;comment:车票类型（1=高铁, 2=动车, 3=普通火车, 4=汽车票）"`
	DepartureStation string         `gorm:"type:varchar(200);not null;comment:出发站"`
	ArrivalStation   string         `gorm:"type:varchar(200);not null;comment:到达站"`
	DepartureTime    time.Time      `gorm:"not null;comment:发车时间"`
	PassengerName    string         `gorm:"type:varchar(50);not null;comment:乘客姓名"`
	PassengerIdCard  string         `gorm:"type:varchar(18);not null;comment:乘客身份证号（脱敏存储）"`
	OrderAmount      float64        `gorm:"type:decimal(10,2);not null;comment:订单金额（元）"`
	ApplyReason      string         `gorm:"type:varchar(255);not null;comment:审核申请原因"`
	TicketExtra      string         `gorm:"type:json;comment:车票扩展字段"`
	CreatedAt        time.Time      `gorm:"comment:创建时间"`
	UpdatedAt        time.Time      `gorm:"comment:更新时间"`
	DeletedAt        gorm.DeletedAt `gorm:"softDelete:delete_at;index;comment:软删除时间"`
}

package audit

import (
	"time"

	"gorm.io/gorm"
)

// 机票审核明细表（audit_flight_orders）- 机票专属审核信息
type AuditFlightOrder struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement;comment:机票审核明细主键ID"`
	AuditMainId      uint64         `gorm:"not null;index:idx_audit_main_id;comment:关联审核主表ID"`
	FlightOrderId    uint64         `gorm:"not null;index:idx_flight_order_id;comment:关联机票订单ID"`
	FlightType       int8           `gorm:"type:tinyint;not null;comment:航班类型（1=国内航班, 2=国际航班）"`
	FlightNo         string         `gorm:"type:varchar(20);not null;comment:航班号"`
	Airline          string         `gorm:"type:varchar(50);not null;comment:航空公司"`
	DepartureAirport string         `gorm:"type:varchar(100);not null;comment:出发机场"`
	ArrivalAirport   string         `gorm:"type:varchar(100);not null;comment:到达机场"`
	DepartureTime    time.Time      `gorm:"not null;comment:起飞时间"`
	ArrivalTime      time.Time      `gorm:"not null;comment:降落时间"`
	CabinClass       string         `gorm:"type:varchar(20);not null;comment:舱位等级（经济舱/商务舱/头等舱）"`
	PassengerName    string         `gorm:"type:varchar(50);not null;comment:乘客姓名"`
	PassengerIdCard  string         `gorm:"type:varchar(18);not null;comment:乘客身份证号（脱敏存储）"`
	PassengerPhone   string         `gorm:"type:varchar(20);comment:乘客联系电话（脱敏存储）"`
	OrderAmount      float64        `gorm:"type:decimal(10,2);not null;comment:订单金额（元）"`
	ApplyReason      string         `gorm:"type:varchar(255);not null;comment:审核申请原因"`
	FlightExtra      string         `gorm:"type:json;comment:机票扩展字段"`
	CreatedAt        time.Time      `gorm:"comment:创建时间"`
	UpdatedAt        time.Time      `gorm:"comment:更新时间"`
	DeletedAt        gorm.DeletedAt `gorm:"softDelete:delete_at;index;comment:软删除时间"`
}

func (AuditFlightOrder) TableName() string {
	return "audit_flight_orders"
}

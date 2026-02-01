package audit

import (
	"time"

	"gorm.io/gorm"
)

// 3. 酒店审核明细表（audit_hotel_orders）- 酒店专属审核信息
type AuditHotelOrder struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement;comment:酒店审核明细主键ID"`
	AuditMainId     uint64         `gorm:"not null;index:idx_audit_main_id;comment:关联审核主表ID"`
	BusinessRelId   uint64         `gorm:"not null;index:idx_business_rel_id;comment:关联业务ID（订单ID或酒店ID）"`
	HotelId         uint64         `gorm:"not null;comment:酒店ID"`
	HotelName       string         `gorm:"type:varchar(100);not null;comment:酒店名称"`
	HotelAddress    string         `gorm:"type:varchar(255);not null;comment:酒店地址"`
	RoomType        string         `gorm:"type:varchar(50);default:'';comment:房间类型"`
	CheckInTime     *time.Time     `gorm:"comment:入住时间"`
	CheckOutTime    *time.Time     `gorm:"comment:退房时间"`
	GuestName       string         `gorm:"type:varchar(50);default:'';comment:入住人姓名"`
	GuestIdCard     string         `gorm:"type:varchar(18);default:'';comment:入住人身份证号（脱敏存储）"`
	OrderAmount     float64        `gorm:"type:decimal(10,2);default:0.00;comment:订单金额（元）"`
	BusinessLicense string         `gorm:"type:varchar(50);default:'';comment:营业执照编号（入驻审核用）"`
	ApplyReason     string         `gorm:"type:varchar(255);not null;comment:审核申请原因"`
	HotelExtra      string         `gorm:"type:json;comment:酒店扩展字段"`
	CreatedAt       time.Time      `gorm:"comment:创建时间"`
	UpdatedAt       time.Time      `gorm:"comment:更新时间"`
	DeletedAt       gorm.DeletedAt `gorm:"softDelete:delete_at;comment:软删除时间"`
}


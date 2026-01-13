package model

import (
	"time"
)

const TableNameTicketType = "ticket_type"

// TicketType 门票类型配置表-含退改规则/库存/价格，下单核心关联表
type TicketType struct {
	BaseModel

	SpotID        uint64    `gorm:"column:spot_id;type:BIGINT UNSIGNED;not null;index:idx_spot_id;comment:所属景点ID，关联spot_info.id" json:"spot_id"`
	TicketName    string    `gorm:"column:ticket_name;type:VARCHAR(100);not null;comment:门票名称（如成人票、儿童票、套票）" json:"ticket_name"`
	Price         int64     `gorm:"column:price;type:BIGINT UNSIGNED;not null;comment:门票售价" json:"price"`
	OriginalPrice int64     `gorm:"column:original_price;type:BIGINT UNSIGNED;not null;comment:门票原价" json:"original_price"`
	Stock         uint32    `gorm:"column:stock;type:INT UNSIGNED;not null;default:0;comment:门票库存数量" json:"stock"`
	Version       uint32    `gorm:"column:version;type:INT UNSIGNED;not null;default:1;comment:乐观锁版本号，扣库存必用，防超卖核心字段" json:"version"`
	ValidStart    time.Time `gorm:"column:valid_start_time;type:DATE;not null;comment:门票有效期开始时间" json:"valid_start_time"`
	ValidEnd      time.Time `gorm:"column:valid_end_time;type:DATE;not null;comment:门票有效期结束时间" json:"valid_end_time"`
	TicketStatus  string    `gorm:"column:ticket_status;type:VARCHAR(20);not null;default:ON_SALE;index:idx_ticket_status;comment:门票状态：ON_SALE-在售，OFF_SALE-下架，STOCK_OUT-售罄" json:"ticket_status"`
	RefundRule    string    `gorm:"column:refund_rule;type:TEXT;not null;comment:退改规则" json:"refund_rule"`
	UseRule       string    `gorm:"column:use_rule;type:TEXT;not null;comment:使用规则" json:"use_rule"`
	ExtFields     *JSON     `gorm:"column:ext_fields;type:JSON;comment:扩展字段，如适用人群、免票政策等" json:"ext_fields,omitempty"`

	Spot *SpotInfo `gorm:"foreignKey:SpotID;references:ID" json:"spot,omitempty"`
}

func (TicketType) TableName() string { return TableNameTicketType }

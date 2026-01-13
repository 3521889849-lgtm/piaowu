package model

const TableNameOrderItem = "order_item"

// OrderItem 订单详情表-主订单的明细
type OrderItem struct {
	BaseModel

	OrderID    uint64 `gorm:"column:order_id;type:BIGINT UNSIGNED;not null;index:idx_order_id;comment:关联主订单ID" json:"order_id"`
	TicketType uint64 `gorm:"column:ticket_type_id;type:BIGINT UNSIGNED;not null;index:idx_ticket_type_id;comment:关联门票类型ID" json:"ticket_type_id"`
	TravelerID uint64 `gorm:"column:traveler_id;type:BIGINT UNSIGNED;not null;index:idx_traveler_id;comment:关联出行人ID" json:"traveler_id"`
	TicketName string `gorm:"column:ticket_name;type:VARCHAR(100);not null;comment:门票名称（冗余存储）" json:"ticket_name"`
	Single     int64  `gorm:"column:single_price;type:BIGINT UNSIGNED;not null;comment:单张门票价格" json:"single_price"`
	TicketNum  uint8  `gorm:"column:ticket_num;type:TINYINT UNSIGNED;not null;default:1;comment:购票数量" json:"ticket_num"`
	ExtFields  *JSON  `gorm:"column:ext_fields;type:JSON;comment:扩展字段" json:"ext_fields,omitempty"`
}

func (OrderItem) TableName() string { return TableNameOrderItem }

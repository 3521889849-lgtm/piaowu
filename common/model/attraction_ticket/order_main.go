package model

import (
	"time"
)

const TableNameOrderMain = "order_main"

// OrderMain 主订单表-核心业务表，订单状态流转全量记录
type OrderMain struct {
	BaseModel

	OrderNo      string     `gorm:"column:order_no;type:VARCHAR(32);not null;uniqueIndex:uk_order_no;comment:订单编号" json:"order_no"`
	UserID       uint64     `gorm:"column:user_id;type:BIGINT UNSIGNED;not null;index:idx_user_id;comment:下单用户ID" json:"user_id"`
	MerchantID   uint64     `gorm:"column:merchant_id;type:BIGINT UNSIGNED;not null;index:idx_merchant_id;comment:所属商家ID" json:"merchant_id"`
	SpotID       uint64     `gorm:"column:spot_id;type:BIGINT UNSIGNED;not null;index:idx_spot_id;comment:所属景点ID" json:"spot_id"`
	TotalAmount  int64      `gorm:"column:total_amount;type:BIGINT UNSIGNED;not null;comment:订单总金额" json:"total_amount"`
	PayAmount    int64      `gorm:"column:pay_amount;type:BIGINT UNSIGNED;not null;comment:实际支付金额" json:"pay_amount"`
	CouponID     uint64     `gorm:"column:coupon_id;type:BIGINT UNSIGNED;not null;default:0;comment:使用的优惠券ID，0=未使用" json:"coupon_id"`
	OrderStatus  string     `gorm:"column:order_status;type:VARCHAR(30);not null;default:DRAFT;index:idx_order_status;comment:订单状态" json:"order_status"`
	PayType      *string    `gorm:"column:pay_type;type:VARCHAR(20);comment:支付方式：WECHAT/ALIPAY" json:"pay_type,omitempty"`
	PayTime      *time.Time `gorm:"column:pay_time;type:DATETIME;comment:支付时间" json:"pay_time,omitempty"`
	VerifyCode   *string    `gorm:"column:verify_code;type:VARCHAR(64);uniqueIndex:uk_verify_code;comment:门票核销码，唯一" json:"verify_code,omitempty"`
	VerifyTime   *time.Time `gorm:"column:verify_time;type:DATETIME;comment:核销使用时间" json:"verify_time,omitempty"`
	CancelTime   *time.Time `gorm:"column:cancel_time;type:DATETIME;comment:订单取消时间" json:"cancel_time,omitempty"`
	RefundAmount int64      `gorm:"column:refund_amount;type:BIGINT UNSIGNED;not null;default:0;comment:退款金额" json:"refund_amount"`
	RefundTime   *time.Time `gorm:"column:refund_time;type:DATETIME;comment:退款完成时间" json:"refund_time,omitempty"`
	ExtFields    *JSON      `gorm:"column:ext_fields;type:JSON;comment:扩展字段" json:"ext_fields,omitempty"`

	Items []OrderItem `gorm:"foreignKey:OrderID;references:ID" json:"items,omitempty"`
}

func (OrderMain) TableName() string { return TableNameOrderMain }

package model

import (
	"time"
)

const TableNameUserCoupon = "user_coupon"

// UserCoupon 用户优惠券表-用户领取后存储
type UserCoupon struct {
	BaseModel

	UserID       uint64     `gorm:"column:user_id;type:BIGINT UNSIGNED;not null;index:idx_user_id;comment:用户ID" json:"user_id"`
	CouponID     uint64     `gorm:"column:coupon_id;type:BIGINT UNSIGNED;not null;index:idx_coupon_id;comment:优惠券ID" json:"coupon_id"`
	CouponName   string     `gorm:"column:coupon_name;type:VARCHAR(100);not null;comment:优惠券名称（冗余）" json:"coupon_name"`
	Denomination int64      `gorm:"column:denomination;type:BIGINT UNSIGNED;not null;comment:优惠券面额" json:"denomination"`
	MinUseAmount int64      `gorm:"column:min_use_amount;type:BIGINT UNSIGNED;not null;comment:最低使用金额" json:"min_use_amount"`
	ValidStart   time.Time  `gorm:"column:valid_start_time;type:DATETIME;not null;comment:有效期开始时间" json:"valid_start_time"`
	ValidEnd     time.Time  `gorm:"column:valid_end_time;type:DATETIME;not null;comment:有效期结束时间" json:"valid_end_time"`
	UseStatus    string     `gorm:"column:use_status;type:VARCHAR(20);not null;default:UNUSED;index:idx_use_status;comment:使用状态：UNUSED/USED/EXPIRED" json:"use_status"`
	OrderID      uint64     `gorm:"column:order_id;type:BIGINT UNSIGNED;not null;default:0;index:idx_order_id;comment:使用的订单ID，0=未使用" json:"order_id"`
	UseTime      *time.Time `gorm:"column:use_time;type:DATETIME;comment:使用时间" json:"use_time,omitempty"`
}

func (UserCoupon) TableName() string { return TableNameUserCoupon }

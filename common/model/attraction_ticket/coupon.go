package model

import (
	"time"
)

const TableNameCoupon = "coupon"

// Coupon 优惠券配置表-管理端配置优惠券
type Coupon struct {
	BaseModel

	CouponName   string    `gorm:"column:coupon_name;type:VARCHAR(100);not null;comment:优惠券名称" json:"coupon_name"`
	CouponType   string    `gorm:"column:coupon_type;type:VARCHAR(20);not null;comment:优惠券类型：FIXED/ DISCOUNT" json:"coupon_type"`
	Denomination int64     `gorm:"column:denomination;type:BIGINT UNSIGNED;not null;comment:优惠券面额/折扣率（以分为单位）" json:"denomination"`
	MinUseAmount int64     `gorm:"column:min_use_amount;type:BIGINT UNSIGNED;not null;comment:最低使用金额（以分为单位）" json:"min_use_amount"`
	ValidStart   time.Time `gorm:"column:valid_start_time;type:DATETIME;not null;index:idx_valid_time,priority:1;comment:有效期开始时间" json:"valid_start_time"`
	ValidEnd     time.Time `gorm:"column:valid_end_time;type:DATETIME;not null;index:idx_valid_time,priority:2;comment:有效期结束时间" json:"valid_end_time"`
	Stock        uint32    `gorm:"column:stock;type:INT UNSIGNED;not null;default:0;comment:优惠券库存" json:"stock"`
	ApplySpotIDs *string   `gorm:"column:apply_spot_ids;type:VARCHAR(512);comment:适用景点ID集合" json:"apply_spot_ids,omitempty"`
	CouponStatus string    `gorm:"column:coupon_status;type:VARCHAR(20);not null;default:VALID;index:idx_coupon_status;comment:状态：VALID-有效，INVALID-失效" json:"coupon_status"`
	ExtFields    *JSON     `gorm:"column:ext_fields;type:JSON;comment:扩展字段" json:"ext_fields,omitempty"`
}

func (Coupon) TableName() string { return TableNameCoupon }

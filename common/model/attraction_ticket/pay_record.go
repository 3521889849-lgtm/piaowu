package model

import (
	"time"
)

const TableNamePayRecord = "pay_record"

// PayRecord 支付记录表-支付回调、退款的核心流水
type PayRecord struct {
	BaseModel

	OrderID          uint64     `gorm:"column:order_id;type:BIGINT UNSIGNED;not null;index:idx_order_id;comment:关联订单ID" json:"order_id"`
	OrderNo          string     `gorm:"column:order_no;type:VARCHAR(32);not null;comment:订单编号（冗余）" json:"order_no"`
	PayType          string     `gorm:"column:pay_type;type:VARCHAR(20);not null;comment:支付方式：WECHAT/ALIPAY" json:"pay_type"`
	PayAmount        int64      `gorm:"column:pay_amount;type:BIGINT UNSIGNED;not null;comment:支付金额" json:"pay_amount"`
	PayStatus        string     `gorm:"column:pay_status;type:VARCHAR(20);not null;index:idx_pay_status;comment:支付状态：SUCCESS/FAIL/REFUND/REFUNDING" json:"pay_status"`
	PlatformTradeNo  *string    `gorm:"column:platform_trade_no;type:VARCHAR(64);index:idx_platform_trade_no;comment:支付平台流水号" json:"platform_trade_no,omitempty"`
	PlatformRefundNo *string    `gorm:"column:platform_refund_no;type:VARCHAR(64);comment:支付平台退款单号" json:"platform_refund_no,omitempty"`
	NotifyTime       *time.Time `gorm:"column:notify_time;type:DATETIME;comment:支付平台回调时间" json:"notify_time,omitempty"`
	ExtFields        *JSON      `gorm:"column:ext_fields;type:JSON;comment:扩展字段" json:"ext_fields,omitempty"`
}

func (PayRecord) TableName() string { return TableNamePayRecord }

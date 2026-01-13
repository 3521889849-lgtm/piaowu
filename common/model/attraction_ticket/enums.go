package model

const (
	OrderStatusDraft          = "DRAFT"
	OrderStatusPendingPayment = "PENDING_PAYMENT"
	OrderStatusPaid           = "PAID"
	OrderStatusUsed           = "USED"
	OrderStatusRefunding      = "REFUNDING"
	OrderStatusRefunded       = "REFUNDED"
	OrderStatusCancelled      = "CANCELLED"
)

const (
	MerchantAuditStatusInitial  = "INITIAL"
	MerchantAuditStatusApproved = "APPROVED"
	MerchantAuditStatusRejected = "REJECTED"
)

const (
	PayTypeWechat = "WECHAT"
	PayTypeAlipay = "ALIPAY"
)

const (
	TicketStatusOnSale   = "ON_SALE"
	TicketStatusOffSale  = "OFF_SALE"
	TicketStatusStockOut = "STOCK_OUT"
)

const (
	PayStatusSuccess   = "SUCCESS"
	PayStatusFail      = "FAIL"
	PayStatusRefund    = "REFUND"
	PayStatusRefunding = "REFUNDING"
)

const (
	CouponStatusValid   = "VALID"
	CouponStatusInvalid = "INVALID"
)

const (
	UserCouponUseStatusUnused  = "UNUSED"
	UserCouponUseStatusUsed    = "USED"
	UserCouponUseStatusExpired = "EXPIRED"
)

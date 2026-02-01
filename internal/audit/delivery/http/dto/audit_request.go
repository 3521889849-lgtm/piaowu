package dto

import "time"

// SubmitAuditRequest 提交审核请求
type SubmitAuditRequest struct {
	BusinessType int8   `json:"business_type" binding:"required,oneof=1 2 3"` // 业务类型  ?=车票订单  ?=酒店订单  ?=酒店入驻
	BusinessID   uint64 `json:"business_id" binding:"required"`               // 业务ID
	ApplyReason  string `json:"apply_reason" binding:"required,max=255"`      // 申请原因
	
	// 车票订单专属字段（当business_type=1时必填）
	TicketOrder *TicketOrderInfo `json:"ticket_order,omitempty"`
	
	// 酒店订单专属字段（当business_type=2时必填）
	HotelOrder *HotelOrderInfo `json:"hotel_order,omitempty"`
	
	// 酒店入驻专属字段（当business_type=3时必填）
	HotelSettle *HotelSettleInfo `json:"hotel_settle,omitempty"`
}

// TicketOrderInfo 车票订单信息
type TicketOrderInfo struct {
	TicketOrderID    uint64    `json:"ticket_order_id" binding:"required"`    // 车票订单ID
	TicketType       int8      `json:"ticket_type" binding:"required"`        // 车票类型  ?=高铁  ?=动车  ?=普通火车，4=汽车  ?
	DepartureStation string    `json:"departure_station" binding:"required"`  // 出发  ?
	ArrivalStation   string    `json:"arrival_station" binding:"required"`    // 到达  ?
	DepartureTime    time.Time `json:"departure_time" binding:"required"`     // 发车时间
	PassengerName    string    `json:"passenger_name" binding:"required"`     // 乘客姓名
	PassengerIDCard  string    `json:"passenger_id_card" binding:"required"`  // 乘客身份证号
	OrderAmount      float64   `json:"order_amount" binding:"required,gt=0"`  // 订单金额
	TicketExtra      string    `json:"ticket_extra,omitempty"`                // 扩展字段（JSON  ?
}

// HotelOrderInfo 酒店订单信息
type HotelOrderInfo struct {
	HotelID       uint64     `json:"hotel_id" binding:"required"`          // 酒店ID
	HotelName     string     `json:"hotel_name" binding:"required"`        // 酒店名称
	HotelAddress  string     `json:"hotel_address" binding:"required"`     // 酒店地址
	RoomType      string     `json:"room_type"`                            // 房间类型
	CheckInTime   *time.Time `json:"check_in_time"`                        // 入住时间
	CheckOutTime  *time.Time `json:"check_out_time"`                       // 退房时  ?
	GuestName     string     `json:"guest_name"`                           // 入住人姓  ?
	GuestIDCard   string     `json:"guest_id_card"`                        // 入住人身份证  ?
	OrderAmount   float64    `json:"order_amount" binding:"required,gt=0"` // 订单金额
	HotelExtra    string     `json:"hotel_extra,omitempty"`                // 扩展字段（JSON  ?
}

// HotelSettleInfo 酒店入驻信息
type HotelSettleInfo struct {
	HotelID         uint64 `json:"hotel_id" binding:"required"`          // 酒店ID
	HotelName       string `json:"hotel_name" binding:"required"`        // 酒店名称
	HotelAddress    string `json:"hotel_address" binding:"required"`     // 酒店地址
	BusinessLicense string `json:"business_license" binding:"required"`  // 营业执照编号
	HotelExtra      string `json:"hotel_extra,omitempty"`                // 扩展字段（JSON  ?
}

// QueryAuditRequest 查询审核请求
type QueryAuditRequest struct {
	AuditID      uint64 `form:"audit_id"`                                    // 审核ID
	BusinessType int8   `form:"business_type" binding:"omitempty,oneof=1 2 3"` // 业务类型
	BusinessID   uint64 `form:"business_id"`                                 // 业务ID
	AuditStatus  int8   `form:"audit_status" binding:"omitempty,oneof=1 2 3 4 5"` // 审核状  ?
	Page         int    `form:"page" binding:"omitempty,gt=0"`               // 页码
	PageSize     int    `form:"page_size" binding:"omitempty,gt=0,lte=100"`  // 每页数量
}

// AuditActionRequest 审核操作请求（审核员使用  ?
type AuditActionRequest struct {
	AuditID     uint64 `json:"audit_id" binding:"required"`                    // 审核ID
	Action      string `json:"action" binding:"required,oneof=approve reject"` // 操作：approve=通过，reject=驳回
	AuditRemark string `json:"audit_remark" binding:"max=500"`                 // 审核备注
}


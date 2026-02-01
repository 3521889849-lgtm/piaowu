package dto

import "time"

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`           // 响应码：0=成功，其  ?失败
	Message string      `json:"message"`        // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据
}

// SubmitAuditResponse 提交审核响应
type SubmitAuditResponse struct {
	AuditID      uint64    `json:"audit_id"`      // 审核ID
	BusinessType int8      `json:"business_type"` // 业务类型
	BusinessID   uint64    `json:"business_id"`   // 业务ID
	AuditStatus  int8      `json:"audit_status"`  // 审核状  ?
	SubmitTime   time.Time `json:"submit_time"`   // 提交时间
}

// AuditDetailResponse 审核详情响应
type AuditDetailResponse struct {
	AuditID         uint64     `json:"audit_id"`          // 审核ID
	BusinessType    int8       `json:"business_type"`     // 业务类型
	BusinessID      uint64     `json:"business_id"`       // 业务ID
	AuditStatus     int8       `json:"audit_status"`      // 审核状  ?
	AuditStatusText string     `json:"audit_status_text"` // 审核状态文  ?
	SubmitUserID    uint64     `json:"submit_user_id"`    // 提交人ID
	SubmitUserName  string     `json:"submit_user_name"`  // 提交人名  ?
	AuditUserID     uint64     `json:"audit_user_id"`     // 审核人ID
	AuditUserName   string     `json:"audit_user_name"`   // 审核人名  ?
	AuditRemark     string     `json:"audit_remark"`      // 审核备注
	AuditTime       *time.Time `json:"audit_time"`        // 审核时间
	SubmitTime      time.Time  `json:"submit_time"`       // 提交时间

	// 业务详情（根据业务类型返回不同的数据  ?
	TicketOrder *TicketOrderDetail `json:"ticket_order,omitempty"` // 车票订单详情
	HotelOrder  *HotelOrderDetail  `json:"hotel_order,omitempty"`  // 酒店订单详情
	HotelSettle *HotelSettleDetail `json:"hotel_settle,omitempty"` // 酒店入驻详情
}

// TicketOrderDetail 车票订单详情
type TicketOrderDetail struct {
	TicketOrderID    uint64    `json:"ticket_order_id"`   // 车票订单ID
	TicketType       int8      `json:"ticket_type"`       // 车票类型
	TicketTypeText   string    `json:"ticket_type_text"`  // 车票类型文本
	DepartureStation string    `json:"departure_station"` // 出发  ?
	ArrivalStation   string    `json:"arrival_station"`   // 到达  ?
	DepartureTime    time.Time `json:"departure_time"`    // 发车时间
	PassengerName    string    `json:"passenger_name"`    // 乘客姓名
	PassengerIDCard  string    `json:"passenger_id_card"` // 乘客身份证号（脱敏）
	OrderAmount      float64   `json:"order_amount"`      // 订单金额
	ApplyReason      string    `json:"apply_reason"`      // 申请原因
}

// HotelOrderDetail 酒店订单详情
type HotelOrderDetail struct {
	HotelID      uint64     `json:"hotel_id"`       // 酒店ID
	HotelName    string     `json:"hotel_name"`     // 酒店名称
	HotelAddress string     `json:"hotel_address"`  // 酒店地址
	RoomType     string     `json:"room_type"`      // 房间类型
	CheckInTime  *time.Time `json:"check_in_time"`  // 入住时间
	CheckOutTime *time.Time `json:"check_out_time"` // 退房时  ?
	GuestName    string     `json:"guest_name"`     // 入住人姓  ?
	GuestIDCard  string     `json:"guest_id_card"`  // 入住人身份证号（脱敏  ?
	OrderAmount  float64    `json:"order_amount"`   // 订单金额
	ApplyReason  string     `json:"apply_reason"`   // 申请原因
}

// HotelSettleDetail 酒店入驻详情
type HotelSettleDetail struct {
	HotelID         uint64 `json:"hotel_id"`         // 酒店ID
	HotelName       string `json:"hotel_name"`       // 酒店名称
	HotelAddress    string `json:"hotel_address"`    // 酒店地址
	BusinessLicense string `json:"business_license"` // 营业执照编号
	ApplyReason     string `json:"apply_reason"`     // 申请原因
}

// AuditListResponse 审核列表响应
type AuditListResponse struct {
	Total int64                  `json:"total"` // 总数
	List  []*AuditDetailResponse `json:"list"`  // 列表
}

//// GetAuditStatusText 获取审核状态文  ?
//func GetAuditStatusText(status int8) string {
//	switch status {
//	case 1:
//		return "待审  ?
//	case 2:
//		return "审核  ?
//	case 3:
//		return "通过"
//	case 4:
//		return "驳回"
//	case 5:
//		return "撤销"
//	default:
//		return "未知"
//	}
//}
//
//// GetTicketTypeText 获取车票类型文本
//func GetTicketTypeText(ticketType int8) string {
//	switch ticketType {
//	case 1:
//		return "高铁"
//	case 2:
//		return "动车"
//	case 3:
//		return "普通火  ?
//	case 4:
//		return "汽车  ?
//	default:
//		return "未知"
//	}
//}

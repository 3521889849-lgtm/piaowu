// Package order 是网关侧“订单域”HTTP Handler。
//
// 典型业务链路：
// 1) 前端调用 /api/v1/order/create 创建订单并锁座
// 2) 前端调用 /api/v1/order/pay 生成支付信息（或直接推进）
// 3) 本项目提供 /api/v1/pay/mock_notify 便于本地用 ApiPost 模拟异步回调
// 4) 网关把关键操作委托给 order_service（Kitex RPC），自身仅做鉴权/参数/聚合响应
package order

import (
	"context"
	"encoding/json"
	"example_shop/common/config"
	"example_shop/common/db"
	"example_shop/internal/gateway/http/dto"
	"example_shop/internal/gateway/http/middleware"
	"example_shop/internal/model"
	kitexorder "example_shop/kitex_gen/orderapi"
	"example_shop/kitex_gen/orderapi/orderservice"
	"example_shop/pkg/alipay"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

func isConnRefused(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "connectex") && strings.Contains(s, "actively refused") {
		return true
	}
	if strings.Contains(s, "connection refused") {
		return true
	}
	if strings.Contains(s, "dial tcp") && strings.Contains(s, "refused") {
		return true
	}
	return false
}

// Handler 承载订单域相关的 HTTP 接口（网关侧）。
type Handler struct {
	OrderClient orderservice.Client
}

// CreateOrder 创建订单并锁座。
//
// HTTP 接口：POST /api/v1/order/create
// 入参：dto.CreateOrderHTTPReq（JSON Body）
// 出参：dto.CreateOrderHTTPResp
func (h *Handler) CreateOrder(ctx context.Context, c *app.RequestContext) {
	var req dto.CreateOrderHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}

	var u model.UserInfo
	if err := db.MysqlDB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		c.JSON(500, dto.BaseHTTPResp{Code: 500, Msg: "查询用户信息失败"})
		return
	}
	if !strings.EqualFold(strings.TrimSpace(u.RealNameVerified), "VERIFIED") {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "未实名认证，无法购票"})
		return
	}

	ps := make([]*kitexorder.CreateOrderPassenger, 0, len(req.Passengers))
	for _, p := range req.Passengers {
		realName := strings.TrimSpace(p.RealName)
		idCard := strings.TrimSpace(p.IDCard)
		if p.UseSelf {
			realName = strings.TrimSpace(u.RealName)
			if u.IDCard != nil {
				idCard = strings.TrimSpace(*u.IDCard)
			} else {
				idCard = ""
			}
		} else if p.PassengerID != 0 {
			var pi model.PassengerInfo
			if err := db.MysqlDB.Where("id = ? AND user_id = ?", p.PassengerID, userID).First(&pi).Error; err != nil {
				c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: "乘车人不存在或无权限"})
				return
			}
			realName = strings.TrimSpace(pi.RealName)
			idCard = strings.TrimSpace(pi.IDCard)
		}
		if realName == "" || idCard == "" {
			c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: "乘客实名信息不完整"})
			return
		}
		ps = append(ps, &kitexorder.CreateOrderPassenger{RealName: realName, IdCard: idCard, SeatType: p.SeatType})
	}

	rpcResp, err := h.OrderClient.CreateOrder(ctx, &kitexorder.CreateOrderReq{
		UserId:           userID,
		TrainId:          req.TrainID,
		DepartureStation: req.DepartureStation,
		ArrivalStation:   req.ArrivalStation,
		Passengers:       ps,
	})
	if err != nil {
		if isConnRefused(err) {
			c.JSON(503, dto.BaseHTTPResp{Code: 503, Msg: "订单服务不可用，请先启动 order_service（默认 127.0.0.1:8890）"})
			return
		}
		c.JSON(502, dto.BaseHTTPResp{Code: 502, Msg: "下单失败: " + err.Error()})
		return
	}

	seats := make([]dto.OrderSeatInfoHTTP, 0, len(rpcResp.Seats))
	for _, s := range rpcResp.Seats {
		if s == nil {
			continue
		}
		seats = append(seats, dto.OrderSeatInfoHTTP{SeatID: s.SeatId, SeatType: s.SeatType, CarriageNum: s.CarriageNum, SeatNum: s.SeatNum, SeatPrice: s.SeatPrice})
	}

	c.JSON(200, dto.CreateOrderHTTPResp{
		Code:            rpcResp.BaseResp.Code,
		Msg:             rpcResp.BaseResp.Msg,
		OrderID:         rpcResp.OrderId,
		PayDeadlineUnix: rpcResp.PayDeadlineUnix,
		Seats:           seats,
	})
}

// PayOrder 支付订单（简化版：由后端完成支付并推进出票/状态流转）。
//
// HTTP 接口：POST /api/v1/order/pay
// 入参：dto.PayOrderHTTPReq（JSON Body）
// 出参：dto.PayOrderHTTPResp
func (h *Handler) PayOrder(ctx context.Context, c *app.RequestContext) {
	var req dto.PayOrderHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}

	rpcReq := &kitexorder.PayOrderReq{UserId: userID, OrderId: req.OrderID, PayChannel: req.PayChannel}
	if req.PayNo != "" {
		rpcReq.PayNo = &req.PayNo
	}
	rpcResp, err := h.OrderClient.PayOrder(ctx, rpcReq)
	if err != nil {
		if isConnRefused(err) {
			c.JSON(503, dto.BaseHTTPResp{Code: 503, Msg: "订单服务不可用，请先启动 order_service（默认 127.0.0.1:8890）"})
			return
		}
		c.JSON(502, dto.BaseHTTPResp{Code: 502, Msg: err.Error()})
		return
	}

	payNo := ""
	if rpcResp.PayNo != nil {
		payNo = *rpcResp.PayNo
	}

	resp := dto.PayOrderHTTPResp{Code: rpcResp.BaseResp.Code, Msg: rpcResp.BaseResp.Msg, OrderStatus: rpcResp.OrderStatus, PayStatus: rpcResp.PayStatus, PayNo: payNo}

	// 支付宝支付对接策略：
	// 1) 业务仍以订单服务推进为准：PayOrder 负责把订单推进到 PAYING，并生成/返回 pay_no
	// 2) 网关侧在 pay_channel=ALIPAY 时，生成支付宝收银台 URL（TradeWapPay）
	// 3) 为了让支付宝异步回调能定位到业务订单，这里将 out_trade_no(pay_no) -> (user_id, order_id, amount) 做临时映射存入 Redis
	if rpcResp.BaseResp != nil && rpcResp.BaseResp.Code == 200 && strings.EqualFold(strings.TrimSpace(req.PayChannel), "ALIPAY") && payNo != "" {
		orderResp, err := h.OrderClient.GetOrder(ctx, &kitexorder.GetOrderReq{UserId: userID, OrderId: req.OrderID})
		if err == nil && orderResp != nil && orderResp.BaseResp != nil && orderResp.BaseResp.Code == 200 && orderResp.Order != nil {
			subject := "车票订单-" + req.OrderID
			payURL, err := alipay.PagePayURL(payNo, subject, orderResp.Order.TotalAmount)
			if err == nil {
				resp.PayURL = payURL

				if db.Rdb != nil {
					deadline := time.Unix(orderResp.Order.PayDeadlineUnix, 0)
					ttl := time.Until(deadline)
					if ttl <= 0 || ttl > 24*time.Hour {
						ttl = 2 * time.Hour
					}
					payload, _ := json.Marshal(map[string]string{
						"user_id":  userID,
						"order_id": req.OrderID,
						"pay_no":   payNo,
						"amount":   fmt.Sprintf("%.2f", orderResp.Order.TotalAmount),
					})
					_ = db.Rdb.Set(db.Ctx, "pay:alipay:out_trade_no:"+payNo, string(payload), ttl).Err()
				}
			}
		}
	}

	c.JSON(200, resp)
}

// PayCallback 模拟第三方支付异步回调（用于把“支付结果”通知到后端）。
//
// HTTP 接口：POST /api/v1/pay/callback
// 入参：dto.ConfirmPayHTTPReq（JSON Body）
// 出参：dto.ConfirmPayHTTPResp
func (h *Handler) PayCallback(ctx context.Context, c *app.RequestContext) {
	// 说明：
	// - 支付宝异步通知是 application/x-www-form-urlencoded（或 multipart/form-data）
	// - 支付宝会携带 sign/sign_type 等字段，需要验签通过后才可继续推进业务订单状态
	ct := strings.ToLower(string(c.ContentType()))
	if strings.Contains(ct, "application/x-www-form-urlencoded") || strings.Contains(ct, "multipart/form-data") {
		values := url.Values{}
		c.PostArgs().VisitAll(func(key, value []byte) {
			values.Add(string(key), string(value))
		})
		c.QueryArgs().VisitAll(func(key, value []byte) {
			if values.Get(string(key)) == "" {
				values.Add(string(key), string(value))
			}
		})

		outTradeNo := strings.TrimSpace(values.Get("out_trade_no"))
		tradeStatus := strings.TrimSpace(values.Get("trade_status"))
		if outTradeNo == "" || tradeStatus == "" {
			c.String(400, "failure")
			return
		}
		if appID := strings.TrimSpace(values.Get("app_id")); appID != "" && strings.TrimSpace(config.Cfg.AliPay.AppId) != "" && appID != strings.TrimSpace(config.Cfg.AliPay.AppId) {
			c.String(400, "failure")
			return
		}
		if err := alipay.Verify(values); err != nil {
			c.String(400, "failure")
			return
		}

		if tradeStatus == "TRADE_SUCCESS" || tradeStatus == "TRADE_FINISHED" {
			if db.Rdb == nil {
				c.String(400, "failure")
				return
			}
			raw, err := db.Rdb.Get(db.Ctx, "pay:alipay:out_trade_no:"+outTradeNo).Result()
			if err != nil {
				c.String(400, "failure")
				return
			}
			var meta struct {
				UserID  string `json:"user_id"`
				OrderID string `json:"order_id"`
				PayNo   string `json:"pay_no"`
				Amount  string `json:"amount"`
			}
			_ = json.Unmarshal([]byte(raw), &meta)
			if strings.TrimSpace(meta.UserID) == "" || strings.TrimSpace(meta.OrderID) == "" {
				c.String(400, "failure")
				return
			}
			if meta.Amount != "" {
				amt := strings.TrimSpace(values.Get("total_amount"))
				if amt != "" && strings.TrimSpace(meta.Amount) != amt {
					c.String(400, "failure")
					return
				}
			}

			third := "SUCCESS"
			_, _ = h.OrderClient.ConfirmPay(ctx, &kitexorder.ConfirmPayReq{
				UserId:           meta.UserID,
				OrderId:          meta.OrderID,
				PayNo:            outTradeNo,
				ThirdPartyStatus: &third,
			})
		}

		c.String(200, "success")
		return
	}
	c.String(400, "failure")
}

// MockPayNotify 触发一条本地模拟回调（便于用 ApiPost/Postman 测试支付回调链路）。
//
// HTTP 接口：POST /api/v1/pay/mock_notify
// 入参：dto.ConfirmPayHTTPReq（JSON Body）
// 出参：dto.BaseHTTPResp
func (h *Handler) MockPayNotify(ctx context.Context, c *app.RequestContext) {
	var req dto.ConfirmPayHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}
	req.UserID = userID
	if req.ThirdPartyStatus == "" {
		req.ThirdPartyStatus = "SUCCESS"
	}

	go func() {
		_, _ = h.OrderClient.ConfirmPay(context.Background(), &kitexorder.ConfirmPayReq{UserId: req.UserID, OrderId: req.OrderID, PayNo: req.PayNo, ThirdPartyStatus: &req.ThirdPartyStatus})
	}()

	c.JSON(200, dto.BaseHTTPResp{Code: 200, Msg: "success"})
}

// RefundOrder 退票接口（触发退票与退款流程，具体规则由后端决定）。
//
// HTTP 接口：POST /api/v1/order/refund
// 入参：dto.RefundOrderHTTPReq（JSON Body）
// 出参：dto.RefundOrderHTTPResp
func (h *Handler) RefundOrder(ctx context.Context, c *app.RequestContext) {
	var req dto.RefundOrderHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}

	rpcReq := &kitexorder.RefundOrderReq{UserId: userID, OrderId: req.OrderID}
	if req.Reason != "" {
		rpcReq.Reason = &req.Reason
	}
	rpcResp, err := h.OrderClient.RefundOrder(ctx, rpcReq)
	if err != nil {
		c.JSON(502, dto.BaseHTTPResp{Code: 502, Msg: err.Error()})
		return
	}
	c.JSON(200, dto.RefundOrderHTTPResp{Code: rpcResp.BaseResp.Code, Msg: rpcResp.BaseResp.Msg, RefundAmount: rpcResp.RefundAmount, RefundStatus: rpcResp.RefundStatus, OrderStatus: rpcResp.OrderStatus})
}

// ChangeOrder 改签接口（更换车次/区间，可能生成新订单）。
//
// HTTP 接口：POST /api/v1/order/change
// 入参：dto.ChangeOrderHTTPReq（JSON Body）
// 出参：dto.ChangeOrderHTTPResp
func (h *Handler) ChangeOrder(ctx context.Context, c *app.RequestContext) {
	var req dto.ChangeOrderHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}

	rpcResp, err := h.OrderClient.ChangeOrder(ctx, &kitexorder.ChangeOrderReq{
		UserId:               userID,
		OrderId:              req.OrderID,
		NewTrainId_:          req.NewTrainID,
		NewDepartureStation_: req.NewDepartureStation,
		NewArrivalStation_:   req.NewArrivalStation,
	})
	if err != nil {
		c.JSON(502, dto.BaseHTTPResp{Code: 502, Msg: err.Error()})
		return
	}
	seats := make([]dto.OrderSeatInfoHTTP, 0, len(rpcResp.Seats))
	for _, s := range rpcResp.Seats {
		if s == nil {
			continue
		}
		seats = append(seats, dto.OrderSeatInfoHTTP{SeatID: s.SeatId, SeatType: s.SeatType, CarriageNum: s.CarriageNum, SeatNum: s.SeatNum, SeatPrice: s.SeatPrice})
	}
	resp := dto.ChangeOrderHTTPResp{
		Code:             rpcResp.BaseResp.Code,
		Msg:              rpcResp.BaseResp.Msg,
		OldOrderStatus:   rpcResp.OldOrderStatus,
		NewOrderID:       rpcResp.NewOrderId_,
		NewTotalAmount:   rpcResp.NewTotalAmount_,
		RefundDiffAmount: rpcResp.RefundDiffAmount,
		Seats:            seats,
	}
	c.JSON(200, resp)
}

// CancelOrder 取消订单（释放锁座/占用）。
//
// HTTP 接口：POST /api/v1/order/cancel
// 入参：dto.CancelOrderHTTPReq（JSON Body）
// 出参：dto.BaseHTTPResp
func (h *Handler) CancelOrder(ctx context.Context, c *app.RequestContext) {
	var req dto.CancelOrderHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}

	rpcResp, err := h.OrderClient.CancelOrder(ctx, &kitexorder.CancelOrderReq{UserId: userID, OrderId: req.OrderID})
	if err != nil {
		c.JSON(502, dto.BaseHTTPResp{Code: 502, Msg: err.Error()})
		return
	}
	c.JSON(200, dto.BaseHTTPResp{Code: rpcResp.Code, Msg: rpcResp.Msg})
}

// GetOrder 查询订单详情。
//
// HTTP 接口：GET /api/v1/order/info
// 入参：dto.GetOrderHTTPReq（Query 参数）
// 出参：dto.GetOrderHTTPResp
func (h *Handler) GetOrder(ctx context.Context, c *app.RequestContext) {
	var req dto.GetOrderHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}

	rpcResp, err := h.OrderClient.GetOrder(ctx, &kitexorder.GetOrderReq{UserId: userID, OrderId: req.OrderID})
	if err != nil {
		c.JSON(502, dto.BaseHTTPResp{Code: 502, Msg: err.Error()})
		return
	}

	var orderInfo *dto.OrderInfoHTTP
	if rpcResp.Order != nil {
		orderInfo = &dto.OrderInfoHTTP{
			OrderID:          rpcResp.Order.OrderId,
			UserID:           rpcResp.Order.UserId,
			TrainID:          rpcResp.Order.TrainId,
			DepartureStation: rpcResp.Order.DepartureStation,
			ArrivalStation:   rpcResp.Order.ArrivalStation,
			FromSeq:          rpcResp.Order.FromSeq,
			ToSeq:            rpcResp.Order.ToSeq,
			TotalAmount:      rpcResp.Order.TotalAmount,
			OrderStatus:      rpcResp.Order.OrderStatus,
			PayDeadlineUnix:  rpcResp.Order.PayDeadlineUnix,
			PayTimeUnix:      rpcResp.Order.PayTimeUnix,
			PayChannel:       rpcResp.Order.PayChannel,
			PayNo:            rpcResp.Order.PayNo,
			CreatedAtUnix:    rpcResp.Order.CreatedAtUnix,
		}
	}

	seats := make([]dto.OrderSeatInfoHTTP, 0, len(rpcResp.Seats))
	for _, s := range rpcResp.Seats {
		if s == nil {
			continue
		}
		seats = append(seats, dto.OrderSeatInfoHTTP{SeatID: s.SeatId, SeatType: s.SeatType, CarriageNum: s.CarriageNum, SeatNum: s.SeatNum, SeatPrice: s.SeatPrice})
	}

	c.JSON(200, dto.GetOrderHTTPResp{Code: rpcResp.BaseResp.Code, Msg: rpcResp.BaseResp.Msg, Order: orderInfo, Seats: seats})
}

// ListOrders 查询订单列表（游标分页）。
//
// HTTP 接口：GET /api/v1/order/list
// 入参：dto.ListOrdersHTTPReq（Query 参数）
// 出参：dto.ListOrdersHTTPResp
func (h *Handler) ListOrders(ctx context.Context, c *app.RequestContext) {
	var req dto.ListOrdersHTTPReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, dto.BaseHTTPResp{Code: 400, Msg: err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(401, dto.BaseHTTPResp{Code: 401, Msg: "未登录"})
		return
	}
	if req.UserID != "" && req.UserID != userID {
		c.JSON(403, dto.BaseHTTPResp{Code: 403, Msg: "user_id不匹配"})
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rpcReq := &kitexorder.ListOrdersReq{UserId: userID, Limit: int32(limit)}
	if req.Status != "" {
		rpcReq.Status = &req.Status
	}
	if req.Cursor != "" {
		rpcReq.Cursor = &req.Cursor
	}
	rpcResp, err := h.OrderClient.ListOrders(ctx, rpcReq)
	if err != nil {
		c.JSON(502, dto.BaseHTTPResp{Code: 502, Msg: err.Error()})
		return
	}

	orders := make([]dto.OrderInfoHTTP, 0, len(rpcResp.Orders))
	for _, o := range rpcResp.Orders {
		if o == nil {
			continue
		}
		orders = append(orders, dto.OrderInfoHTTP{
			OrderID:          o.OrderId,
			UserID:           o.UserId,
			TrainID:          o.TrainId,
			DepartureStation: o.DepartureStation,
			ArrivalStation:   o.ArrivalStation,
			FromSeq:          o.FromSeq,
			ToSeq:            o.ToSeq,
			TotalAmount:      o.TotalAmount,
			OrderStatus:      o.OrderStatus,
			PayDeadlineUnix:  o.PayDeadlineUnix,
			PayTimeUnix:      o.PayTimeUnix,
			PayChannel:       o.PayChannel,
			PayNo:            o.PayNo,
			CreatedAtUnix:    o.CreatedAtUnix,
		})
	}

	c.JSON(200, dto.ListOrdersHTTPResp{Code: rpcResp.BaseResp.Code, Msg: rpcResp.BaseResp.Msg, Orders: orders, NextCursor: rpcResp.NextCursor})
}

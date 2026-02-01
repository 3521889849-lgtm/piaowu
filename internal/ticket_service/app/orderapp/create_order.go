package orderapp

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"example_shop/common/db"
	model2 "example_shop/internal/model"
	"example_shop/internal/ticket_service/app/shared"
	"example_shop/internal/ticket_service/app/ticketapp"
	kitexuser "example_shop/kitex_gen/user"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateOrder 下单用例：锁座 + 创建订单（待支付）+ 返回支付截止时间与座位信息。
//
// 业务目标：
// - 给定用户、车次、购票区间（上车站/下车站）、乘客列表（含席别），创建一笔“待支付”的订单
// - 在下单时立即完成“锁座”，避免支付期间被其他人抢走库存
//
// 核心不变量：
// - 同一张座位在同一区间（按站序）在同一时刻只能被一个订单占用（LOCKED 或 SOLD）
// - 下单写入是强一致的：订单/占用/乘客/关联表必须在一个事务里一起成功或一起失败
// - 支持幂等：同一用户对同一车次+区间+乘客证件+席别组合重复提交会命中幂等键，返回原订单
func (s *Service) CreateOrder(ctx context.Context, req *kitexuser.CreateOrderReq) (*kitexuser.CreateOrderResp, error) {
	if strings.TrimSpace(req.UserId) == "" || strings.TrimSpace(req.TrainId) == "" {
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "user_id/train_id不能为空")}, nil
	}
	if strings.TrimSpace(req.DepartureStation) == "" || strings.TrimSpace(req.ArrivalStation) == "" {
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "departure_station/arrival_station不能为空")}, nil
	}
	if len(req.Passengers) == 0 {
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "passengers不能为空")}, nil
	}
	if len(req.Passengers) > 5 {
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "单次最多支持5名乘客")}, nil
	}

	// 1) 读取车次基础信息，用于校验“是否存在/是否已发车”
	var t model2.TrainInfo
	if err := db.MysqlDB.Where("train_id = ?", req.TrainId).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &kitexuser.CreateOrderResp{BaseResp: baseResp(404, "车次不存在")}, nil
		}
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(500, "查询车次失败: "+err.Error())}, nil
	}
	if time.Now().After(t.DepartureTime) {
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "车次已发车，无法下单")}, nil
	}

	// 2) 把“站点名称”转换为“站序区间”（站序用于区间重叠判定）
	//    站序 sequence 是 TrainStationPass 上的自然序，值越小越靠前。
	fromStation := strings.TrimSpace(req.DepartureStation)
	toStation := strings.TrimSpace(req.ArrivalStation)
	var depStop model2.TrainStationPass
	if err := db.MysqlDB.Where("train_id = ? AND station_name = ?", t.ID, fromStation).First(&depStop).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &kitexuser.CreateOrderResp{BaseResp: baseResp(404, "上车站不在该车次途经站列表")}, nil
		}
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(500, "查询上车站失败: "+err.Error())}, nil
	}
	var arrStop model2.TrainStationPass
	if err := db.MysqlDB.Where("train_id = ? AND station_name = ?", t.ID, toStation).First(&arrStop).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &kitexuser.CreateOrderResp{BaseResp: baseResp(404, "下车站不在该车次途经站列表")}, nil
		}
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(500, "查询下车站失败: "+err.Error())}, nil
	}
	if depStop.Sequence >= arrStop.Sequence {
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "上车站必须早于下车站")}, nil
	}

	// 3) 统计每种席别的购票人数，用于按席别分配座位。
	//    need: seat_type -> count
	need := make(map[string]int)
	for _, p := range req.Passengers {
		if p == nil {
			return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "passengers包含空元素")}, nil
		}
		if strings.TrimSpace(p.RealName) == "" || strings.TrimSpace(p.IdCard) == "" {
			return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "乘客 real_name/id_card 不能为空")}, nil
		}
		if !seatTypeAllowed(p.SeatType) {
			return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, "座位类型无效")}, nil
		}
		need[p.SeatType]++
	}

	// 4) 准备订单主键与支付截止时间
	//    - orderID: 业务订单号（UUID）
	//    - payDeadline: 锁座到期时间（下单后默认给 15 分钟完成支付）
	//    - idemKey: 下单幂等键（用于防止前端重试/重复点击导致重复锁座）
	orderID := uuid.New().String()
	payDeadline := time.Now().Add(15 * time.Minute)
	idemKey := buildOrderIdempotentKey(req)

	var allocated []*model2.SeatInfo
	var total float64

	// 5) 事务写入：分配座位 + 写订单 + 写占用 + 写乘客与关联
	//    这里必须用事务把多个表的写入绑定在一起，否则会出现：
	//    - 订单写成功但占用没写成功（超卖）
	//    - 占用写成功但订单没写成功（脏占用，影响后续购买）
	err := db.MysqlDB.Transaction(func(tx *gorm.DB) error {
		allocated = allocated[:0]
		total = 0

		// 5.1 分配座位（票务域能力）：在同一事务内查询可用座位并返回 seat 列表与总价
		//     注意：AllocateSeats 内部会用“区间重叠”规则过滤冲突占用，并对候选 seat 行加锁。
		got, sum, err := ticketapp.AllocateSeats(tx, t.ID, need, depStop.Sequence, arrStop.Sequence, payDeadline)
		if err != nil {
			return err
		}
		allocated = append(allocated, got...)
		total = sum

		// 5.2 先写订单主表（状态：PENDING_PAY）
		//     - total_amount 在拿到 seat 后再更新（也可直接在这里写入 sum，当前实现选择后置更新）
		//     - idempotent_key 用于唯一约束，重复下单直接命中并返回原订单
		order := model2.OrderInfo{
			ID:               orderID,
			UserID:           req.UserId,
			TrainID:          t.ID,
			DepartureStation: fromStation,
			ArrivalStation:   toStation,
			FromSeq:          depStop.Sequence,
			ToSeq:            arrStop.Sequence,
			TotalAmount:      0,
			OrderStatus:      "PENDING_PAY",
			PayDeadline:      nullTime(payDeadline),
			RefundAmount:     0,
			RefundStatus:     "NO_REFUND",
			IdempotentKey:    idemKey,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		if err := tx.Create(&order).Error; err != nil {
			if isDuplicateKeyError(err) {
				return &idempotentHitError{Key: idemKey}
			}
			return err
		}
		writeOrderAuditLog(tx, orderID, "CREATE_ORDER", req.UserId, "", "PENDING_PAY", map[string]any{"train_id": t.ID, "from": fromStation, "to": toStation})

		// 5.3 写入区间占用（锁座）
		//     - 每个乘客对应一张 seat，占用区间为 [from_seq, to_seq)
		//     - 状态 LOCKED 表示“已锁定待支付”；lock_expire_time 到期后可被系统释放
		occs := make([]model2.SeatSegmentOccupancy, 0, len(allocated))
		for _, seat := range allocated {
			occs = append(occs, model2.SeatSegmentOccupancy{
				TrainID:        t.ID,
				SeatID:         seat.ID,
				FromStation:    fromStation,
				ToStation:      toStation,
				FromSeq:        depStop.Sequence,
				ToSeq:          arrStop.Sequence,
				OrderID:        orderID,
				Status:         "LOCKED",
				LockExpireTime: nullTime(payDeadline),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
		}
		if err := tx.CreateInBatches(&occs, 500).Error; err != nil {
			return err
		}

		// 5.4 更新订单金额（避免在创建订单时写 0）
		if err := tx.Model(&model2.OrderInfo{}).Where("order_id = ?", orderID).Updates(map[string]any{"total_amount": shared.Money2(total)}).Error; err != nil {
			return err
		}

		// 5.5 乘客表 + 订单座位关系表
		//     - PassengerInfo 用于查询“历史乘车人”
		//     - OrderSeatRelation 是订单维度的 seat 冗余关系（便于查订单座位与计算退款）
		rel := make([]model2.OrderSeatRelation, 0, len(allocated))
		pass := make([]model2.PassengerInfo, 0, len(req.Passengers))
		seatByType := map[string][]*model2.SeatInfo{}
		for _, seat := range allocated {
			seatByType[seat.SeatType] = append(seatByType[seat.SeatType], seat)
		}
		for _, p := range req.Passengers {
			list := seatByType[p.SeatType]
			seat := list[0]
			seatByType[p.SeatType] = list[1:]
			pass = append(pass, model2.PassengerInfo{
				OrderID:   orderID,
				UserID:    req.UserId,
				RealName:  p.RealName,
				IDCard:    p.IdCard,
				SeatID:    seat.ID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
			rel = append(rel, model2.OrderSeatRelation{
				OrderID:    orderID,
				SeatID:     seat.ID,
				SeatType:   seat.SeatType,
				SeatPrice:  seat.SeatPrice,
				IsRefunded: "NO",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})
		}
		if err := tx.Create(&pass).Error; err != nil {
			return err
		}
		if err := tx.Create(&rel).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if hit, ok := err.(*idempotentHitError); ok {
			return loadIdempotentOrder(hit.Key)
		}
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(400, err.Error())}, nil
	}

	seats := make([]*kitexuser.OrderSeatInfo, 0, len(allocated))
	for _, seat := range allocated {
		seats = append(seats, &kitexuser.OrderSeatInfo{SeatId: seat.ID, SeatType: seat.SeatType, CarriageNum: seat.CarriageNum, SeatNum: seat.SeatNum, SeatPrice: shared.Money2(seat.SeatPrice)})
	}
	return &kitexuser.CreateOrderResp{BaseResp: baseResp(200, "success"), OrderId: orderID, PayDeadlineUnix: payDeadline.Unix(), Seats: seats}, nil
}

func seatTypeAllowed(seatType string) bool {
	switch strings.TrimSpace(seatType) {
	case "硬座", "二等座", "一等座", "商务座", "硬卧", "软卧":
		return true
	default:
		return false
	}
}

type idempotentHitError struct {
	Key string
}

func (e *idempotentHitError) Error() string {
	return "idempotent hit"
}

// buildOrderIdempotentKey 生成“下单幂等键”：同一用户对同一车次、同一区间、同一乘客+席别组合重复提交会命中唯一索引。
func buildOrderIdempotentKey(req *kitexuser.CreateOrderReq) string {
	type p struct {
		IDCard   string `json:"id_card"`
		SeatType string `json:"seat_type"`
	}
	ps := make([]p, 0, len(req.Passengers))
	for _, it := range req.Passengers {
		if it == nil {
			continue
		}
		ps = append(ps, p{IDCard: strings.TrimSpace(it.IdCard), SeatType: strings.TrimSpace(it.SeatType)})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].SeatType == ps[j].SeatType {
			return ps[i].IDCard < ps[j].IDCard
		}
		return ps[i].SeatType < ps[j].SeatType
	})
	b, _ := json.Marshal(ps)
	src := req.UserId + "|" + req.TrainId + "|" + strings.TrimSpace(req.DepartureStation) + "|" + strings.TrimSpace(req.ArrivalStation) + "|" + string(b)
	h := md5.Sum([]byte(src))
	return fmt.Sprintf("%x", h)
}

func loadIdempotentOrder(idemKey string) (*kitexuser.CreateOrderResp, error) {
	var o model2.OrderInfo
	if err := db.ReadDB().Unscoped().Where("idempotent_key = ?", idemKey).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &kitexuser.CreateOrderResp{BaseResp: baseResp(409, "重复请求但未找到原订单")}, nil
		}
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(500, "查询原订单失败: "+err.Error())}, nil
	}
	if o.DeletedAt.Valid {
		return &kitexuser.CreateOrderResp{BaseResp: baseResp(409, "重复请求命中已删除订单，请更换乘客/席别/车次或联系管理员清理幂等键")}, nil
	}
	var rel []model2.OrderSeatRelation
	_ = db.ReadDB().Where("order_id = ?", o.ID).Find(&rel).Error
	ids := make([]string, 0, len(rel))
	for _, r := range rel {
		ids = append(ids, r.SeatID)
	}
	seatDetail := map[string]model2.SeatInfo{}
	if len(ids) > 0 {
		var ss []model2.SeatInfo
		_ = db.ReadDB().Where("seat_id IN ?", ids).Find(&ss).Error
		for _, seat := range ss {
			seatDetail[seat.ID] = seat
		}
	}
	seats := make([]*kitexuser.OrderSeatInfo, 0, len(rel))
	for _, r := range rel {
		d, ok := seatDetail[r.SeatID]
		if ok {
			seats = append(seats, &kitexuser.OrderSeatInfo{SeatId: r.SeatID, SeatType: r.SeatType, CarriageNum: d.CarriageNum, SeatNum: d.SeatNum, SeatPrice: shared.Money2(r.SeatPrice)})
			continue
		}
		seats = append(seats, &kitexuser.OrderSeatInfo{SeatId: r.SeatID, SeatType: r.SeatType, SeatPrice: shared.Money2(r.SeatPrice)})
	}
	deadline := int64(0)
	if o.PayDeadline.Valid {
		deadline = o.PayDeadline.Time.Unix()
	}
	return &kitexuser.CreateOrderResp{BaseResp: baseResp(200, "success"), OrderId: o.ID, PayDeadlineUnix: deadline, Seats: seats}, nil
}

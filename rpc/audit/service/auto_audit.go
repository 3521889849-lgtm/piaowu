package service

import (
	"context"
	"encoding/json"
	"log"
	"piaowu/common/config"
	"piaowu/common/db"
	"piaowu/common/model/audit"
	audit_kitex "piaowu/kitex_gen/audit"
	"piaowu/rpc/audit/component/alert"
	"piaowu/rpc/audit/component/decision"
	"piaowu/rpc/audit/component/metrics"
	"piaowu/rpc/audit/component/ml"
	"piaowu/rpc/audit/component/rule_engine"
	"piaowu/rpc/audit/component/threshold"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

type AutoAuditService struct {
	engine         *rule_engine.RuleEngine
	decisionEngine *decision.Engine
	initOnce       sync.Once
}

var AutoAuditSvc = new(AutoAuditService)

// Init 初始化服务（加载规则引擎/模型/阈  ?监控/告警  ?
func (s *AutoAuditService) Init() {
	s.initOnce.Do(func() {
		if db.DB == nil {
			log.Fatal("Database not initialized")
		}
		s.engine = rule_engine.NewRuleEngine(db.DB)

		// 监控初始  ?
		collector := metrics.NewCollector(config.Cfg.Audit.Metrics.WindowSeconds)
		metrics.DefaultCollector = collector

		// 模型初始  ?
		var model ml.Model
		if config.Cfg.Audit.Model.Enabled {
			model = ml.NewLinearModel(config.Cfg.Audit.Model.Weights, config.Cfg.Audit.Model.Bias)
		}

		// 动态阈值初始化
		thresholdMgr := threshold.NewManager(threshold.Config{
			WindowSize: config.Cfg.Audit.Threshold.WindowSize,
			Percentile: config.Cfg.Audit.Threshold.Percentile,
			Min:        config.Cfg.Audit.Threshold.Min,
			Max:        config.Cfg.Audit.Threshold.Max,
			Default:    config.Cfg.Audit.Threshold.Default,
		})

		// 插件管理器（可扩展）
		plugins := decision.NewPluginManager()
		// 注册黑名单插件
		plugins.Register(decision.NewBlacklistPlugin([]string{"违禁品", "敏感词", "垃圾广告"}))

		// 决策引擎初始化

		s.decisionEngine = decision.NewEngine(
			s.engine,
			model,
			thresholdMgr,
			plugins,
			collector,
			config.Cfg.Audit.Decision,
		)

		// 告警初始  ?
		alertMgr := alert.NewManager(alert.Config{
			Enabled:         config.Cfg.Audit.Alert.Enabled,
			IntervalSeconds: config.Cfg.Audit.Alert.IntervalSeconds,
			MaxLatencyMs:    config.Cfg.Audit.Alert.MaxLatencyMs,
			MinAccuracy:     config.Cfg.Audit.Alert.MinAccuracy,
			MinThroughput:   config.Cfg.Audit.Alert.MinThroughput,
			WebhookURL:      config.Cfg.Audit.Alert.WebhookURL,
		}, collector, log.Default())
		alertMgr.Start(context.Background())
	})
}

// 审核结果结构  ?
type AuditResult struct {
	Status int8   // 审核状  ?
	Remark string // 审核备注
}

// AutoAudit 自动审核入口
func (s *AutoAuditService) AutoAudit(ctx context.Context, req *audit_kitex.ApplyAuditReq) (int64, error) {
	// 0. 确保引擎已初始化
	if s.engine == nil {
		s.Init()
	}

	// 1. 准备 Fact (解析 Content)
	var fact rule_engine.Fact
	if err := json.Unmarshal([]byte(req.Content), &fact); err != nil {
		// 解析失败，直接转人工或拒  ?
		return 0, err
	}
	// 注入基础信息
	fact["biz_type"] = req.BizType.String()
	fact["submitter_id"] = req.SubmitterId
	fact["biz_id"] = req.BizId

	// 2. 执行决策引擎（规  ?模型+动态阈值）
	//   ?BizType 枚举转为字符串作  ?Key (例如 "TICKET_ORDER")
	bizTypeKey := req.BizType.String()
	if s.decisionEngine == nil {
		s.Init()
	}
	decisionResult, err := s.decisionEngine.Decide(ctx, bizTypeKey, fact)
	if err != nil {
		return 0, err
	}

	// 3. 处理结果
	result := AuditResult{
		Status: decisionResult.Status,
		Remark: decisionResult.Remark,
	}
	// 预留扩展字段（模型评  ?阈值等  ?
	extraJSON := decision.BuildExtraJSON(decisionResult.Extra)

	// 4. 持久化审核记录（事务  ?
	var auditMainID uint64
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// 4.1 创建主表记录
		businessID, _ := strconv.ParseUint(req.BizId, 10, 64)
		submitUserID, _ := strconv.ParseUint(req.SubmitterId, 10, 64)

		auditMain := &audit.AuditMain{
			BusinessType:   int8(req.BizType),
			BusinessId:     businessID,
			AuditStatus:    result.Status,
			SubmitUserId:   submitUserID,
			SubmitUserName: "User", // 暂时 mock
			AuditRemark:    result.Remark,
			Extra:          extraJSON, // 记录模型评分/阈值等扩展信息

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if result.Status != int8(audit_kitex.AuditStatus_PENDING) {
			now := time.Now()
			auditMain.AuditTime = &now
			auditMain.AuditUserName = "RuleEngine"
		}

		if err := tx.Create(auditMain).Error; err != nil {
			return err
		}
		auditMainID = auditMain.ID

		// 4.2 创建子表记录（保持原逻辑  ?
		if err := s.createSubRecord(tx, auditMainID, req); err != nil {
			return err
		}

		// 4.3 记录操作日志
		log := &audit.AuditOperationLog{
			AuditMainId:     auditMainID,
			OperatorId:      0, // 0代表系统
			OperatorName:    "System",
			OperationType:   1, // 提交
			OperationRemark: "规则引擎审核：" + result.Remark,
			CreatedAt:       time.Now(),
		}
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return int64(auditMainID), nil
}

// 创建子表记录 (保持不变)
func (s *AutoAuditService) createSubRecord(tx *gorm.DB, mainID uint64, req *audit_kitex.ApplyAuditReq) error {
	switch req.BizType {
	case audit_kitex.BizType_TICKET_ORDER:
		var info map[string]interface{}
		json.Unmarshal([]byte(req.Content), &info)

		// 安全转换 float64 -> uint64
		orderID := uint64(0)
		if v, ok := info["order_id"].(float64); ok {
			orderID = uint64(v)
		}

		ticketOrder := &audit.AuditTicketOrder{
			AuditMainId:      mainID,
			TicketOrderId:    orderID,
			PassengerName:    getString(info, "passenger_name"),
			PassengerIdCard:  getString(info, "passenger_id_card"),
			DepartureStation: getString(info, "departure_station"),
			ArrivalStation:   getString(info, "arrival_station"),
			ApplyReason:      getString(info, "apply_reason"),
			TicketExtra:      "{}", // 修复：MySQL JSON 字段默认值
			CreatedAt:        time.Now(),
		}
		if tStr, ok := info["departure_time"].(string); ok {
			t, _ := time.Parse("2006-01-02 15:04:05", tStr)
			ticketOrder.DepartureTime = t
		} else {
			// 容错：如果没有提供时间，默认给当前时间，避免数据库报错
			ticketOrder.DepartureTime = time.Now()
		}
		if amt, ok := info["order_amount"].(float64); ok {
			ticketOrder.OrderAmount = amt
		}
		return tx.Create(ticketOrder).Error

	case audit_kitex.BizType_HOTEL_ORDER:
		var info map[string]interface{}
		json.Unmarshal([]byte(req.Content), &info)

		orderID := uint64(0)
		if v, ok := info["order_id"].(float64); ok {
			orderID = uint64(v)
		}

		hotelOrder := &audit.AuditHotelOrder{
			AuditMainId:   mainID,
			BusinessRelId: orderID,
			HotelName:     getString(info, "hotel_name"),
			HotelAddress:  getString(info, "hotel_address"),
			GuestName:     getString(info, "guest_name"),
			GuestIdCard:   getString(info, "guest_id_card"),
			CreatedAt:     time.Now(),
		}
		return tx.Create(hotelOrder).Error

	case audit_kitex.BizType_FLIGHT_ORDER:
		// 处理机票订单审核数据
		var info map[string]interface{}
		json.Unmarshal([]byte(req.Content), &info)

		// 解析机票订单ID，JSON中数字类型默认为float64
		orderID := uint64(0)
		if v, ok := info["flight_order_id"].(float64); ok {
			orderID = uint64(v)
		}

		// 解析航班类型（1=国内航班, 2=国际航班）
		flightType := int8(1) // 默认为国内航班
		if v, ok := info["flight_type"].(float64); ok {
			flightType = int8(v)
		}

		// 构建机票审核明细记录
		flightOrder := &audit.AuditFlightOrder{
			AuditMainId:      mainID,                               // 关联审核主表ID
			FlightOrderId:    orderID,                              // 机票订单ID
			FlightType:       flightType,                           // 航班类型
			FlightNo:         getString(info, "flight_no"),         // 航班号（如CA1234）
			Airline:          getString(info, "airline"),           // 航空公司名称
			DepartureAirport: getString(info, "departure_airport"), // 出发机场
			ArrivalAirport:   getString(info, "arrival_airport"),   // 到达机场
			CabinClass:       getString(info, "cabin_class"),       // 舱位等级（经济舱/商务舱/头等舱）
			PassengerName:    getString(info, "passenger_name"),    // 乘客姓名
			PassengerIdCard:  getString(info, "passenger_id_card"), // 乘客身份证号
			PassengerPhone:   getString(info, "passenger_phone"),   // 乘客联系电话
			ApplyReason:      getString(info, "apply_reason"),      // 审核申请原因
			FlightExtra:      "{}",                                 // 扩展字段（JSON格式）
			CreatedAt:        time.Now(),
		}

		// 解析起飞时间（尝试标准时间格式）
		if tStr, ok := info["departure_time"].(string); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", tStr); err == nil {
				flightOrder.DepartureTime = t
			}
		}

		// 解析降落时间
		if tStr, ok := info["arrival_time"].(string); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", tStr); err == nil {
				flightOrder.ArrivalTime = t
			}
		}

		// 解析订单金额
		if amt, ok := info["order_amount"].(float64); ok {
			flightOrder.OrderAmount = amt
		}

		// 插入数据库
		return tx.Create(flightOrder).Error

	case audit_kitex.BizType_SCENIC_ORDER:
		// 处理旅游门票订单审核数据
		var info map[string]interface{}
		json.Unmarshal([]byte(req.Content), &info)

		// 解析门票订单ID
		orderID := uint64(0)
		if v, ok := info["scenic_order_id"].(float64); ok {
			orderID = uint64(v)
		}

		// 解析门票类型（1=景区门票, 2=演出票, 3=展览票, 4=游乐园票）
		ticketType := int8(1) // 默认为景区门票
		if v, ok := info["ticket_type"].(float64); ok {
			ticketType = int8(v)
		}

		// 解析门票数量
		ticketQty := 0
		if v, ok := info["ticket_quantity"].(float64); ok {
			ticketQty = int(v)
		}

		// 构建旅游门票审核明细记录
		scenicOrder := &audit.AuditScenicOrder{
			AuditMainId:    mainID,                             // 关联审核主表ID
			ScenicOrderId:  orderID,                            // 门票订单ID
			TicketType:     ticketType,                         // 门票类型
			ScenicName:     getString(info, "scenic_name"),     // 景区/场馆名称
			ScenicAddress:  getString(info, "scenic_address"),  // 景区地址
			TicketName:     getString(info, "ticket_name"),     // 门票名称（如成人票、儿童票）
			TicketQuantity: ticketQty,                          // 门票数量
			ContactName:    getString(info, "contact_name"),    // 联系人姓名
			ContactPhone:   getString(info, "contact_phone"),   // 联系人电话
			ContactIdCard:  getString(info, "contact_id_card"), // 联系人身份证号
			ApplyReason:    getString(info, "apply_reason"),    // 审核申请原因
			ScenicExtra:    "{}",                               // 扩展字段（JSON格式）
			CreatedAt:      time.Now(),
		}

		// 解析游玩日期（支持两种格式：纯日期 或 日期+时间）
		if tStr, ok := info["visit_date"].(string); ok {
			// 先尝试纯日期格式（YYYY-MM-DD）
			if t, err := time.Parse("2006-01-02", tStr); err == nil {
				scenicOrder.VisitDate = t
			} else if t, err := time.Parse("2006-01-02 15:04:05", tStr); err == nil {
				// 如果失败则尝试日期时间格式
				scenicOrder.VisitDate = t
			}
		}

		// 解析单价
		if amt, ok := info["unit_price"].(float64); ok {
			scenicOrder.UnitPrice = amt
		}

		// 解析订单总金额
		if amt, ok := info["order_amount"].(float64); ok {
			scenicOrder.OrderAmount = amt
		}

		// 插入数据库
		return tx.Create(scenicOrder).Error
	}
	return nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

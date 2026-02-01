package usecase

import (
	"context"
	"errors"
	"piaowu/common/model/audit"
	"piaowu/internal/audit/delivery/http/dto"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AuditUsecase 审核用例
type AuditUsecase struct {
	db *gorm.DB
}

// NewAuditUsecase 创建审核用例
func NewAuditUsecase(db *gorm.DB) *AuditUsecase {
	return &AuditUsecase{
		db: db,
	}
}

// SubmitAudit 提交审核
func (uc *AuditUsecase) SubmitAudit(ctx context.Context, req *dto.SubmitAuditRequest, userID uint64, userName string) (*dto.SubmitAuditResponse, error) {
	// 开启事  ?
	tx := uc.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 创建审核主表记录
	auditMain := &audit.AuditMain{
		BusinessType:   req.BusinessType,
		BusinessId:     req.BusinessID,
		AuditStatus:    1, // 待审  ?
		SubmitUserId:   userID,
		SubmitUserName: userName,
		Extra:          "{}",  // 默认空JSON对象
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := tx.Create(auditMain).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建审核主表记录失败: %w", err)
	}

	// 2. 根据业务类型创建对应的明细表记录
	switch req.BusinessType {
	case 1: // 车票订单
		if req.TicketOrder == nil {
			tx.Rollback()
			return nil, errors.New("车票订单信息不能为空")
		}
		// 处理ticket_extra字段
		ticketExtra := req.TicketOrder.TicketExtra
		if ticketExtra == "" {
			ticketExtra = "{}"
		}
		ticketOrder := &audit.AuditTicketOrder{
			AuditMainId:      auditMain.ID,
			TicketOrderId:    req.TicketOrder.TicketOrderID,
			TicketType:       req.TicketOrder.TicketType,
			DepartureStation: req.TicketOrder.DepartureStation,
			ArrivalStation:   req.TicketOrder.ArrivalStation,
			DepartureTime:    req.TicketOrder.DepartureTime,
			PassengerName:    req.TicketOrder.PassengerName,
			PassengerIdCard:  uc.maskIDCard(req.TicketOrder.PassengerIDCard),
			OrderAmount:      req.TicketOrder.OrderAmount,
			ApplyReason:      req.ApplyReason,
			TicketExtra:      ticketExtra,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		if err := tx.Create(ticketOrder).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建车票审核明细失败: %w", err)
		}

	case 2: // 酒店订单
		if req.HotelOrder == nil {
			tx.Rollback()
			return nil, errors.New("酒店订单信息不能为空")
		}
		// 处理hotel_extra字段，如果为空则设置为空JSON对象
		hotelExtra := req.HotelOrder.HotelExtra
		if hotelExtra == "" {
			hotelExtra = "{}"
		}
		hotelOrder := &audit.AuditHotelOrder{
			AuditMainId:   auditMain.ID,
			BusinessRelId: req.BusinessID,
			HotelId:       req.HotelOrder.HotelID,
			HotelName:     req.HotelOrder.HotelName,
			HotelAddress:  req.HotelOrder.HotelAddress,
			RoomType:      req.HotelOrder.RoomType,
			CheckInTime:   req.HotelOrder.CheckInTime,
			CheckOutTime:  req.HotelOrder.CheckOutTime,
			GuestName:     req.HotelOrder.GuestName,
			GuestIdCard:   uc.maskIDCard(req.HotelOrder.GuestIDCard),
			OrderAmount:   req.HotelOrder.OrderAmount,
			ApplyReason:   req.ApplyReason,
			HotelExtra:    hotelExtra,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := tx.Create(hotelOrder).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建酒店审核明细失败: %w", err)
		}

	case 3: // 酒店入驻
		if req.HotelSettle == nil {
			tx.Rollback()
			return nil, errors.New("酒店入驻信息不能为空")
		}
		// 处理hotel_extra字段
		hotelExtra := req.HotelSettle.HotelExtra
		if hotelExtra == "" {
			hotelExtra = "{}"
		}
		hotelSettle := &audit.AuditHotelOrder{
			AuditMainId:     auditMain.ID,
			BusinessRelId:   req.HotelSettle.HotelID,
			HotelId:         req.HotelSettle.HotelID,
			HotelName:       req.HotelSettle.HotelName,
			HotelAddress:    req.HotelSettle.HotelAddress,
			BusinessLicense: req.HotelSettle.BusinessLicense,
			ApplyReason:     req.ApplyReason,
			HotelExtra:      hotelExtra,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := tx.Create(hotelSettle).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建酒店入驻审核明细失败: %w", err)
		}

	default:
		tx.Rollback()
		return nil, errors.New("不支持的业务类型")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &dto.SubmitAuditResponse{
		AuditID:      auditMain.ID,
		BusinessType: auditMain.BusinessType,
		BusinessID:   auditMain.BusinessId,
		AuditStatus:  auditMain.AuditStatus,
		SubmitTime:   auditMain.CreatedAt,
	}, nil
}

// GetAuditDetail 获取审核详情
func (uc *AuditUsecase) GetAuditDetail(ctx context.Context, auditID uint64) (*dto.AuditDetailResponse, error) {
	var auditMain audit.AuditMain
	if err := uc.db.WithContext(ctx).First(&auditMain, auditID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("审核记录不存  ?)
		}
		return nil, fmt.Errorf("查询审核记录失败: %w", err)
	}

	response := &dto.AuditDetailResponse{
		AuditID:         auditMain.ID,
		BusinessType:    auditMain.BusinessType,
		BusinessID:      auditMain.BusinessId,
		AuditStatus:     auditMain.AuditStatus,
		AuditStatusText: dto.GetAuditStatusText(auditMain.AuditStatus),
		SubmitUserID:    auditMain.SubmitUserId,
		SubmitUserName:  auditMain.SubmitUserName,
		AuditUserID:     auditMain.AuditUserId,
		AuditUserName:   auditMain.AuditUserName,
		AuditRemark:     auditMain.AuditRemark,
		AuditTime:       auditMain.AuditTime,
		SubmitTime:      auditMain.CreatedAt,
	}

	// 根据业务类型查询明细
	switch auditMain.BusinessType {
	case 1: // 车票订单
		var ticketOrder audit.AuditTicketOrder
		if err := uc.db.WithContext(ctx).Where("audit_main_id = ?", auditMain.ID).First(&ticketOrder).Error; err == nil {
			response.TicketOrder = &dto.TicketOrderDetail{
				TicketOrderID:    ticketOrder.TicketOrderId,
				TicketType:       ticketOrder.TicketType,
				TicketTypeText:   dto.GetTicketTypeText(ticketOrder.TicketType),
				DepartureStation: ticketOrder.DepartureStation,
				ArrivalStation:   ticketOrder.ArrivalStation,
				DepartureTime:    ticketOrder.DepartureTime,
				PassengerName:    ticketOrder.PassengerName,
				PassengerIDCard:  ticketOrder.PassengerIdCard,
				OrderAmount:      ticketOrder.OrderAmount,
				ApplyReason:      ticketOrder.ApplyReason,
			}
		}

	case 2, 3: // 酒店订单或酒店入  ?
		var hotelOrder audit.AuditHotelOrder
		if err := uc.db.WithContext(ctx).Where("audit_main_id = ?", auditMain.ID).First(&hotelOrder).Error; err == nil {
			if auditMain.BusinessType == 2 {
				response.HotelOrder = &dto.HotelOrderDetail{
					HotelID:      hotelOrder.HotelId,
					HotelName:    hotelOrder.HotelName,
					HotelAddress: hotelOrder.HotelAddress,
					RoomType:     hotelOrder.RoomType,
					CheckInTime:  hotelOrder.CheckInTime,
					CheckOutTime: hotelOrder.CheckOutTime,
					GuestName:    hotelOrder.GuestName,
					GuestIDCard:  hotelOrder.GuestIdCard,
					OrderAmount:  hotelOrder.OrderAmount,
					ApplyReason:  hotelOrder.ApplyReason,
				}
			} else {
				response.HotelSettle = &dto.HotelSettleDetail{
					HotelID:         hotelOrder.HotelId,
					HotelName:       hotelOrder.HotelName,
					HotelAddress:    hotelOrder.HotelAddress,
					BusinessLicense: hotelOrder.BusinessLicense,
					ApplyReason:     hotelOrder.ApplyReason,
				}
			}
		}
	}

	return response, nil
}

// QueryAuditList 查询审核列表
func (uc *AuditUsecase) QueryAuditList(ctx context.Context, req *dto.QueryAuditRequest) (*dto.AuditListResponse, error) {
	query := uc.db.WithContext(ctx).Model(&audit.AuditMain{})

	// 添加查询条件
	if req.BusinessType > 0 {
		query = query.Where("business_type = ?", req.BusinessType)
	}
	if req.BusinessID > 0 {
		query = query.Where("business_id = ?", req.BusinessID)
	}
	if req.AuditStatus > 0 {
		query = query.Where("audit_status = ?", req.AuditStatus)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	// 分页参数
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	// 查询列表
	var auditMains []audit.AuditMain
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&auditMains).Error; err != nil {
		return nil, fmt.Errorf("查询列表失败: %w", err)
	}

	// 如果没有数据，直接返  ?
	if len(auditMains) == 0 {
		return &dto.AuditListResponse{
			Total: total,
			List:  []*dto.AuditDetailResponse{},
		}, nil
	}

	// 批量查询关联数据
	mainIDs := make([]uint64, len(auditMains))
	for i, main := range auditMains {
		mainIDs[i] = main.ID
	}

	// 批量查询车票订单
	var ticketOrders []audit.AuditTicketOrder
	ticketOrderMap := make(map[uint64]*audit.AuditTicketOrder)
	if len(mainIDs) > 0 {
		uc.db.WithContext(ctx).Where("audit_main_id IN ?", mainIDs).Find(&ticketOrders)
		for i := range ticketOrders {
			ticketOrderMap[ticketOrders[i].AuditMainId] = &ticketOrders[i]
		}
	}

	// 批量查询酒店订单
	var hotelOrders []audit.AuditHotelOrder
	hotelOrderMap := make(map[uint64]*audit.AuditHotelOrder)
	if len(mainIDs) > 0 {
		uc.db.WithContext(ctx).Where("audit_main_id IN ?", mainIDs).Find(&hotelOrders)
		for i := range hotelOrders {
			hotelOrderMap[hotelOrders[i].AuditMainId] = &hotelOrders[i]
		}
	}

	// 转换为响应格  ?
	list := make([]*dto.AuditDetailResponse, 0, len(auditMains))
	for _, main := range auditMains {
		response := &dto.AuditDetailResponse{
			AuditID:         main.ID,
			BusinessType:    main.BusinessType,
			BusinessID:      main.BusinessId,
			AuditStatus:     main.AuditStatus,
			AuditStatusText: dto.GetAuditStatusText(main.AuditStatus),
			SubmitUserID:    main.SubmitUserId,
			SubmitUserName:  main.SubmitUserName,
			AuditUserID:     main.AuditUserId,
			AuditUserName:   main.AuditUserName,
			AuditRemark:     main.AuditRemark,
			AuditTime:       main.AuditTime,
			SubmitTime:      main.CreatedAt,
		}

		// 根据业务类型填充明细
		switch main.BusinessType {
		case 1: // 车票订单
			if ticketOrder, ok := ticketOrderMap[main.ID]; ok {
				response.TicketOrder = &dto.TicketOrderDetail{
					TicketOrderID:    ticketOrder.TicketOrderId,
					TicketType:       ticketOrder.TicketType,
					TicketTypeText:   dto.GetTicketTypeText(ticketOrder.TicketType),
					DepartureStation: ticketOrder.DepartureStation,
					ArrivalStation:   ticketOrder.ArrivalStation,
					DepartureTime:    ticketOrder.DepartureTime,
					PassengerName:    ticketOrder.PassengerName,
					PassengerIDCard:  ticketOrder.PassengerIdCard,
					OrderAmount:      ticketOrder.OrderAmount,
					ApplyReason:      ticketOrder.ApplyReason,
				}
			}

		case 2: // 酒店订单
			if hotelOrder, ok := hotelOrderMap[main.ID]; ok {
				response.HotelOrder = &dto.HotelOrderDetail{
					HotelID:      hotelOrder.HotelId,
					HotelName:    hotelOrder.HotelName,
					HotelAddress: hotelOrder.HotelAddress,
					RoomType:     hotelOrder.RoomType,
					CheckInTime:  hotelOrder.CheckInTime,
					CheckOutTime: hotelOrder.CheckOutTime,
					GuestName:    hotelOrder.GuestName,
					GuestIDCard:  hotelOrder.GuestIdCard,
					OrderAmount:  hotelOrder.OrderAmount,
					ApplyReason:  hotelOrder.ApplyReason,
				}
			}

		case 3: // 酒店入驻
			if hotelOrder, ok := hotelOrderMap[main.ID]; ok {
				response.HotelSettle = &dto.HotelSettleDetail{
					HotelID:         hotelOrder.HotelId,
					HotelName:       hotelOrder.HotelName,
					HotelAddress:    hotelOrder.HotelAddress,
					BusinessLicense: hotelOrder.BusinessLicense,
					ApplyReason:     hotelOrder.ApplyReason,
				}
			}
		}

		list = append(list, response)
	}

	return &dto.AuditListResponse{
		Total: total,
		List:  list,
	}, nil
}

// ProcessAudit 处理审核（审核员操作  ?
func (uc *AuditUsecase) ProcessAudit(ctx context.Context, req *dto.AuditActionRequest, auditorID uint64, auditorName string) error {
	// 查询审核记录
	var auditMain audit.AuditMain
	if err := uc.db.WithContext(ctx).First(&auditMain, req.AuditID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("审核记录不存  ?)
		}
		return fmt.Errorf("查询审核记录失败: %w", err)
	}

	// 检查审核状  ?
	if auditMain.AuditStatus != 1 && auditMain.AuditStatus != 2 {
		return errors.New("该审核记录不能被处理")
	}

	// 更新审核状  ?
	now := time.Now()
	updates := map[string]interface{}{
		"audit_user_id":   auditorID,
		"audit_user_name": auditorName,
		"audit_remark":    req.AuditRemark,
		"audit_time":      &now,
		"updated_at":      now,
	}

	if req.Action == "approve" {
		updates["audit_status"] = 3 // 通过
	} else if req.Action == "reject" {
		updates["audit_status"] = 4 // 驳回
	} else {
		return errors.New("无效的审核操  ?)
	}

	if err := uc.db.WithContext(ctx).Model(&auditMain).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新审核状态失  ? %w", err)
	}

	return nil
}

// maskIDCard 身份证号脱敏
func (uc *AuditUsecase) maskIDCard(idCard string) string {
	if len(idCard) < 8 {
		return idCard
	}
	return idCard[:4] + "**********" + idCard[len(idCard)-4:]
}

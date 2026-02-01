package handler

import (
	"context"
	"errors"
	"piaowu/internal/audit/delivery/http/dto"
	"piaowu/internal/audit/usecase"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditHandler 审核处理  ?
type AuditHandler struct {
	auditUsecase *usecase.AuditUsecase
}

// NewAuditHandler 创建审核处理  ?
func NewAuditHandler(auditUsecase *usecase.AuditUsecase) *AuditHandler {
	return &AuditHandler{
		auditUsecase: auditUsecase,
	}
}

// SubmitAudit 提交审核
// @Summary 提交审核
// @Description 外部系统提交审核申请，支持车票订单、酒店订单、酒店入驻三种业务类  ?
// @Tags 审核管理
// @Accept json
// @Produce json
// @Param request body dto.SubmitAuditRequest true "审核请求"
// @Success 200 {object} dto.Response{data=dto.SubmitAuditResponse}
// @Failure 400 {object} dto.Response
// @Router /api/v1/audit/submit [post]
func (h *AuditHandler) SubmitAudit(c *gin.Context) {
	var req dto.SubmitAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证业务类型对应的数据是否完  ?
	if err := h.validateBusinessData(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	// 从上下文获取用户信息（这里简化处理，实际应该从JWT或Session中获取）
	userID := h.getUserID(c)
	userName := h.getUserName(c)

	// 提交审核
	resp, err := h.auditUsecase.SubmitAudit(c.Request.Context(), &req, userID, userName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "提交审核失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: "提交成功",
		Data:    resp,
	})
}

// GetAuditDetail 获取审核详情
// @Summary 获取审核详情
// @Description 根据审核ID获取审核详情
// @Tags 审核管理
// @Accept json
// @Produce json
// @Param audit_id path int true "审核ID"
// @Success 200 {object} dto.Response{data=dto.AuditDetailResponse}
// @Failure 400 {object} dto.Response
// @Router /api/v1/audit/{audit_id} [get]
func (h *AuditHandler) GetAuditDetail(c *gin.Context) {
	auditIDStr := c.Param("audit_id")
	auditID, err := strconv.ParseUint(auditIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "审核ID格式错误",
		})
		return
	}

	resp, err := h.auditUsecase.GetAuditDetail(c.Request.Context(), auditID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: "查询成功",
		Data:    resp,
	})
}

// QueryAuditList 查询审核列表
// @Summary 查询审核列表
// @Description 根据条件查询审核列表
// @Tags 审核管理
// @Accept json
// @Produce json
// @Param business_type query int false "业务类型  ?=车票订单  ?=酒店订单  ?=酒店入驻"
// @Param business_id query int false "业务ID"
// @Param audit_status query int false "审核状态：1=待审核，2=审核中，3=通过  ?=驳回  ?=撤销"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.AuditListResponse}
// @Failure 400 {object} dto.Response
// @Router /api/v1/audit/list [get]
func (h *AuditHandler) QueryAuditList(c *gin.Context) {
	var req dto.QueryAuditRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 创建一个独立的context，不受HTTP请求超时影响
	// 设置90秒超时，足够完成数据库查  ?
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	resp, err := h.auditUsecase.QueryAuditList(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: "查询成功",
		Data:    resp,
	})
}

// ProcessAudit 处理审核（审核员操作  ?
// @Summary 处理审核
// @Description 审核员审核操作，可以通过或驳回审  ?
// @Tags 审核管理
// @Accept json
// @Produce json
// @Param request body dto.AuditActionRequest true "审核操作请求"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Router /api/v1/audit/process [post]
func (h *AuditHandler) ProcessAudit(c *gin.Context) {
	var req dto.AuditActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 从上下文获取审核员信  ?
	auditorID := h.getUserID(c)
	auditorName := h.getUserName(c)

	// 处理审核
	if err := h.auditUsecase.ProcessAudit(c.Request.Context(), &req, auditorID, auditorName); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "审核处理失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: "审核处理成功",
	})
}

// validateBusinessData 验证业务数据完整  ?
func (h *AuditHandler) validateBusinessData(req *dto.SubmitAuditRequest) error {
	switch req.BusinessType {
	case 1: // 车票订单
		if req.TicketOrder == nil {
			return errors.New("车票订单信息不能为空")
		}
	case 2: // 酒店订单
		if req.HotelOrder == nil {
			return errors.New("酒店订单信息不能为空")
		}
	case 3: // 酒店入驻
		if req.HotelSettle == nil {
			return errors.New("酒店入驻信息不能为空")
		}
	}
	return nil
}

// getUserID 获取用户ID（从上下文中获取，实际应该从JWT或Session中解析）
func (h *AuditHandler) getUserID(c *gin.Context) uint64 {
	// 这里简化处理，实际应该从JWT token或Session中获  ?
	// 示例：从header中获  ?
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		return 0
	}
	userID, _ := strconv.ParseUint(userIDStr, 10, 64)
	return userID
}

// getUserName 获取用户名（从上下文中获取）
func (h *AuditHandler) getUserName(c *gin.Context) string {
	// 这里简化处理，实际应该从JWT token或Session中获  ?
	// 示例：从header中获  ?
	userName := c.GetHeader("X-User-Name")
	if userName == "" {
		return "匿名用户"
	}
	return userName
}


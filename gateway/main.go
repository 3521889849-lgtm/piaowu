package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"piaowu/kitex_gen/audit"
	"piaowu/kitex_gen/audit/auditservice"

	"github.com/cloudwego/kitex/client"
	"github.com/gin-gonic/gin"
)

var auditClient auditservice.Client

func main() {
	// 初始化 Kitex RPC 客户端，连接到审核服务
	var err error
	auditClient, err = auditservice.NewClient("audit_service", client.WithHostPorts("localhost:8889"))
	if err != nil {
		log.Fatal("❌ Failed to create audit client:", err)
	}

	// 创建 Gin HTTP 服务器
	r := gin.Default()

	// 启用 CORS 跨域支持
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API 路由组
	api := r.Group("/api/audit")
	{
		api.POST("/submit", handleSubmitAudit)
		api.GET("/list", handleQueryAuditList)
		api.POST("/process", handleProcessAudit)
	}

	log.Println("✅ HTTP Gateway 启动成功，监听端口 :8080")
	log.Println("   - POST /api/audit/submit - 提交审核")
	log.Println("   - GET  /api/audit/list   - 查询审核列表")
	log.Println("   - POST /api/audit/process - 处理审核")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("❌ Failed to start HTTP server:", err)
	}
}

// HTTP 响应结构
type HTTPResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 提交审核HTTP请求
type SubmitAuditHTTPReq struct {
	BusinessType int32                  `json:"business_type"`
	BusinessID   string                 `json:"business_id"`
	ApplyReason  string                 `json:"apply_reason"`
	FlightOrder  *FlightOrderInfo       `json:"flight_order,omitempty"`
	ScenicOrder  *ScenicOrderInfo       `json:"scenic_order,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

type FlightOrderInfo struct {
	FlightNo         string  `json:"flight_no"`
	Airline          string  `json:"airline"`
	DepartureAirport string  `json:"departure_airport"`
	ArrivalAirport   string  `json:"arrival_airport"`
	DepartureTime    string  `json:"departure_time"`
	PassengerName    string  `json:"passenger_name"`
	OrderAmount      float64 `json:"order_amount"`
}

type ScenicOrderInfo struct {
	ScenicName    string  `json:"scenic_name"`
	ScenicAddress string  `json:"scenic_address"`
	TicketName    string  `json:"ticket_name"`
	VisitDate     string  `json:"visit_date"`
	OrderAmount   float64 `json:"order_amount"`
	ContactName   string  `json:"contact_name"`
}

// handleSubmitAudit 处理提交审核请求
func handleSubmitAudit(c *gin.Context) {
	var httpReq SubmitAuditHTTPReq
	if err := c.ShouldBindJSON(&httpReq); err != nil {
		c.JSON(http.StatusBadRequest, HTTPResponse{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// 将业务数据序列化为JSON字符串
	contentData := make(map[string]interface{})
	contentData["apply_reason"] = httpReq.ApplyReason
	if httpReq.FlightOrder != nil {
		contentData["flight_order"] = httpReq.FlightOrder
	}
	if httpReq.ScenicOrder != nil {
		contentData["scenic_order"] = httpReq.ScenicOrder
	}
	if httpReq.Extra != nil {
		contentData["extra"] = httpReq.Extra
	}

	contentJSON, _ := json.Marshal(contentData)

	// 构建 RPC 请求
	rpcReq := &audit.ApplyAuditReq{
		BizType:     audit.BizType(httpReq.BusinessType),
		BizId:       httpReq.BusinessID,
		SubmitterId: "web_user_1", // TODO: 从会话中获取
		Content:     string(contentJSON),
		Attachments: []string{},
		Extra:       map[string]string{},
	}

	// 调用 RPC 服务
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := auditClient.ApplyAudit(ctx, rpcReq)
	if err != nil {
		log.Printf("❌ RPC call failed: %v", err)
		c.JSON(http.StatusInternalServerError, HTTPResponse{
			Code:    500,
			Message: "Failed to submit audit: " + err.Error(),
		})
		return
	}

	if resp.BaseResp.Code != 0 {
		c.JSON(http.StatusOK, HTTPResponse{
			Code:    int(resp.BaseResp.Code),
			Message: resp.BaseResp.Msg,
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, HTTPResponse{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"audit_id":      resp.AuditId,
			"business_type": httpReq.BusinessType,
			"business_id":   httpReq.BusinessID,
			"audit_status":  1, // PENDING
			"submit_time":   time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}

// handleQueryAuditList 处理查询审核列表请求
func handleQueryAuditList(c *gin.Context) {
	// 解析查询参数
	pageNum := int32(1)
	pageSize := int32(10)

	if page := c.Query("page"); page != "" {
		if p, err := strconv.ParseInt(page, 10, 32); err == nil {
			pageNum = int32(p)
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if size, err := strconv.ParseInt(ps, 10, 32); err == nil {
			pageSize = int32(size)
		}
	}

	status := audit.AuditStatus_PENDING
	if auditStatus := c.Query("audit_status"); auditStatus != "" {
		if as, err := strconv.ParseInt(auditStatus, 10, 32); err == nil {
			status = audit.AuditStatus(as)
		}
	}

	req := &audit.FetchManualTasksReq{
		AuditorId: "",
		BizTypes:  []audit.BizType{}, // 空表示所有类型
		Status:    status,
		Pagination: &audit.Pagination{
			PageNum:  pageNum,
			PageSize: pageSize,
		},
	}

	// 调用 RPC 服务
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := auditClient.FetchManualTasks(ctx, req)
	if err != nil {
		log.Printf("❌ RPC call failed: %v", err)
		c.JSON(http.StatusInternalServerError, HTTPResponse{
			Code:    500,
			Message: "Failed to query audit list: " + err.Error(),
		})
		return
	}

	if resp.BaseResp.Code != 0 {
		c.JSON(http.StatusOK, HTTPResponse{
			Code:    int(resp.BaseResp.Code),
			Message: resp.BaseResp.Msg,
		})
		return
	}

	// 转换响应数据
	list := make([]map[string]interface{}, 0)
	for _, task := range resp.Tasks {
		// 解析content JSON
		var contentData map[string]interface{}
		json.Unmarshal([]byte(task.Content), &contentData)

		item := map[string]interface{}{
			"audit_id":          task.AuditId,
			"business_type":     int32(task.BizType),
			"business_id":       task.BizId,
			"audit_status":      int32(task.Status),
			"audit_status_text": getAuditStatusText(task.Status),
			"submit_time":       task.ApplyTime,
			"submitter_id":      task.SubmitterId,
			"priority":          task.Priority,
		}

		// 添加业务详情
		if contentData != nil {
			if applyReason, ok := contentData["apply_reason"].(string); ok {
				item["apply_reason"] = applyReason
			}
			if flightOrder, ok := contentData["flight_order"].(map[string]interface{}); ok {
				item["flight_order"] = flightOrder
			}
			if scenicOrder, ok := contentData["scenic_order"].(map[string]interface{}); ok {
				item["scenic_order"] = scenicOrder
			}
		}

		list = append(list, item)
	}

	c.JSON(http.StatusOK, HTTPResponse{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"total": resp.Total,
			"list":  list,
		},
	})
}

// handleProcessAudit 处理审核操作请求
func handleProcessAudit(c *gin.Context) {
	var httpReq struct {
		AuditID     int64  `json:"audit_id"`
		Action      string `json:"action"`
		AuditRemark string `json:"audit_remark"`
	}

	if err := c.ShouldBindJSON(&httpReq); err != nil {
		c.JSON(http.StatusBadRequest, HTTPResponse{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// 构建 RPC 请求
	isPassed := false
	if httpReq.Action == "approve" {
		isPassed = true
	} else if httpReq.Action != "reject" {
		c.JSON(http.StatusBadRequest, HTTPResponse{
			Code:    400,
			Message: "Invalid action, must be 'approve' or 'reject'",
		})
		return
	}

	rpcReq := &audit.ProcessManualAuditReq{
		AuditId:   httpReq.AuditID,
		AuditorId: "auditor_1", // TODO: 从会话中获取
		IsPassed:  isPassed,
		Remark:    httpReq.AuditRemark,
		Opinion:   httpReq.AuditRemark,
	}

	// 调用 RPC 服务
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := auditClient.ProcessManualAudit(ctx, rpcReq)
	if err != nil {
		log.Printf("❌ RPC call failed: %v", err)
		c.JSON(http.StatusInternalServerError, HTTPResponse{
			Code:    500,
			Message: "Failed to process audit: " + err.Error(),
		})
		return
	}

	if resp.BaseResp.Code != 0 {
		c.JSON(http.StatusOK, HTTPResponse{
			Code:    int(resp.BaseResp.Code),
			Message: resp.BaseResp.Msg,
		})
		return
	}

	c.JSON(http.StatusOK, HTTPResponse{
		Code:    0,
		Message: "success",
	})
}

// getAuditStatusText 获取审核状态文本
func getAuditStatusText(status audit.AuditStatus) string {
	switch status {
	case audit.AuditStatus_PENDING:
		return "待审核"
	case audit.AuditStatus_PROCESSING:
		return "审核中"
	case audit.AuditStatus_PASSED:
		return "通过"
	case audit.AuditStatus_REJECTED:
		return "驳回"
	case audit.AuditStatus_CANCELLED:
		return "撤销"
	default:
		return "未知"
	}
}

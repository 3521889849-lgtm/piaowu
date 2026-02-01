package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"piaowu/kitex_gen/audit"
	"piaowu/kitex_gen/audit/auditservice"
	"strconv"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var auditClient auditservice.Client

func init() {
	var err error
	// 连接 Kitex RPC 服务 (注意端口 8889)
	auditClient, err = auditservice.NewClient("audit_service", client.WithHostPorts("127.0.0.1:8889"))
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	r := gin.Default()

	// 配置 CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许前端访问
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api/audit")
	{
		api.POST("/submit", applyAudit)
		api.POST("/process", processManualAudit)
		api.GET("/record", getAuditRecord)
		api.GET("/list", getAuditList)
		api.GET("/metrics", proxyMetrics)
	}

	fmt.Println("HTTP API Gateway listening on :8080")
	r.Run(":8080")
}

// 1. 提交审核
func applyAudit(c *gin.Context) {
	var req audit.ApplyAuditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := auditClient.ApplyAudit(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// 2. 获取人工审核任务
func fetchManualTasks(c *gin.Context) {
	var req audit.FetchManualTasksReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := auditClient.FetchManualTasks(context.Background(), &req)
	if err != nil {
		fmt.Printf("RPC Error: %v\n", err) // 添加日志
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// 3. 处理人工审核
func processManualAudit(c *gin.Context) {
	var req audit.ProcessManualAuditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := auditClient.ProcessManualAudit(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// 4. 获取审核记录
func getAuditRecord(c *gin.Context) {
	auditIDStr := c.Query("audit_id")
	bizID := c.Query("biz_id")
	bizTypeStr := c.Query("biz_type")

	req := &audit.GetAuditRecordReq{
		BizId: bizID,
	}

	if auditIDStr != "" {
		if id, err := strconv.ParseInt(auditIDStr, 10, 64); err == nil {
			req.AuditId = id
		}
	}
	if bizTypeStr != "" {
		if bt, err := strconv.ParseInt(bizTypeStr, 10, 64); err == nil {
			req.BizType = audit.BizType(bt)
		}
	}

	resp, err := auditClient.GetAuditRecord(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// 4.5 获取审核列表 (对接前端 /list)
func getAuditList(c *gin.Context) {
	var req audit.FetchManualTasksReq

	// 解析分页
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	pageNum, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	req.Pagination = &audit.Pagination{
		PageNum:  int32(pageNum),
		PageSize: int32(pageSize),
	}

	// 解析业务类型
	if bt := c.Query("business_type"); bt != "" {
		if val, err := strconv.Atoi(bt); err == nil {
			req.BizTypes = []audit.BizType{audit.BizType(val)}
		}
	}

	// 解析状态
	if st := c.Query("audit_status"); st != "" {
		if val, err := strconv.Atoi(st); err == nil {
			req.Status = audit.AuditStatus(val)
		}
	}

	resp, err := auditClient.FetchManualTasks(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 适配前端 AuditListResponse 结构
	// 前端期望: { code: 0, data: { total: number, list: AuditDetail[] } }
	type AuditDetail struct {
		AuditId         int64  `json:"audit_id"`
		BusinessType    int32  `json:"business_type"`
		BusinessId      int64  `json:"business_id"`
		AuditStatus     int32  `json:"audit_status"`
		AuditStatusText string `json:"audit_status_text"`
		SubmitUserId    int64  `json:"submit_user_id"`
		SubmitUserName  string `json:"submit_user_name"`
		SubmitTime      string `json:"submit_time"`
		AuditRemark     string `json:"audit_remark"`
	}

	list := make([]AuditDetail, 0)
	statusMap := map[int32]string{
		1: "待审核",
		2: "审核中",
		3: "通过",
		4: "驳回",
		5: "已撤销",
	}

	for _, t := range resp.Tasks {
		bizID, _ := strconv.ParseInt(t.BizId, 10, 64)
		subID, _ := strconv.ParseInt(t.SubmitterId, 10, 64)
		list = append(list, AuditDetail{
			AuditId:         t.AuditId,
			BusinessType:    int32(t.BizType),
			BusinessId:      bizID,
			AuditStatus:     int32(t.Status),
			AuditStatusText: statusMap[int32(t.Status)],
			SubmitUserId:    subID,
			SubmitUserName:  "User_" + t.SubmitterId,
			SubmitTime:      t.ApplyTime,
			AuditRemark:     t.Content,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"total": resp.Total,
			"list":  list,
		},
	})
}

// 5. 转发监控数据
func proxyMetrics(c *gin.Context) {
	resp, err := http.Get("http://127.0.0.1:8890/api/audit/metrics")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Metrics service unavailable"})
		return
	}
	defer resp.Body.Close()
	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

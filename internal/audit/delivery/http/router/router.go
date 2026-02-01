package router

import (
	"piaowu/internal/audit/delivery/http/handler"
	"piaowu/internal/audit/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter 设置路由
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// 配置CORS中间  ?
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 创建用例
	auditUsecase := usecase.NewAuditUsecase(db)

	// 创建处理  ?
	auditHandler := handler.NewAuditHandler(auditUsecase)

	// API v1 路由  ?
	v1 := r.Group("/api/v1")
	{
		// 审核相关路由
		audit := v1.Group("/audit")
		{
			// 提交审核（外部系统调用）
			audit.POST("/submit", auditHandler.SubmitAudit)

			// 查询审核详情
			audit.GET("/:audit_id", auditHandler.GetAuditDetail)

			// 查询审核列表
			audit.GET("/list", auditHandler.QueryAuditList)

			// 处理审核（审核员操作  ?
			audit.POST("/process", auditHandler.ProcessAudit)
		}
	}

	// 健康检  ?
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return r
}

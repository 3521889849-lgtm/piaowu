package main

import (
	"piaowu/common/config"
	"piaowu/common/db"
	"piaowu/internal/audit/delivery/http/router"
	"log"
)

func main() {
	// 加载配置
	if err := config.ViperInit(); err != nil {
		log.Fatalf("配置初始化失  ? %v", err)
	}

	// 初始化数据库
	if err := db.MysqlInit(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 设置路由
	r := router.SetupRouter(db.DB)

	// 启动服务器（默认8080端口  ?
	port := ":8080"
	log.Printf("审核服务启动成功，监听端  ? 8080")
	log.Printf("提交审核接口: http://localhost:8080/api/v1/audit/submit")
	log.Printf("查询审核列表: http://localhost:8080/api/v1/audit/list")
	log.Printf("健康检  ? http://localhost:8080/health")
	
	if err := r.Run(port); err != nil {
		log.Fatalf("服务器启动失  ? %v", err)
	}
}

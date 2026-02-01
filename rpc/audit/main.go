package main

import (
	"encoding/json"
	"piaowu/common/db"
	_ "piaowu/common/init" // 加载配置
	"piaowu/kitex_gen/audit/auditservice"
	"piaowu/rpc/audit/component/metrics"
	"log"
	"net"
	"net/http"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

func main() {
	// 1. 初始化数据库和Redis（复用common/db）
	// common/init包的init()方法已经完成了配置加载、MySQL和Redis的初始化
	// 这里显式检查连接状态（可选，增加健壮性）
	if db.DB == nil {
		log.Fatal("MySQL连接未初始化")
	}
	if db.Rdb == nil {
		log.Println("Redis连接未初始化，已跳过")
	}

	// 启动监控 API（预留接口）
	go startMetricsServer()

	// 2. 绑定服务地址
	addr, err := net.ResolveTCPAddr("tcp", ":8889")
	if err != nil {
		log.Fatal(err)
	}

	// 3. 初始化 Server
	svr := auditservice.NewServer(
		new(AuditServiceImpl),
		server.WithServiceAddr(addr),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "audit_service",
		}),
	)

	// 4. 启动服务
	log.Println("✅ Audit Service 启动成功，监听端口 :8889")
	if err := svr.Run(); err != nil {
		log.Println("启动失败:", err)
	}
}

// startMetricsServer 启动监控 API
func startMetricsServer() {
	http.HandleFunc("/api/audit/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		snapshot := metrics.DefaultCollector.Snapshot()
		b, _ := json.Marshal(snapshot)
		_, _ = w.Write(b)
	})
	log.Println("✅ Audit Metrics API 启动成功，监听端口 :8890")
	if err := http.ListenAndServe(":8890", nil); err != nil {
		log.Println("Metrics API 启动失败:", err)
	}
}

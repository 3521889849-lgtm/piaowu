package main

import (
	// 触发项目统一初始化：配置加载、MySQL/Redis 连接等
	_ "example_shop/common/init"
	// 业务 handler 放在 rpc/coupon 包中，main 只负责启动并注入实现
	handler "example_shop/rpc/coupon"
	// Kitex 生成的服务端注册代码
	"example_shop/kitex_gen/coupon/couponservice"

	"log"

	// Kitex server 启动相关配置
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

func main() {
	// 创建并启动 Kitex Server
	svr := couponservice.NewServer(
		// 注入业务实现：已在 handler.go 中实现 Test/SpotList/SpotDetail 等方法
		handler.NewCouponServiceImpl(),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			// 对外暴露的服务名（可在服务发现/治理中使用）
			ServiceName: "coupon_service",
		}),
	)

	// 启动服务并监听端口
	log.Println("✅ 极简空服务启动成功！")
	if err := svr.Run(); err != nil {
		// 启动失败时打印错误信息
		log.Println("启动失败：", err)
	}
}

/*
 * Gateway服务主入口
 *
 * 功能说明：
 * - 作为API网关，统一接收客户端HTTP请求
 * - 负责路由转发和请求分发到后端微服务
 * - 使用Hertz作为HTTP框架，Kitex作为RPC客户端
 *
 * 架构位置：
 * 客户端 -> Gateway(本服务) -> User Service(通过RPC调用)
 *
 * 技术栈：
 * - Hertz: CloudWeGo团队开源的高性能HTTP框架
 * - Kitex: CloudWeGo团队开源的高性能RPC框架
 */
package main

import (
	"example_shop/common/config"
	initpkg "example_shop/common/init"
	"example_shop/internal/gateway/http/handler"
	"example_shop/internal/gateway/http/router"
	"example_shop/internal/gateway/rpc"
	// 标准库日志模块：用于输出程序运行日志（错误/信息）
	"fmt"
	"log"

	// Hertz核心包：创建HTTP服务器实例
	"github.com/cloudwego/hertz/pkg/app/server"
)

// main函数：程序入口，负责初始化网关服务的所有依赖并启动服务
//
// 执行流程：
// 1. 加载配置文件（config.yaml）
// 2. 初始化RPC客户端（连接User Service）
// 3. 创建HTTP服务器
// 4. 注册路由和处理器
// 5. 启动服务监听请求
func main() {
	// 初始化配置、MySQL、Redis（票务查询接口会直接访问缓存与数据库）
	shutdown := initpkg.Init("api_gateway")
	defer shutdown()

	// 2. 从配置文件读取服务地址（避免硬编码，提高灵活性）
	// Gateway服务监听地址：接收客户端HTTP请求
	gatewayCfg := config.Cfg.Server.Gateway
	// User Service地址：用于RPC调用后端用户服务
	userServiceCfg := config.Cfg.Server.UserService
	ticketServiceCfg := config.Cfg.Server.TicketService
	orderServiceCfg := config.Cfg.Server.OrderService
	// 格式化监听地址：0.0.0.0:5200（0.0.0.0表示监听所有网卡）
	listenAddr := fmt.Sprintf("%s:%d", gatewayCfg.Host, gatewayCfg.Port)
	// 格式化RPC服务地址：127.0.0.1:8888
	userServiceAddr := fmt.Sprintf("%s:%d", userServiceCfg.Host, userServiceCfg.Port)
	ticketServiceAddr := fmt.Sprintf("%s:%d", ticketServiceCfg.Host, ticketServiceCfg.Port)
	orderServiceAddr := fmt.Sprintf("%s:%d", orderServiceCfg.Host, orderServiceCfg.Port)

	// 3. 创建Kitex RPC客户端（用于调用user_service服务）
	// RPC客户端采用直连模式，后续可扩展为服务发现（Consul/Nacos等）
	userClient, err := rpc.NewUserClient(userServiceAddr)
	if err != nil {
		// RPC客户端创建失败则退出程序
		log.Fatalf("ticket_service client 初始化失败: %v", err)
	}
	ticketClient, err := rpc.NewTicketClient(ticketServiceAddr)
	if err != nil {
		log.Fatalf("ticket_service client 初始化失败: %v", err)
	}
	orderClient, err := rpc.NewOrderClient(orderServiceAddr)
	if err != nil {
		log.Fatalf("order_service client 初始化失败: %v", err)
	}

	// 4. 创建Hertz HTTP服务器实例
	// server.Default：创建默认配置的Hertz服务器，指定监听地址
	h := server.Default(server.WithHostPorts(listenAddr))

	// 5. 初始化网关业务实例，注入RPC客户端依赖
	// 依赖注入模式：将RPC客户端注入到Handler中，便于测试和解耦
	// App结构体包含所有需要调用的RPC客户端
	app := handler.NewApp(userClient, ticketClient, orderClient)

	// 6. 注册网关所有路由
	// 路由注册：将HTTP请求路径与处理函数绑定
	// 例如：POST /api/v1/user/register -> app.Register
	router.RegisterRoutes(h, app)

	// 7. 打印启动日志，提示网关服务已启动
	log.Printf("api_gateway started at %s", listenAddr)

	// 8. 启动Hertz服务器（阻塞运行，直到程序终止）
	// Spin()：启动HTTP服务，监听端口并处理请求，不会立即返回
	h.Spin()
}

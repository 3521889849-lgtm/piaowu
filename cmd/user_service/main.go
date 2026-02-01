/*
 * User Service服务主入口
 *
 * 功能说明：
 * - 提供用户相关的RPC服务（注册、登录、实名认证等）
 * - 使用Kitex框架提供RPC服务能力
 * - 处理用户相关的业务逻辑和数据库操作
 *
 * 架构位置：
 * Gateway -> User Service(本服务) -> MySQL/Redis
 *
 * 技术栈：
 * - Kitex: RPC服务框架
 * - GORM: ORM框架操作MySQL
 * - Redis: 缓存和会话存储
 */
package main

import (
	"example_shop/common/config"
	initpkg "example_shop/common/init"
	"example_shop/internal/ticket_service/handler/userapi"
	"example_shop/kitex_gen/userapi/userservice"
	"fmt"
	"log"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/utils"
	"github.com/cloudwego/kitex/server"
	ktracer "github.com/kitex-contrib/tracer-opentracing"
)

// main函数：用户服务的入口函数
//
// 执行流程：
// 1. 初始化配置、MySQL、Redis连接
// 2. 创建Kitex RPC服务器
// 3. 注册服务处理器（UserServiceImpl）
// 4. 启动RPC服务监听请求
func main() {
	// 初始化配置和数据库
	// 包括：配置加载、MySQL连接、Redis连接、数据库表迁移
	shutdown := initpkg.Init("user_service")
	defer shutdown()

	listenAddr := fmt.Sprintf("%s:%d", config.Cfg.Server.UserService.Host, config.Cfg.Server.UserService.Port)

	// 创建Kitex RPC服务器实例
	// NewServer：创建新的RPC服务器
	// UserServiceImpl：实现用户服务接口的具体业务逻辑处理器
	// WithServerBasicInfo：设置服务基本信息（服务名用于服务发现和监控）
	svr := userservice.NewServer(
		userapi.NewUserServiceImpl(),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "user_service",
		}),
		server.WithServiceAddr(utils.NewNetAddr("tcp", listenAddr)),
		server.WithSuite(ktracer.NewDefaultServerSuite()),
	)

	// 启动RPC服务
	// Run()：阻塞运行，监听RPC请求直到程序终止
	log.Println("user_service started")
	if err := svr.Run(); err != nil {
		log.Println("user_service stopped:", err)
	}
}

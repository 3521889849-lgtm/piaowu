/*
 * 公共初始化模块
 * 
 * 功能说明：
 * - 统一管理所有服务的初始化逻辑
 * - 按顺序初始化配置、数据库、缓存等基础设施
 * 
 * 使用场景：
 * - 各微服务启动时调用，确保依赖资源已就绪
 * - 避免在各个服务中重复初始化代码
 */
package init

import (
	"example_shop/common/config"
	"example_shop/common/db"
	"example_shop/common/trace"
	"log"
)

// Init 初始化配置和数据库连接
// 
// 初始化顺序（重要！）：
// 1. 配置初始化：必须先加载配置，后续初始化依赖配置
// 2. MySQL初始化：数据库连接和表迁移
// 3. Redis初始化：缓存连接
// 
// 注意：如果任何一个步骤失败，程序会退出（log.Fatalf）
// 这样可以在启动时就发现问题，而不是运行时才暴露
func Init(serviceName string) (shutdown func()) {
	// 1. 初始化配置（加载config.yaml文件）
	if err := config.ViperInit(); err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	}

	// 2. 初始化MySQL连接
	// 包括：建立连接、测试连接、数据库表迁移、连接池配置
	if err := db.MysqlInit(); err != nil {
		log.Fatalf("MySQL连接初始化失败: %v", err)
	}

	// 3. 初始化Redis连接
	// 用于缓存、分布式锁、会话存储等
	if err := db.RedisInit(); err != nil {
		log.Printf("Redis连接初始化失败: %v", err)
	}

	closer, err := trace.InitJaeger(serviceName)
	if err != nil {
		log.Fatalf("Tracing初始化失败: %v", err)
	}
	shutdown = func() {
		if closer != nil {
			_ = closer.Close()
		}
	}
	
	// 所有初始化完成
	log.Println("所有初始化完成，服务就绪")
	return shutdown
}

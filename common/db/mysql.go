/*
 * MySQL数据库连接和初始化模块
 *
 * 功能说明：
 * - 建立MySQL数据库连接
 * - 执行数据库表自动迁移（AutoMigrate）
 * - 配置连接池参数
 *
 * 使用的ORM框架：GORM
 * - 支持自动迁移（根据Model自动创建/更新表结构）
 * - 支持软删除、时间戳自动管理等功能
 */
package db

import (
	"example_shop/common/config"
	"example_shop/internal/model"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger" // 引入日志包，方便调试
)

// MysqlDB 全局MySQL数据库连接对象
// 初始化后供所有模块使用
var MysqlDB *gorm.DB

// MysqlReadDB 只读MySQL副本连接对象（读写分离）
// - 用于高频查询，降低主库压力
// - 未配置只读库时，会退化为使用MysqlDB
var MysqlReadDB *gorm.DB

func ReadDB() *gorm.DB {
	if MysqlReadDB != nil {
		return MysqlReadDB
	}
	return MysqlDB
}

// MysqlInit 初始化MySQL连接并完成表迁移
//
// 执行流程：
// 1. 读取MySQL配置
// 2. 构建DSN连接字符串
// 3. 建立数据库连接
// 4. 测试连接有效性
// 5. 执行数据库表迁移（AutoMigrate）
// 6. 配置连接池参数
//
// 返回值：
// - error: 如果初始化失败，返回错误信息
func MysqlInit() error {
	// 修正：读取正确的配置节点（你的配置是 Mysql，不是 MysqlInit）
	MysqlS := config.Cfg.Mysql

	// 配置GORM日志级别
	logLevel := strings.ToLower(strings.TrimSpace(MysqlS.LogLevel))
	gormLogMode := logger.Warn
	switch logLevel {
	case "silent":
		gormLogMode = logger.Silent
	case "error":
		gormLogMode = logger.Error
	case "warn", "":
		gormLogMode = logger.Warn
	case "info":
		gormLogMode = logger.Info
	default:
		gormLogMode = logger.Warn
	}
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(gormLogMode),
	}

	// 校验配置是否为空（关键：避免读取到空配置导致地址无效）
	if MysqlS.Host == "" || MysqlS.Port == 0 || MysqlS.User == "" || MysqlS.Database == "" {
		return fmt.Errorf("mysql配置不完整，必填项：Host/Port/User/Database")
	}

	// 构建DSN（Data Source Name）连接字符串
	// 时区URL编码（兼容Windows/Linux），使用上海时区
	loc := url.QueryEscape("Asia/Shanghai")

	// DSN格式：username:password@tcp(host:port)/database?params
	// 参数说明：
	// - charset=utf8mb4: 支持完整的UTF-8字符集（包括emoji）
	// - parseTime=True: 自动解析时间类型
	// - loc: 时区设置
	// - timeout: 连接超时时间
	// - readTimeout/writeTimeout: 读写超时时间
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=%s&timeout=10s&readTimeout=30s&writeTimeout=30s",
		MysqlS.User,
		MysqlS.Password,
		MysqlS.Host,
		MysqlS.Port,
		MysqlS.Database,
		loc,
	)

	// 建立连接
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("mysql连接失败：%w", err) // 包装错误，保留原始信息
	}
	fmt.Printf("数据库连接成功：%s:%d/%s（AutoMigrate=%t）\n", MysqlS.Host, MysqlS.Port, MysqlS.Database, MysqlS.AutoMigrate)

	// 获取底层sql.DB对象，验证连接有效性（关键：主动Ping）
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取sql.DB对象失败：%w", err)
	}
	// 主动Ping测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("验证mysql连接失败：%w", err)
	}

	// 临时禁用外键检查，避免迁移时的外键约束问题
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	defer func() {
		// 延迟恢复外键检查，确保无论迁移成功/失败都会执行
		db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}()

	if MysqlS.AutoMigrate {
		// 执行数据库表自动迁移（AutoMigrate）
		// 功能：根据Model结构体自动创建表或更新表结构
		// 注意：只会添加缺失的列和索引，不会删除/修改已存在的列
		//
		// 按依赖关系迁移表（先迁移基础表，再迁移依赖表）
		err = db.AutoMigrate(
			// ========== 基础表（无外键依赖）==========
			&model.UserInfo{},         // 用户表：存储用户基本信息
			&model.PassengerInfo{},    // 乘客表：存储乘客信息（关联订单）
			&model.TrainInfo{},        // 车次表：存储车次基本信息
			&model.SeatInfo{},         // 座位表：存储座位信息和状态
			&model.TicketRuleConfig{}, // 购票规则配置表：存储购票规则和限购策略
			&model.KnowledgeDoc{},

			// ========== 依赖基础表的表 ==========
			&model.TrainStationPass{},     // 车次途径站点表（依赖TrainInfo）
			&model.SeatSegmentOccupancy{}, // 座位区间占用表（依赖TrainInfo/SeatInfo）
			&model.OrderInfo{},            // 车票订单表（依赖UserInfo/PassengerInfo/TrainInfo）
			&model.OrderSeatRelation{},    // 订单座位关联表（依赖OrderInfo/SeatInfo）
			&model.OrderAuditLog{},        // 订单操作审计表（依赖OrderInfo）：记录订单操作日志
			&model.TicketInventoryLog{},   // 余票变更日志表（依赖TrainInfo/SeatInfo）：记录库存变化
			&model.KnowledgeDoc{},         //智能助手知识库文档表（用于RAG召回）
		)
		if err != nil {
			return fmt.Errorf("数据库表迁移失败：%w", err)
		}
		if !db.Migrator().HasTable(&model.UserInfo{}) {
			return fmt.Errorf("数据库表迁移后仍未发现用户表(user_infos)，请检查连接的Database与权限")
		}

		if MysqlS.StartupFixes {
			if err := db.Exec("ALTER TABLE `user_infos` MODIFY COLUMN `id_card` VARCHAR(32) NULL DEFAULT NULL").Error; err != nil {
				return fmt.Errorf("修复user_infos.id_card列为NULL失败：%w", err)
			}
			if err := db.Exec("UPDATE `user_infos` SET `id_card` = NULL WHERE `id_card` = ''").Error; err != nil {
				return fmt.Errorf("修复user_infos.id_card空字符串数据失败：%w", err)
			}
		}

		fmt.Println("数据库迁移成功")
	}

	// 配置数据库连接池参数（优化数据库性能）
	// 连接池：复用数据库连接，避免频繁创建/销毁连接的开销
	sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数：保持10个连接处于空闲状态
	sqlDB.SetMaxOpenConns(100)                 // 最大打开连接数：最多同时打开100个连接
	sqlDB.SetConnMaxLifetime(30 * time.Hour)   // 连接最大存活时间：连接使用30小时后强制关闭重建（避免MySQL超时）
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 连接最大空闲时间：空闲10分钟后关闭（优化资源释放）

	MysqlDB = db

	if MysqlS.ReadReplica.Enabled {
		readCfg := MysqlS.ReadReplica
		if readCfg.Database == "" {
			readCfg.Database = MysqlS.Database
		}
		if readCfg.Host == "" || readCfg.Port == 0 || readCfg.User == "" || readCfg.Database == "" {
			return fmt.Errorf("mysql只读库配置不完整，必填项：Host/Port/User/Database")
		}

		readDSN := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=%s&timeout=10s&readTimeout=30s&writeTimeout=30s",
			readCfg.User,
			readCfg.Password,
			readCfg.Host,
			readCfg.Port,
			readCfg.Database,
			loc,
		)

		readDB, err := gorm.Open(mysql.Open(readDSN), gormConfig)
		if err != nil {
			return fmt.Errorf("mysql只读库连接失败：%w", err)
		}
		MysqlReadDB = readDB
		fmt.Println("MySQL只读库连接成功")
	}

	return nil
}

package db

import (
	"database/sql"
	"fmt"
	"log"
	"piaowu/common/config"
	"piaowu/common/model/audit"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 全局数据库对象（GORM的DB）
var DB *gorm.DB

// 全局数据库连接池对象（原生sql.DB）
var SqlDB *sql.DB

func MysqlInit() error {
	// 从配置中获取数据库信息
	mysqlConf := config.Cfg.Mysql
	// 修复format格式串（正确的DSN格式）
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FShanghai",
		mysqlConf.User,
		mysqlConf.Password,
		mysqlConf.Host,
		mysqlConf.Port,
		mysqlConf.Database,
	)

	// 修复gorm.Open的拼写，添加日志配置（便于调试）
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 打印SQL日志
	})
	if err != nil {
		// 连接失败时返回具体错误
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 赋值全局GORM DB对象
	DB = db

	// 获取原生sql.DB连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取连接池失败: %w", err)
	}
	// 赋值全局连接池对象
	SqlDB = sqlDB

	// 配置连接池（修复变量作用域）
	SqlDB.SetMaxIdleConns(10)                // 最大空闲连接数
	SqlDB.SetMaxOpenConns(100)               // 最大打开连接数
	SqlDB.SetConnMaxLifetime(time.Hour * 30) // 连接最大存活时间
	// 执行自动迁移（根据模型创建/更新表）
	// 注：跳过历史业务表的自动迁移（存在历史数据兼容问题，需手动处理）
	if err := DB.AutoMigrate(
		&audit.AuditMain{},
		// &audit.AuditTicketOrder{}, // 暂时跳过（历史数据）
		// &audit.AuditHotelOrder{},  // 暂时跳过（历史数据）
		&audit.AuditFlightOrder{}, // 新增：机票审核
		&audit.AuditScenicOrder{}, // 新增：旅游门票审核
		&audit.AuditOperationLog{},
		&audit.AuditFlowConfig{},
		&audit.AuditRule{},
		&audit.RuleConfig{},
	); err != nil {
		return fmt.Errorf("自动迁移失败: %w", err)
	}

	log.Println("数据库初始化成功")
	return nil
}

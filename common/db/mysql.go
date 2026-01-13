package db

import (
	"example_shop/common/config"
	"example_shop/common/model"
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var MysqlDB *gorm.DB

func MysqlInit() error {
	MysqlS := config.Cfg.MysqlInit
	// 在 Windows 上使用 Asia/Shanghai 时区，URL编码后使用
	loc := url.QueryEscape("Asia/Shanghai")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=%s",
		MysqlS.User,
		MysqlS.Password,
		MysqlS.Host,
		MysqlS.Port,
		MysqlS.Database,
		loc,
	)
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		return err
	}
	fmt.Println("数据库连接成功")
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// 临时禁用外键检查，避免迁移时的外键约束问题
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")

	// 按照依赖关系顺序迁移：先创建父表，再创建子表
	// 第一层：基础表（无外键依赖）
	// 第二层：依赖第一层的表
	// 第三层：依赖第二层的表
	err = db.AutoMigrate(
		// 第一层：基础表（无外键依赖）
		&model.SysAdmin{},    // 管理员表
		&model.SysUser{},     // 用户表
		&model.SysMerchant{}, // 商家表（依赖 SysAdmin，但 AdminID 可为空，所以可以先创建）
		&model.Coupon{},      // 优惠券表
		// 第二层：依赖第一层的表
		&model.SpotInfo{},    // 景点表（依赖 SysMerchant）
		&model.Traveler{},    // 出行人表（依赖 SysUser）
		&model.UserCoupon{}, // 用户优惠券表（依赖 SysUser, Coupon）
		&model.TicketType{},  // 门票类型表（依赖 SpotInfo）
		// 第三层：依赖第二层的表
		&model.OrderMain{},   // 主订单表（依赖 SysUser, SysMerchant, SpotInfo）
		&model.OrderItem{},   // 订单详情表（依赖 OrderMain, TicketType, Traveler）
		&model.PayRecord{},   // 支付记录表（依赖 OrderMain）
		&model.SysOperLog{},  // 操作日志表（依赖 SysAdmin）
	)
	if err != nil {
		// 恢复外键检查
		db.Exec("SET FOREIGN_KEY_CHECKS = 1")
		return err
	}

	// 恢复外键检查
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	fmt.Println("数据库迁移成功")

	// SetMaxIdleConns 设置空闲连接池中连接的最大数量。
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime 设置了可以重新使用连接的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour * 30)
	MysqlDB = db
	return nil
}

package db

import (
	"example_shop/common/config"
	model "example_shop/common/model/attraction_ticket"
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
	err = db.AutoMigrate(&model.BaseModel{}, &model.Coupon{}, &model.OrderItem{}, &model.OrderMain{}, &model.PayRecord{},
		&model.SpotInfo{}, &model.SysMerchant{}, &model.SysAdmin{}, &model.SysUser{}, &model.SysOperLog{}, &model.TicketType{},
		&model.Traveler{}, &model.UserCoupon{})
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

package init

import (
	"piaowu/common/config"
	"piaowu/common/db"
	"log"
)

func init() {

	if err := config.ViperInit(); err != nil {
		log.Fatalf("配置初始化失  ? %v", err)
	}
	if err := db.MysqlInit(); err != nil {
		log.Fatalf("MySQL连接初始化失  ? %v", err)
	}
	if err := db.RedisInit(); err != nil {
		log.Printf("Redis连接初始化失  ? %v，已跳过", err)
	}
}


package db

import (
	"context"
	"fmt"
	"piaowu/common/config"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var Ctx = context.Background()

func RedisInit() error {
	redisCfg := config.Cfg.Redis

	Rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password, // no password set
		DB:       redisCfg.Database, // use default DB
	})

	//测试连接
	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		return err
	}
	fmt.Println("Redis连接成功")
	return nil
}

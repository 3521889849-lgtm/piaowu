package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func ViperInit() error {
	var err error
	// 添加多个配置路径，支持不同运行目录
	viper.AddConfigPath("conf")
	viper.AddConfigPath("../conf")
	viper.AddConfigPath("../../conf")
	viper.AddConfigPath("../../../conf") // 新增这一行，对应根目录的conf
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("配置文件读取失败: %w", err)
	}
	err = viper.Unmarshal(&Cfg)
	if err != nil {
		return fmt.Errorf("配置文件解析失败: %w", err)
	}
	fmt.Println("配置动态加载成功", Cfg)
	return nil
}

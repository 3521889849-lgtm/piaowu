package config

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateAndNormalize() error {
	if Cfg == nil {
		return errors.New("Cfg未初始化")
	}

	if Cfg.Mysql.Port == 0 {
		Cfg.Mysql.Port = 3306
	}
	if Cfg.Redis.Port == 0 {
		Cfg.Redis.Port = 6379
	}
	if Cfg.JWT.Expire == 0 {
		Cfg.JWT.Expire = 7200
	}
	if Cfg.Tracing.Jaeger.AgentPort == 0 {
		Cfg.Tracing.Jaeger.AgentPort = 6831
	}
	if strings.TrimSpace(Cfg.Tracing.Jaeger.SamplerType) == "" {
		Cfg.Tracing.Jaeger.SamplerType = "const"
	}
	if Cfg.Tracing.Jaeger.SamplerParam == 0 {
		Cfg.Tracing.Jaeger.SamplerParam = 1
	}

	if strings.TrimSpace(Cfg.Mysql.Host) == "" || strings.TrimSpace(Cfg.Mysql.User) == "" || strings.TrimSpace(Cfg.Mysql.Database) == "" {
		return fmt.Errorf("Mysql配置缺失(Host/User/Database)")
	}
	if strings.TrimSpace(Cfg.Redis.Host) == "" {
		return fmt.Errorf("Redis配置缺失(Host)")
	}
	if strings.TrimSpace(Cfg.Server.Gateway.Host) == "" || Cfg.Server.Gateway.Port == 0 {
		return fmt.Errorf("Server.Gateway配置缺失(Host/Port)")
	}
	if strings.TrimSpace(Cfg.JWT.Secret) == "" {
		return fmt.Errorf("JWT.Secret缺失")
	}
	return nil
}

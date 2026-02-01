package decision

import (
	"context"
	"piaowu/rpc/audit/component/rule_engine"
	"sync"
)

// Plugin 审核插件接口
// 允许外部扩展：特征补充、黑名单校验、风控标签、外部服务调用等
// 通过注册插件实现“插件式”功能扩  ?

type Plugin interface {
	Name() string
	BeforeDecision(ctx context.Context, fact rule_engine.Fact) error
	AfterDecision(ctx context.Context, result *Result) error
}

// PluginManager 插件管理  ?
// 支持并发安全注册，满足运行期动态扩  ?

type PluginManager struct {
	mu       sync.RWMutex
	plugins  []Plugin
}

func NewPluginManager() *PluginManager {
	return &PluginManager{plugins: make([]Plugin, 0)}
}

func (pm *PluginManager) Register(p Plugin) {
	if p == nil {
		return
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.plugins = append(pm.plugins, p)
}

func (pm *PluginManager) BeforeDecision(ctx context.Context, fact rule_engine.Fact) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, p := range pm.plugins {
		if err := p.BeforeDecision(ctx, fact); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PluginManager) AfterDecision(ctx context.Context, result *Result) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, p := range pm.plugins {
		if err := p.AfterDecision(ctx, result); err != nil {
			return err
		}
	}
	return nil
}


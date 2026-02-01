package decision

import (
	"context"
	"piaowu/rpc/audit/component/rule_engine"
	"strings"
)

// BlacklistPlugin 黑名单插件示  ?
type BlacklistPlugin struct {
	blacklist []string
}

func NewBlacklistPlugin(list []string) *BlacklistPlugin {
	return &BlacklistPlugin{blacklist: list}
}

func (p *BlacklistPlugin) Name() string {
	return "BlacklistPlugin"
}

func (p *BlacklistPlugin) BeforeDecision(ctx context.Context, fact rule_engine.Fact) error {
	content, ok := fact["content"].(string)
	if !ok {
		return nil
	}

	for _, word := range p.blacklist {
		if strings.Contains(content, word) {
			// 如果命中黑名单，可以在这里做标记或记录，
			// 虽然接口返回 error，但在决策流中我们可以定义更复杂的逻辑
			// 示例：直接在 fact 中注入黑名单标记
			fact["is_blacklisted"] = true
			fact["blacklist_word"] = word
		}
	}
	return nil
}

func (p *BlacklistPlugin) AfterDecision(ctx context.Context, result *Result) error {
	// 如果  ?BeforeDecision 中标记了命中黑名单，则在此处强制更改结果为拒  ?
	// 这样可以确保黑名单策略具有最高优先级（在决策流之后覆盖）
	if result.Extra != nil {
		if isBlacklisted, ok := result.Extra["is_blacklisted"].(bool); ok && isBlacklisted {
			result.FinalAction = rule_engine.ActionReject
			result.Status = 4 // 审核拒绝
			result.Remark = "命中系统黑名  ? " + result.Extra["blacklist_word"].(string)
			result.Extra["final_action"] = string(rule_engine.ActionReject)
		}
	}
	return nil
}



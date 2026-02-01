package rule_engine

import (
	"sort"
	"testing"
)

func TestRuleEngine_Execute(t *testing.T) {
	// 初始化引擎 (手动注入规则)
	re := &RuleEngine{
		rules: make(map[string][]*Rule),
	}

	// 构造测试规则
	// 规则1: 订单金额 > 2000 -> Reject (优先级 10)
	rule1 := &Rule{
		ID:       1,
		BizType:  "TICKET_ORDER",
		RuleName: "High Amount Limit",
		Action:   ActionReject,
		Priority: 10,
		Expression: &RuleExpression{
			Op:    ">",
			Field: "order_amount",
			Value: 2000.0,
		},
	}

	// 规则2: 身份证格式错误 -> Reject (优先级 20)
	rule2 := &Rule{
		ID:       2,
		BizType:  "TICKET_ORDER",
		RuleName: "Invalid ID Card",
		Action:   ActionReject,
		Priority: 20,
		Expression: &RuleExpression{
			Op:    "<",
			Field: "order_amount",
			Value: 0.0,
		},
	}

	// 规则3: 组合逻辑 (VIP AND Amount < 100) -> Pass (优先级 30)
	rule3 := &Rule{
		ID:       3,
		BizType:  "TICKET_ORDER",
		RuleName: "VIP Small Order",
		Action:   ActionPass,
		Priority: 30,
		Expression: &RuleExpression{
			Op: "AND",
			SubRules: []*RuleExpression{
				{Op: "==", Field: "is_vip", Value: true},
				{Op: "<", Field: "order_amount", Value: 100.0},
			},
		},
	}

	// 预处理表达式（确保 REGEX/IN 等预编译逻辑生效）
	_ = prepareExpression(rule1.Expression)
	_ = prepareExpression(rule2.Expression)
	_ = prepareExpression(rule3.Expression)

	rules := []*Rule{rule3, rule2, rule1}
	// 按优先级排序，模拟 Reload 行为
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	re.rules["TICKET_ORDER"] = rules

	tests := []struct {
		name       string
		fact       Fact
		wantMatch  bool
		wantAction RuleAction
	}{
		{
			name: "VIP Small Order (Should Match Rule 3)",
			fact: Fact{
				"is_vip":       true,
				"order_amount": 50.0,
			},
			wantMatch:  true,
			wantAction: ActionPass,
		},
		{
			name: "High Amount (Should Match Rule 1)",
			fact: Fact{
				"is_vip":       false,
				"order_amount": 2500.0,
			},
			wantMatch:  true,
			wantAction: ActionReject,
		},
		{
			name: "Normal Order (No Match)",
			fact: Fact{
				"is_vip":       false,
				"order_amount": 500.0,
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := re.Execute("TICKET_ORDER", tt.fact)
			if got.Matched != tt.wantMatch {
				t.Errorf("Execute() matched = %v, want %v", got.Matched, tt.wantMatch)
			}
			if tt.wantMatch && got.Action != tt.wantAction {
				t.Errorf("Execute() action = %v, want %v", got.Action, tt.wantAction)
			}
		})
	}
}

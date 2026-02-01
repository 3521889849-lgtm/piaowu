package rule_engine

import "regexp"

// RuleAction 规则动作
type RuleAction string

const (
	ActionPass   RuleAction = "Pass"
	ActionReject RuleAction = "Reject"
	ActionReview RuleAction = "Review"
)

// Rule 内存中的规则对象
type Rule struct {
	ID         int64
	BizType    string
	RuleName   string
	Expression *RuleExpression // 解析后的表达式
	Action     RuleAction
	Priority   int
}

// RuleExpression 规则表达式结构 (JSON)
// 支持简单结构: { "op": "AND", "rules": [ ... ] } 或 { "field": "amount", "op": ">", "value": 100 }
type RuleExpression struct {
	Op       string            `json:"op"`                  // 操作符: AND, OR, >, >=, <, <=, ==, !=, IN, REGEX
	Field    string            `json:"field,omitempty"`     // 字段名 (仅叶子节点)
	Value    interface{}       `json:"value,omitempty"`     // 比较值 (仅叶子节点)
	SubRules []*RuleExpression `json:"sub_rules,omitempty"` // 子规则 (仅组合节点)

	// 预编译字段（运行期使用，不入库）
	regex *regexp.Regexp      `json:"-"`
	inSet map[string]struct{} `json:"-"`
}

// Fact 审核事实数据
type Fact map[string]interface{}

// Result 规则执行结果
type Result struct {
	Matched bool
	Action  RuleAction
	Reason  string // 规则名称或失败原因
}


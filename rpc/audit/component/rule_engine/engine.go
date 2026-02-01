package rule_engine

import (
	"encoding/json"
	"piaowu/common/model/audit"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"gorm.io/gorm"
)

type RuleEngine struct {
	db    *gorm.DB
	rules map[string][]*Rule // map[BizType][]Rule
	mu    sync.RWMutex
}

func NewRuleEngine(db *gorm.DB) *RuleEngine {
	re := &RuleEngine{
		db:    db,
		rules: make(map[string][]*Rule),
	}
	// 启动时加载规则
	if err := re.Reload(); err != nil {
		log.Printf("RuleEngine load rules failed: %v", err)
	}
	return re
}

// Reload 从数据库重新加载规则
func (re *RuleEngine) Reload() error {
	var configs []audit.RuleConfig
	if err := re.db.Where("status = ?", 1).Find(&configs).Error; err != nil {
		return err
	}

	newRules := make(map[string][]*Rule)
	for _, cfg := range configs {
		var expr RuleExpression
		if err := json.Unmarshal([]byte(cfg.Expression), &expr); err != nil {
			log.Printf("Rule %d expression invalid: %v", cfg.ID, err)
			continue
		}
		if err := prepareExpression(&expr); err != nil {
			log.Printf("Rule %d expression prepare failed: %v", cfg.ID, err)
			continue
		}

		rule := &Rule{
			ID:         cfg.ID,
			BizType:    cfg.BizType,
			RuleName:   cfg.RuleName,
			Expression: &expr,
			Action:     RuleAction(cfg.Action),
			Priority:   cfg.Priority,
		}
		newRules[cfg.BizType] = append(newRules[cfg.BizType], rule)
	}

	// 按优先级排序 (高优先级在前)
	for _, rules := range newRules {
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Priority > rules[j].Priority
		})
	}

	re.mu.Lock()
	re.rules = newRules
	re.mu.Unlock()

	log.Printf("RuleEngine reloaded %d rules", len(configs))
	return nil
}

// Execute 执行规则引擎
func (re *RuleEngine) Execute(bizType string, fact Fact) Result {
	re.mu.RLock()
	rules, ok := re.rules[bizType]
	re.mu.RUnlock()

	if !ok || len(rules) == 0 {
		return Result{Matched: false}
	}

	for _, rule := range rules {
		if re.eval(rule.Expression, fact) {
			return Result{
				Matched: true,
				Action:  rule.Action,
				Reason:  rule.RuleName,
			}
		}
	}

	return Result{Matched: false}
}

// eval 递归评估表达式
func (re *RuleEngine) eval(expr *RuleExpression, fact Fact) bool {
	if expr == nil {
		return true
	}

	switch strings.ToUpper(expr.Op) {
	case "AND":
		for _, sub := range expr.SubRules {
			if !re.eval(sub, fact) {
				return false
			}
		}
		return true
	case "OR":
		for _, sub := range expr.SubRules {
			if re.eval(sub, fact) {
				return true
			}
		}
		return false
	default:
		return re.evalLeaf(expr, fact)
	}
}

// evalLeaf 评估叶子节点
func (re *RuleEngine) evalLeaf(expr *RuleExpression, fact Fact) bool {
	if expr == nil {
		return false
	}
	factVal, exists := fact[expr.Field]
	if !exists {
		// 如果字段不存在，视作不匹配 (或者可以根据策略报错)
		return false
	}

	// 使用预编译信息进行比较，提升性能
	return compareWithExpr(factVal, expr)
}

// prepareExpression 预处理表达式（递归）
// 关键算法说明：
// - REGEX 预编译，避免运行时重复编译
// - IN 预生成集合，避免字符串包含误判
// - 递归处理子表达式
func prepareExpression(expr *RuleExpression) error {
	if expr == nil {
		return nil
	}
	if len(expr.SubRules) > 0 {
		for _, sub := range expr.SubRules {
			if err := prepareExpression(sub); err != nil {
				return err
			}
		}
		return nil
	}

	switch strings.ToUpper(expr.Op) {
	case "REGEX":
		pattern := fmt.Sprintf("%v", expr.Value)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		expr.regex = re
	case "IN":
		expr.inSet = buildInSet(expr.Value)
	}
	return nil
}

func buildInSet(val interface{}) map[string]struct{} {
	set := make(map[string]struct{})
	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			set[fmt.Sprintf("%v", item)] = struct{}{}
		}
	case []string:
		for _, item := range v {
			set[item] = struct{}{}
		}
	case string:
		parts := strings.Split(v, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				set[p] = struct{}{}
			}
		}
	default:
		set[fmt.Sprintf("%v", v)] = struct{}{}
	}
	return set
}

// compareWithExpr 比较逻辑（使用预编译字段）
func compareWithExpr(factVal interface{}, expr *RuleExpression) bool {
	op := strings.ToUpper(expr.Op)
	// 处理数值比较 (统一转为 float64)
	f1, ok1 := toFloat(factVal)
	f2, ok2 := toFloat(expr.Value)

	if ok1 && ok2 {
		switch op {
		case ">":
			return f1 > f2
		case ">=":
			return f1 >= f2
		case "<":
			return f1 < f2
		case "<=":
			return f1 <= f2
		case "==":
			return f1 == f2
		case "!=":
			return f1 != f2
		}
	}

	// 字符串处理
	s1 := fmt.Sprintf("%v", factVal)
	s2 := fmt.Sprintf("%v", expr.Value)

	switch op {
	case "==":
		return s1 == s2
	case "!=":
		return s1 != s2
	case "IN":
		if expr.inSet != nil {
			_, ok := expr.inSet[s1]
			return ok
		}
		return strings.Contains(s2, s1)
	case "REGEX":
		if expr.regex != nil {
			return expr.regex.MatchString(s1)
		}
		matched, _ := regexp.MatchString(s2, s1)
		return matched
	}

	return false
}

func toFloat(v interface{}) (float64, bool) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return val.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), true
	}
	return 0, false
}


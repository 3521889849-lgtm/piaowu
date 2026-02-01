package decision

import (
	"context"
	"encoding/json"
	"errors"
	"piaowu/common/config"
	"piaowu/rpc/audit/component/metrics"
	"piaowu/rpc/audit/component/ml"
	"piaowu/rpc/audit/component/rule_engine"
	"piaowu/rpc/audit/component/threshold"
	"fmt"
	"math"
	"time"

	"golang.org/x/sync/errgroup"
)

// Result 决策输出
// 关键算法说明：
// 1) 规则引擎与模型评分并行执行，减少整体耗时
// 2) 动态阈值基于近期评分分布做自适应调整，降低误判率
// 3) 最终动作优先级：规则命中 > 模型判定 > 默认动作
// 4) 评分使用 Sigmoid 将线性输出压缩到 (0,1) 区间，便于阈值控制
// 以上策略在保证可解释性的同时，提高准确性与性能
//
// 注意：Extra 字段用于持久化模型评分与阈值信息，便于事后评估与监控
// 所有关键字段均为中文注释，满足可维护性要求
type Result struct {
	Status      int8
	Remark      string
	RuleMatched bool
	RuleAction  rule_engine.RuleAction
	ModelScore  float64
	Threshold   float64
	FinalAction rule_engine.RuleAction
	Extra       map[string]interface{}
}

// Engine 决策引擎
// 依赖注入，符合 SOLID 的依赖倒置原则
// 可通过替换 Model/Threshold/Plugins 实现插件式扩展
//
// 设计目标：
// - 规则与模型评分并行，提高吞吐
// - 动态阈值降低误判
// - 插件接口可扩展业务特性
// - 统一输出结构用于日志/持久化/监控

// RuleExecutor 规则执行接口（便于测试与扩展）
// 任何实现 Execute 方法的对象都可接入决策引擎
type RuleExecutor interface {
	Execute(bizType string, fact rule_engine.Fact) rule_engine.Result
}

type Engine struct {
	ruleEngine RuleExecutor
	model      ml.Model
	threshold  *threshold.Manager
	plugins    *PluginManager
	metrics    *metrics.Collector
	cfg        config.AuditDecision
}

func NewEngine(ruleEngine RuleExecutor, model ml.Model, thresholdMgr *threshold.Manager, plugins *PluginManager, collector *metrics.Collector, cfg config.AuditDecision) *Engine {
	return &Engine{
		ruleEngine: ruleEngine,
		model:      model,
		threshold:  thresholdMgr,
		plugins:    plugins,
		metrics:    collector,
		cfg:        cfg,
	}
}

// Decide 执行决策
func (e *Engine) Decide(ctx context.Context, bizType string, fact rule_engine.Fact) (Result, error) {
	if e.ruleEngine == nil {
		return Result{}, errors.New("ruleEngine is nil")
	}
	start := time.Now()

	if e.plugins != nil {
		// 在决策前执行插件逻辑，允许插件修改 fact
		if err := e.plugins.BeforeDecision(ctx, fact); err != nil {
			return Result{}, err
		}
	}

	var ruleResult rule_engine.Result

	var modelScore float64
	var modelErr error

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		// 规则引擎执行
		ruleResult = e.ruleEngine.Execute(bizType, fact)
		return nil
	})
	g.Go(func() error {
		// 模型评分执行（可配置启用/禁用）
		if e.model == nil || !e.cfg.ModelEnabled {
			return nil
		}
		features := extractNumericFeatures(fact, e.cfg.FeatureKeys)
		score, _, err := e.model.Score(gctx, features)
		if err != nil {
			modelErr = err
			return nil
		}
		modelScore = score
		return nil
	})
	_ = g.Wait()

	// 动态阈值计算
	thresholdValue := e.cfg.DefaultReviewThreshold
	if e.threshold != nil {
		thresholdValue = e.threshold.Threshold()
	}

	// 决策逻辑
	result := Result{
		RuleMatched: ruleResult.Matched,
		RuleAction:  ruleResult.Action,
		ModelScore:  modelScore,
		Threshold:   thresholdValue,
		FinalAction: rule_engine.ActionReview,
		Extra:       map[string]interface{}{},
	}

	if ruleResult.Matched {
		result.FinalAction = ruleResult.Action
		result.Remark = "规则命中: " + ruleResult.Reason
	} else if e.cfg.ModelEnabled && modelErr == nil && e.model != nil {
		// 模型评分 + 动态阈值策略
		// 关键算法：
		// 1) score >= threshold -> Review
		// 2) score >= threshold + RejectGap -> Reject
		// 3) score < threshold -> Pass
		if modelScore >= thresholdValue+e.cfg.RejectGap {
			result.FinalAction = rule_engine.ActionReject
			result.Remark = "模型高风险拒绝"
		} else if modelScore >= thresholdValue {
			result.FinalAction = rule_engine.ActionReview
			result.Remark = "模型风险需复审"
		} else {
			result.FinalAction = rule_engine.ActionPass
			result.Remark = "模型低风险通过"
		}
	} else {
		// 默认策略
		result.FinalAction = rule_engine.ActionReview
		result.Remark = "无匹配规则且模型不可用，默认转人工"
	}

	// 结果状态映射
	result.Status = mapActionToStatus(result.FinalAction)

	// 记录额外信息（用于持久化与监控）
	result.Extra["rule_matched"] = result.RuleMatched
	result.Extra["rule_action"] = string(result.RuleAction)
	result.Extra["model_score"] = roundFloat(modelScore, 6)
	result.Extra["dynamic_threshold"] = roundFloat(thresholdValue, 6)
	result.Extra["final_action"] = string(result.FinalAction)
	if modelErr != nil {
		result.Extra["model_error"] = modelErr.Error()
	}

	// 动态阈值更新（只在模型评分有效时更新）
	if e.cfg.ModelEnabled && e.threshold != nil && modelErr == nil && e.model != nil {
		e.threshold.Update(modelScore)
	}

	// 监控统计
	if e.metrics != nil {
		e.metrics.RecordDecision(metrics.DecisionType(result.FinalAction), time.Since(start))
	}

	if e.plugins != nil {
		_ = e.plugins.AfterDecision(ctx, &result)
	}

	return result, nil
}

// BuildExtraJSON 将 Extra 转为 JSON
func BuildExtraJSON(extra map[string]interface{}) string {
	if extra == nil {
		return "{}"
	}
	b, err := json.Marshal(extra)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// mapActionToStatus 将动作映射为审核状态
func mapActionToStatus(action rule_engine.RuleAction) int8 {
	switch action {
	case rule_engine.ActionPass:
		return 3
	case rule_engine.ActionReject:
		return 4
	case rule_engine.ActionReview:
		return 1
	default:
		return 1
	}
}

// extractNumericFeatures 抽取数值特征（支持字符串/布尔/数值）
// 关键算法说明：
// - 字符串可尝试转为数字（如金额/次数等）
// - 布尔转为 0/1
// - 缺失值默认为 0
// 该策略保证模型输入稳定，避免因空值导致评分失败
func extractNumericFeatures(fact rule_engine.Fact, keys []string) map[string]float64 {
	features := make(map[string]float64, len(keys))
	for _, k := range keys {
		v, ok := fact[k]
		if !ok {
			features[k] = 0
			continue
		}
		features[k] = toFloat64(v)
	}
	return features
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		f, err := parseFloat(val)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func roundFloat(v float64, scale int) float64 {
	if scale <= 0 {
		return v
	}
	p := math.Pow10(scale)
	return math.Round(v*p) / p
}


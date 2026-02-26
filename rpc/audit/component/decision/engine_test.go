package decision

import (
	"context"
	"piaowu/common/config"
	"piaowu/rpc/audit/component/metrics"
	"piaowu/rpc/audit/component/ml"
	"piaowu/rpc/audit/component/rule_engine"
	"piaowu/rpc/audit/component/threshold"
	"testing"
)

// fakeRuleEngine 用于测试的规则执行器
type fakeRuleEngine struct {
	result rule_engine.Result
}

func (f *fakeRuleEngine) Execute(bizType string, fact rule_engine.Fact) rule_engine.Result {
	return f.result
}

// fakeModel 用于测试的模  ?
type fakeModel struct {
	score float64
}

func (m *fakeModel) Score(ctx context.Context, features map[string]float64) (float64, ml.ResultExplain, error) {
	return m.score, ml.ResultExplain{}, nil
}

func TestEngine_Decide(t *testing.T) {
	cfg := config.AuditDecision{
		ModelEnabled:           true,
		DefaultReviewThreshold: 0.6,
		RejectGap:              0.2,
		FeatureKeys:            []string{"order_amount"},
	}

	thresholdMgr := threshold.NewManager(threshold.Config{Default: 0.6, Percentile: 0.9, WindowSize: 10})
	collector := metrics.NewCollector(10)

	// 场景1：规则命中优  ?
	re1 := &fakeRuleEngine{result: rule_engine.Result{Matched: true, Action: rule_engine.ActionReject, Reason: "High Amount"}}
	engine1 := NewEngine(re1, &fakeModel{score: 0.1}, thresholdMgr, NewPluginManager(), collector, cfg)
	res1, err := engine1.Decide(context.Background(), "TICKET_ORDER", rule_engine.Fact{"order_amount": 1000})
	if err != nil {
		t.Fatalf("Decide error: %v", err)
	}
	if res1.FinalAction != rule_engine.ActionReject {
		t.Fatalf("want reject, got %v", res1.FinalAction)
	}

	// 场景2：模型高风险拒绝
	re2 := &fakeRuleEngine{result: rule_engine.Result{Matched: false}}
	engine2 := NewEngine(re2, &fakeModel{score: 0.9}, thresholdMgr, NewPluginManager(), collector, cfg)
	res2, _ := engine2.Decide(context.Background(), "TICKET_ORDER", rule_engine.Fact{"order_amount": 500})
	if res2.FinalAction != rule_engine.ActionReject {
		t.Fatalf("want reject, got %v", res2.FinalAction)
	}

	// 场景3：模型低风险通过
	engine3 := NewEngine(re2, &fakeModel{score: 0.1}, thresholdMgr, NewPluginManager(), collector, cfg)
	res3, _ := engine3.Decide(context.Background(), "TICKET_ORDER", rule_engine.Fact{"order_amount": 50})
	if res3.FinalAction != rule_engine.ActionPass {
		t.Fatalf("want pass, got %v", res3.FinalAction)
	}
}

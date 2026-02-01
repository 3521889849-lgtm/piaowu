package config

var Cfg = new(Config)

type Config struct {
	Mysql
	Redis
	Coupon
	Audit
}

type Mysql struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type Redis struct {
	Host     string
	Port     int
	Password string
	Database int
}

type Coupon struct {
	AntiBrushLimit  int
	AntiBrushExpire int
	AesKey          string
}

// Audit 审核模块配置
// 通过配置实现模型、阈值、监控、告警与并发策略的灵活扩展

type Audit struct {
	Decision  AuditDecision
	Model     AuditModel
	Threshold AuditThreshold
	Metrics   AuditMetrics
	Alert     AuditAlert
}

// AuditDecision 决策策略配置

type AuditDecision struct {
	ModelEnabled           bool
	DefaultReviewThreshold float64
	RejectGap              float64
	FeatureKeys            []string
}

// AuditModel 模型参数配置

type AuditModel struct {
	Enabled bool
	Weights map[string]float64
	Bias    float64
}

// AuditThreshold 动态阈值配置

type AuditThreshold struct {
	WindowSize int
	Percentile float64
	Min        float64
	Max        float64
	Default    float64
}

// AuditMetrics 监控配置

type AuditMetrics struct {
	WindowSeconds int
}

// AuditAlert 告警配置

type AuditAlert struct {
	Enabled         bool
	IntervalSeconds int
	MaxLatencyMs    float64
	MinAccuracy     float64
	MinThroughput   float64
	WebhookURL      string
}

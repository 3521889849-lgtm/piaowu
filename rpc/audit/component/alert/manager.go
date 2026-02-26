package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"piaowu/rpc/audit/component/metrics"
	"time"
)

// Config 告警配置
// 触发条件可按需扩展

type Config struct {
	Enabled         bool
	IntervalSeconds int
	MaxLatencyMs    float64
	MinAccuracy     float64
	MinThroughput   float64
	WebhookURL      string
}

// Manager 告警管理
// 周期性拉取监控快照，触发自动化报警

type Manager struct {
	cfg       Config
	collector *metrics.Collector
	logger    *log.Logger
}

func NewManager(cfg Config, collector *metrics.Collector, logger *log.Logger) *Manager {
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 30
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Manager{cfg: cfg, collector: collector, logger: logger}
}

// Start 启动告警检查
func (m *Manager) Start(ctx context.Context) {
	if m == nil || m.collector == nil || !m.cfg.Enabled {
		return
	}
	ticker := time.NewTicker(time.Duration(m.cfg.IntervalSeconds) * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				snapshot := m.collector.Snapshot()
				m.checkAndAlert(snapshot)
			}
		}
	}()
}

func (m *Manager) checkAndAlert(s metrics.Snapshot) {
	// 触发条件判断
	var alerts []string
	if m.cfg.MaxLatencyMs > 0 && s.AvgLatencyMs > m.cfg.MaxLatencyMs {
		alerts = append(alerts, "平均延迟超限")
	}
	if m.cfg.MinAccuracy > 0 && s.Accuracy < m.cfg.MinAccuracy {
		alerts = append(alerts, "准确率低于阈值")
	}
	if m.cfg.MinThroughput > 0 && s.ThroughputQPS < m.cfg.MinThroughput {
		alerts = append(alerts, "吞吐量低于阈值")
	}

	if len(alerts) == 0 {
		return
	}

	payload := map[string]interface{}{
		"alerts":    alerts,
		"snapshot":  s,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	m.logger.Printf("[ALERT] %v", alerts)

	if m.cfg.WebhookURL != "" {
		b, _ := json.Marshal(payload)
		_, _ = http.Post(m.cfg.WebhookURL, "application/json", bytes.NewReader(b))
	}
}

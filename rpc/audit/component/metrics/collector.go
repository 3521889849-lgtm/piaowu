package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// DecisionType 决策类型

type DecisionType string

const (
	DecisionPass   DecisionType = "Pass"
	DecisionReject DecisionType = "Reject"
	DecisionReview DecisionType = "Review"
)

// Snapshot 监控快照

type Snapshot struct {
	TotalCount    int64
	AvgLatencyMs  float64
	ThroughputQPS float64
	Accuracy      float64
	ManualTotal   int64
	ManualCorrect int64
	UpdatedAt     time.Time
}

// Collector 监控数据采集
// 关键算法说明  ?
// - 吞吐量使用滑动时间窗  ?秒级  ?统计
// - 延迟使用累计求和/计数求平  ?
// - 准确率基于人工审核结果校验模型建  ?

type Collector struct {
	totalCount   int64
	latencySumNs int64
	latencyCount int64

	manualTotal   int64
	manualCorrect int64

	window *throughputWindow
}

// DefaultCollector 全局监控实例

var DefaultCollector = NewCollector(60)

func NewCollector(windowSeconds int) *Collector {
	if windowSeconds <= 0 {
		windowSeconds = 60
	}
	return &Collector{window: newThroughputWindow(windowSeconds)}
}

// RecordDecision 记录决策与延  ?
func (c *Collector) RecordDecision(decision DecisionType, latency time.Duration) {
	atomic.AddInt64(&c.totalCount, 1)
	atomic.AddInt64(&c.latencySumNs, latency.Nanoseconds())
	atomic.AddInt64(&c.latencyCount, 1)
	c.window.Add(time.Now())
}

// RecordManualOutcome 记录人工审核结果与模型建议的一致  ?
// 仅当 suggested   ?Pass/Reject 时参与准确率统计
func (c *Collector) RecordManualOutcome(suggested DecisionType, final DecisionType) {
	if suggested != DecisionPass && suggested != DecisionReject {
		return
	}
	atomic.AddInt64(&c.manualTotal, 1)
	if suggested == final {
		atomic.AddInt64(&c.manualCorrect, 1)
	}
}

// Snapshot 获取监控快照
func (c *Collector) Snapshot() Snapshot {
	total := atomic.LoadInt64(&c.totalCount)
	latSum := atomic.LoadInt64(&c.latencySumNs)
	latCount := atomic.LoadInt64(&c.latencyCount)
	manualTotal := atomic.LoadInt64(&c.manualTotal)
	manualCorrect := atomic.LoadInt64(&c.manualCorrect)

	avgLatency := 0.0
	if latCount > 0 {
		avgLatency = float64(latSum) / float64(latCount) / 1e6
	}

	accuracy := 0.0
	if manualTotal > 0 {
		accuracy = float64(manualCorrect) / float64(manualTotal)
	}

	return Snapshot{
		TotalCount:    total,
		AvgLatencyMs:  avgLatency,
		ThroughputQPS: c.window.QPS(time.Now()),
		Accuracy:      accuracy,
		ManualTotal:   manualTotal,
		ManualCorrect: manualCorrect,
		UpdatedAt:     time.Now(),
	}
}

// throughputWindow 秒级滑动窗口

type throughputWindow struct {
	mu      sync.Mutex
	size    int
	buckets []bucket
}

type bucket struct {
	ts    int64
	count int64
}

func newThroughputWindow(size int) *throughputWindow {
	return &throughputWindow{
		size:    size,
		buckets: make([]bucket, size),
	}
}

func (w *throughputWindow) Add(t time.Time) {
	sec := t.Unix()
	idx := int(sec % int64(w.size))
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buckets[idx].ts != sec {
		w.buckets[idx].ts = sec
		w.buckets[idx].count = 0
	}
	w.buckets[idx].count++
}

func (w *throughputWindow) QPS(now time.Time) float64 {
	sec := now.Unix()
	w.mu.Lock()
	defer w.mu.Unlock()
	var total int64
	for i := 0; i < w.size; i++ {
		if sec-w.buckets[i].ts < int64(w.size) {
			total += w.buckets[i].count
		}
	}
	return float64(total) / float64(w.size)
}

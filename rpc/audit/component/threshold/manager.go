package threshold

import (
	"math"
	"sort"
	"sync"
)

// Config 动态阈值配  ?
// Percentile 取值范  ?0,1)，例  ?0.9 表示  ?90 分位
// WindowSize 为窗口大小，越大越稳  ?

type Config struct {
	WindowSize int
	Percentile float64
	Min        float64
	Max        float64
	Default    float64
}

// Manager 动态阈值管理器
// 关键算法说明  ?
// - 使用滑动窗口保存近期评分
// - 基于分位数进行阈值计算，避免被极端值干  ?
// - 最终阈值通过 Min/Max 进行夹断，防止漂移过  ?

type Manager struct {
	mu     sync.RWMutex
	cfg    Config
	scores []float64
	idx    int
	filled bool
}

func NewManager(cfg Config) *Manager {
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 200
	}
	if cfg.Percentile <= 0 || cfg.Percentile >= 1 {
		cfg.Percentile = 0.9
	}
	return &Manager{
		cfg:    cfg,
		scores: make([]float64, cfg.WindowSize),
	}
}

// Update 写入最新评  ?
func (m *Manager) Update(score float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scores[m.idx] = score
	m.idx = (m.idx + 1) % len(m.scores)
	if m.idx == 0 {
		m.filled = true
	}
}

// Threshold 计算动态阈  ?
func (m *Manager) Threshold() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.filled && m.idx == 0 {
		return clamp(m.cfg.Default, m.cfg.Min, m.cfg.Max)
	}

	var data []float64
	if m.filled {
		data = append([]float64{}, m.scores...)
	} else {
		data = append([]float64{}, m.scores[:m.idx]...)
	}

	if len(data) == 0 {
		return clamp(m.cfg.Default, m.cfg.Min, m.cfg.Max)
	}

	sort.Float64s(data)
	pos := int(math.Ceil(float64(len(data))*m.cfg.Percentile)) - 1
	if pos < 0 {
		pos = 0
	}
	if pos >= len(data) {
		pos = len(data) - 1
	}

	return clamp(data[pos], m.cfg.Min, m.cfg.Max)
}

func clamp(v, min, max float64) float64 {
	if min != 0 && v < min {
		return min
	}
	if max != 0 && v > max {
		return max
	}
	return v
}


package threshold

import "testing"

func TestManager_Threshold(t *testing.T) {
	mgr := NewManager(Config{WindowSize: 5, Percentile: 0.8, Min: 0.2, Max: 0.9, Default: 0.6})

	// 无数据时使用默认  ?
	if v := mgr.Threshold(); v != 0.6 {
		t.Fatalf("want default 0.6, got %v", v)
	}

	// 写入评分
	scores := []float64{0.1, 0.2, 0.3, 0.8, 0.9}
	for _, s := range scores {
		mgr.Update(s)
	}

	v := mgr.Threshold()
	if v < 0.3 || v > 0.9 {
		t.Fatalf("threshold out of range: %v", v)
	}
}


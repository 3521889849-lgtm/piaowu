package ml

import (
	"context"
	"errors"
	"math"
)

// Model 模型接口（可替换为外部服务或本地模型  ?
// 关键算法说明  ?
// - 本实现为线性模  ?+ Sigmoid 输出
// - Sigmoid 可将分数压缩  ?(0,1)，适合阈值策  ?
// - 该模型具备可解释性：可输出每个特征的贡献  ?

// ResultExplain 解释信息
// key: 特征名，value: 贡献  ?

type ResultExplain map[string]float64

// Model 模型接口
// Score 返回评分(0-1)、特征贡献、错  ?

type Model interface {
	Score(ctx context.Context, features map[string]float64) (float64, ResultExplain, error)
}

// LinearModel 线性模型（可用于快速部署或作为占位模型  ?
type LinearModel struct {
	Weights map[string]float64
	Bias    float64
}

func NewLinearModel(weights map[string]float64, bias float64) *LinearModel {
	return &LinearModel{Weights: weights, Bias: bias}
}

// Score 评分
// 关键算法说明  ?
// 1) 线性部分：sum(w_i * x_i) + bias
// 2) 非线性部分：sigmoid(linear)
// 3) 结果范围  ?0,1)
func (m *LinearModel) Score(ctx context.Context, features map[string]float64) (float64, ResultExplain, error) {
	if m == nil {
		return 0, nil, errors.New("model is nil")
	}
	if features == nil {
		return 0, nil, errors.New("features is nil")
	}
	linear := m.Bias
	explain := make(ResultExplain, len(features))
	for k, v := range features {
		w := m.Weights[k]
		contrib := w * v
		linear += contrib
		explain[k] = contrib
	}
	return sigmoid(linear), explain, nil
}

func sigmoid(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}

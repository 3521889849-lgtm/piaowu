/*
 * 规则型 NLU (自然语言理解)
 *
 * 功能说明：
 * - 提取用户文本中的意图 (Intent) 和槽位 (Slots)
 * - 使用正则表达式和关键词匹配（轻量级方案）
 * - 支持日期推断（今天/明天 -> YYYY-MM-DD）
 */
package nlu

import (
	"context"
	"example_shop/internal/gateway/http/dto"
	"regexp"
	"strings"
	"time"
)

// 预定义意图常量
const (
	IntentUnknown     = "UNKNOWN"      // 未知意图
	IntentSearchTrain = "SEARCH_TRAIN" // 查车次
	IntentTrainDetail = "TRAIN_DETAIL" // 查车次详情
	IntentListOrders  = "LIST_ORDERS"  // 查订单
)

// Recognizer NLU 识别器接口
type Recognizer interface {
	Recognize(ctx context.Context, text string) *dto.AssistantIntent
}

// ruleBased 基于规则的实现
type ruleBased struct {
	reFromTo  *regexp.Regexp // 匹配 "从X到Y"
	reDate    *regexp.Regexp // 匹配 "YYYY-MM-DD"
	reTrainID *regexp.Regexp // 匹配 "train_id:xxx"
}

// NewRuleBased 创建规则 NLU 实例
func NewRuleBased() Recognizer {
	return &ruleBased{
		reFromTo:  regexp.MustCompile(`从(?P<dep>[^到\s]{1,16})到(?P<arr>[^\s]{1,16})`),
		reDate:    regexp.MustCompile(`(20[0-9]{2}-[0-9]{2}-[0-9]{2})`),
		reTrainID: regexp.MustCompile(`(?i)train[_-]?id[:：\s]*([a-z0-9_-]{3,64})`),
	}
}

// Recognize 执行识别逻辑
// 匹配优先级：
// 1. 查订单
// 2. 查详情 (train_id)
// 3. 查车次 (从...到...)
func (r *ruleBased) Recognize(ctx context.Context, text string) *dto.AssistantIntent {
	_ = ctx
	raw := strings.TrimSpace(text)
	lower := strings.ToLower(raw)

	intent := &dto.AssistantIntent{
		Name:       IntentUnknown,
		Confidence: 0.4,
		Slots:      map[string]string{},
	}

	// 1. 匹配查订单
	// 关键词：订单 + (列表|全部|我的)
	if strings.Contains(raw, "订单") && (strings.Contains(raw, "列表") || strings.Contains(raw, "全部") || strings.Contains(raw, "我的")) {
		intent.Name = IntentListOrders
		intent.Confidence = 0.9
		return intent
	}

	// 2. 匹配车次详情
	// 关键词：详情, train_detail + 正则提取 train_id
	if strings.Contains(raw, "详情") || strings.Contains(raw, "车次详情") || strings.Contains(raw, "train_detail") {
		if m := r.reTrainID.FindStringSubmatch(raw); len(m) >= 2 {
			intent.Name = IntentTrainDetail
			intent.Confidence = 0.9
			intent.Slots["train_id"] = m[1]
			return intent
		}
	}

	// 3. 匹配查车次
	// 关键词：查/余票/车次 + 正则 "从X到Y"
	if strings.Contains(raw, "查") || strings.Contains(raw, "余票") || strings.Contains(raw, "车次") {
		m := r.reFromTo.FindStringSubmatch(raw)
		if len(m) > 0 {
			dep, arr := "", ""
			for i, name := range r.reFromTo.SubexpNames() {
				if name == "dep" && i < len(m) {
					dep = strings.TrimSpace(m[i])
				}
				if name == "arr" && i < len(m) {
					arr = strings.TrimSpace(m[i])
				}
			}
			if dep != "" && arr != "" {
				intent.Name = IntentSearchTrain
				intent.Confidence = 0.85
				intent.Slots["departure_station"] = dep
				intent.Slots["arrival_station"] = arr
			}
		}

		// 日期提取
		if d := r.reDate.FindString(raw); d != "" {
			intent.Slots["travel_date"] = d
		} else if strings.Contains(raw, "明天") {
			intent.Slots["travel_date"] = time.Now().Add(24 * time.Hour).Format("2006-01-02")
		} else if strings.Contains(raw, "今天") {
			intent.Slots["travel_date"] = time.Now().Format("2006-01-02")
		}

		if intent.Name == IntentSearchTrain {
			return intent
		}
	}

	// 兜底匹配 train_id
	if strings.Contains(lower, "train_id") {
		if m := r.reTrainID.FindStringSubmatch(raw); len(m) >= 2 {
			intent.Name = IntentTrainDetail
			intent.Confidence = 0.8
			intent.Slots["train_id"] = m[1]
			return intent
		}
	}

	return intent
}

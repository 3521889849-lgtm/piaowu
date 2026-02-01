package nlu

import (
	"context"
	"testing"
)

func TestRuleBasedRecognize_SearchTrain(t *testing.T) {
	r := NewRuleBased()
	intent := r.Recognize(context.Background(), "帮我查 2026-02-03 从北京到上海 的车次")
	if intent.Name != IntentSearchTrain {
		t.Fatalf("expected %s, got %s", IntentSearchTrain, intent.Name)
	}
	if intent.Slots["departure_station"] != "北京" {
		t.Fatalf("departure_station mismatch: %q", intent.Slots["departure_station"])
	}
	if intent.Slots["arrival_station"] != "上海" {
		t.Fatalf("arrival_station mismatch: %q", intent.Slots["arrival_station"])
	}
	if intent.Slots["travel_date"] != "2026-02-03" {
		t.Fatalf("travel_date mismatch: %q", intent.Slots["travel_date"])
	}
}

func TestRuleBasedRecognize_ListOrders(t *testing.T) {
	r := NewRuleBased()
	intent := r.Recognize(context.Background(), "给我看一下我的订单列表")
	if intent.Name != IntentListOrders {
		t.Fatalf("expected %s, got %s", IntentListOrders, intent.Name)
	}
}


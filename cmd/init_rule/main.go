package main

import (
	"log"
	"piaowu/common/db"
	_ "piaowu/common/init"
	"piaowu/common/model/audit"
	"time"
)

func main() {
	if db.DB == nil {
		log.Fatal("DB not initialized")
	}

	// 先清理旧的同名测试规则，防止重复
	db.DB.Where("rule_name = ?", "大额订单拦截").Delete(&audit.RuleConfig{})

	rule := audit.RuleConfig{
		BizType:    "TICKET_ORDER",
		RuleName:   "大额订单拦截",
		Expression: `{"field":"order_amount", "op":">", "value":2000}`,
		Action:     "Reject",
		Priority:   10,
		Status:     1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := db.DB.Create(&rule).Error; err != nil {
		log.Fatalf("插入规则失败: %v", err)
	}
	log.Printf("✅ 成功插入测试规则: [大额订单拦截] 金额>2000 -> Reject")
}

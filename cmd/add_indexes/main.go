package main

import (
	"piaowu/common/config"
	"piaowu/common/db"
	"fmt"
	"log"

	"gorm.io/gorm"
)

func main() {
	// 初始化配  ?
	config.ViperInit()

	// 初始化数据库
	if err := db.MysqlInit(); err != nil {
		log.Fatal("数据库初始化失败:", err)
	}

	fmt.Println("开始添加索  ?..")

	// 添加索引的SQL语句
	sqls := []string{
		// 审核主表索引
		"CREATE INDEX idx_created_at ON audit_mains(created_at DESC)",
		"CREATE INDEX idx_deleted_at_main ON audit_mains(deleted_at)",
		
		// 车票订单表索  ?
		"CREATE INDEX idx_audit_main_id_ticket ON audit_ticket_orders(audit_main_id)",
		"CREATE INDEX idx_deleted_at_ticket ON audit_ticket_orders(deleted_at)",
		
		// 酒店订单表索  ?
		"CREATE INDEX idx_audit_main_id_hotel ON audit_hotel_orders(audit_main_id)",
		"CREATE INDEX idx_deleted_at_hotel ON audit_hotel_orders(deleted_at)",
	}

	for i, sql := range sqls {
		fmt.Printf("[%d/%d] 执行: %s\n", i+1, len(sqls), sql)
		if err := db.DB.Exec(sql).Error; err != nil {
			log.Printf("警告: 索引可能已存在或创建失败: %v\n", err)
		} else {
			fmt.Println("  ?成功")
		}
	}

	fmt.Println("\n索引添加完成  ?)
	
	// 查看索引信息
	fmt.Println("\n=== 审核主表索引 ===")
	showIndexes(db.DB, "audit_mains")
	
	fmt.Println("\n=== 车票订单表索  ?===")
	showIndexes(db.DB, "audit_ticket_orders")
	
	fmt.Println("\n=== 酒店订单表索  ?===")
	showIndexes(db.DB, "audit_hotel_orders")
}

func showIndexes(database *gorm.DB, tableName string) {
	type IndexInfo struct {
		Table      string `gorm:"column:Table"`
		NonUnique  int    `gorm:"column:Non_unique"`
		KeyName    string `gorm:"column:Key_name"`
		SeqInIndex int    `gorm:"column:Seq_in_index"`
		ColumnName string `gorm:"column:Column_name"`
	}
	
	var indexes []IndexInfo
	database.Raw("SHOW INDEX FROM " + tableName).Scan(&indexes)
	
	for _, idx := range indexes {
		fmt.Printf("  - %s (%s)\n", idx.KeyName, idx.ColumnName)
	}
}


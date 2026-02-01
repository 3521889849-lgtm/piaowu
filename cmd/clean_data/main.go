package main

import (
	"log"
	"piaowu/common/config"
	"piaowu/common/db"

	_ "piaowu/common/init"
)

func main() {
	log.Println("开始清理无效业务类型数据...")

	// 有效的业务类型: 1=车票订单, 2=酒店订单, 3=酒店入驻
	validTypes := []int{1, 2, 3}

	// 1. 先查询要删除的数据量
	var count int64
	if err := db.DB.Table("audit_mains").
		Where("business_type NOT IN ?", validTypes).
		Count(&count).Error; err != nil {
		log.Fatal("查询失败:", err)
	}
	log.Printf("发现 %d 条无效业务类型数据\n", count)

	if count == 0 {
		log.Println("无需清理，数据库中只有有效数据")
		return
	}

	// 2. 获取要删除的 audit_main IDs
	var invalidIDs []uint64
	if err := db.DB.Table("audit_mains").
		Select("id").
		Where("business_type NOT IN ?", validTypes).
		Pluck("id", &invalidIDs).Error; err != nil {
		log.Fatal("获取无效ID失败:", err)
	}

	// 3. 删除关联的子表数据
	log.Println("删除关联的操作日志...")
	if err := db.DB.Exec("DELETE FROM audit_operation_logs WHERE audit_main_id IN ?", invalidIDs).Error; err != nil {
		log.Printf("删除操作日志失败: %v\n", err)
	}

	log.Println("删除关联的车票订单...")
	if err := db.DB.Exec("DELETE FROM audit_ticket_orders WHERE audit_main_id IN ?", invalidIDs).Error; err != nil {
		log.Printf("删除车票订单失败: %v\n", err)
	}

	log.Println("删除关联的酒店订单...")
	if err := db.DB.Exec("DELETE FROM audit_hotel_orders WHERE audit_main_id IN ?", invalidIDs).Error; err != nil {
		log.Printf("删除酒店订单失败: %v\n", err)
	}

	// 4. 删除主表数据
	log.Println("删除主表无效数据...")
	result := db.DB.Exec("DELETE FROM audit_mains WHERE business_type NOT IN ?", validTypes)
	if result.Error != nil {
		log.Fatal("删除主表数据失败:", result.Error)
	}

	log.Printf("✅ 清理完成！共删除 %d 条无效记录\n", result.RowsAffected)

	// 5. 显示清理后的数据统计
	log.Println("\n当前数据库统计:")
	for _, t := range validTypes {
		var c int64
		db.DB.Table("audit_mains").Where("business_type = ?", t).Count(&c)
		typeName := map[int]string{1: "车票订单", 2: "酒店订单", 3: "酒店入驻"}[t]
		log.Printf("  - %s: %d 条\n", typeName, c)
	}

	_ = config.Cfg // 确保配置加载
}

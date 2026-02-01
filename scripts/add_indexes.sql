-- 审核系统性能优化 - 添加索引
-- 执行时间：2026-01-23

-- 1. 审核主表索引优化
ALTER TABLE `audit_mains` 
ADD INDEX IF NOT EXISTS `idx_created_at` (`created_at` DESC),
ADD INDEX IF NOT EXISTS `idx_deleted_at` (`deleted_at`);

-- 2. 车票订单表索引优化
ALTER TABLE `audit_ticket_orders`
ADD INDEX IF NOT EXISTS `idx_audit_main_id` (`audit_main_id`),
ADD INDEX IF NOT EXISTS `idx_deleted_at` (`deleted_at`);

-- 3. 酒店订单表索引优化
ALTER TABLE `audit_hotel_orders`
ADD INDEX IF NOT EXISTS `idx_audit_main_id` (`audit_main_id`),
ADD INDEX IF NOT EXISTS `idx_deleted_at` (`deleted_at`);

-- 查看索引创建结果
SHOW INDEX FROM `audit_mains`;
SHOW INDEX FROM `audit_ticket_orders`;
SHOW INDEX FROM `audit_hotel_orders`;

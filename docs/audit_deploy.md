# 审核模块部署指南 / Audit Module Deployment

## 中文
1. 配置文件
- 位置：`conf/config.yaml`
- 关键配置：
  - `Audit.Decision`（模型启用、阈值、特征字段）
  - `Audit.Model`（线性模型权重/偏置）
  - `Audit.Threshold`（动态阈值窗口与分位数）
  - `Audit.Metrics`（监控窗口）
  - `Audit.Alert`（告警阈值与 Webhook）

2. 启动服务
- 启动审核 RPC：`rpc/audit/main.go`
- 启动网关：`api/main.go`
- 监控接口：`/api/audit/metrics` 默认端口 `:8890`

3. 数据库
- 需提前建表：`audit_mains`、`audit_operation_logs`、`audit_ticket_orders`、`audit_hotel_orders`、`rule_configs`

---

## English
1. Config
- Path: `conf/config.yaml`
- Key settings:
  - `Audit.Decision`, `Audit.Model`, `Audit.Threshold`, `Audit.Metrics`, `Audit.Alert`

2. Start Services
- Audit RPC: `rpc/audit/main.go`
- API Gateway: `api/main.go`
- Metrics endpoint: `/api/audit/metrics` (default `:8890`)

3. Database
- Create tables: `audit_mains`, `audit_operation_logs`, `audit_ticket_orders`, `audit_hotel_orders`, `rule_configs`

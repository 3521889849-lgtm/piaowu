# 审核模块 API 文档 / Audit Module API

## 中文
### 1. 提交审核
- URL: `POST /api/audit/apply`
- 请求体（JSON）：
  - `biz_type`：业务类型（枚举）
  - `biz_id`：业务ID
  - `submitter_id`：提交人ID
  - `content`：审核内容（JSON字符串）
  - `attachments`：附件列表（可选）
  - `extra`：扩展字段（可选）
- 说明：`content` 会被解析为 Fact，参与规则引擎与模型判定
- 返回：`audit_id`

### 2. 获取人工审核任务
- URL: `GET /api/audit/manual/tasks`
- 参数：
  - `status`：审核状态（可选）
  - `biz_types`：业务类型列表（可选）
  - `auditor_id`：审核人ID（可选）
  - `pagination`：分页参数

### 3. 处理人工审核
- URL: `POST /api/audit/manual/process`
- 请求体：
  - `audit_id`：审核单ID
  - `is_passed`：是否通过
  - `auditor_id`：审核人ID
  - `remark`：审核备注

### 4. 查询审核详情
- URL: `GET /api/audit/record`
- 参数：
  - `audit_id` 或 `biz_id` + `biz_type`

### 5. 监控指标
- URL: `GET /api/audit/metrics`（由审核服务提供，端口默认 `:8890`）
- 返回字段：
  - `total_count`：总处理量
  - `avg_latency_ms`：平均延迟
  - `throughput_qps`：吞吐量
  - `accuracy`：准确率（基于人工审核对比）

---

## English
### 1. Submit Audit
- URL: `POST /api/audit/apply`
- Body (JSON):
  - `biz_type`, `biz_id`, `submitter_id`, `content`, `attachments`, `extra`
- Note: `content` is parsed into Fact for rule + model decision
- Response: `audit_id`

### 2. Fetch Manual Tasks
- URL: `GET /api/audit/manual/tasks`
- Params: `status`, `biz_types`, `auditor_id`, `pagination`

### 3. Process Manual Audit
- URL: `POST /api/audit/manual/process`
- Body: `audit_id`, `is_passed`, `auditor_id`, `remark`

### 4. Get Audit Record
- URL: `GET /api/audit/record`
- Params: `audit_id` or `biz_id` + `biz_type`

### 5. Metrics
- URL: `GET /api/audit/metrics` (audit service, default `:8890`)
- Fields: `total_count`, `avg_latency_ms`, `throughput_qps`, `accuracy`

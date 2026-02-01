# 审核模块异常处理手册 / Audit Exception Handbook

## 中文
### 1. 规则解析失败
- 现象：规则未生效
- 原因：`rule_configs.expression` JSON 不合法
- 处理：修正 JSON 后触发规则重载

### 2. 模型评分失败
- 现象：自动转人工
- 原因：特征缺失或模型不可用
- 处理：检查 `Audit.Model` 配置与特征字段

### 3. 动态阈值异常
- 现象：阈值过高/过低
- 原因：窗口过小或数据分布异常
- 处理：调整 `WindowSize` / `Percentile` / `Min` / `Max`

### 4. 告警频繁触发
- 现象：频繁收到告警
- 原因：阈值设置不合理或系统性能下降
- 处理：调高 `MaxLatencyMs` 或 `MinThroughput`，排查数据库/网络

---

## English
### 1. Rule Parse Failure
- Symptom: rules not applied
- Cause: invalid JSON in `rule_configs.expression`
- Fix: correct JSON and reload rules

### 2. Model Scoring Failure
- Symptom: default to manual review
- Cause: missing features or model disabled
- Fix: check `Audit.Model` config and feature keys

### 3. Dynamic Threshold Issue
- Symptom: threshold too high/low
- Cause: small window or abnormal distribution
- Fix: tune `WindowSize` / `Percentile` / `Min` / `Max`

### 4. Alerts Too Frequent
- Symptom: frequent alerts
- Cause: strict thresholds or performance regression
- Fix: adjust `MaxLatencyMs` / `MinThroughput`, check DB/network

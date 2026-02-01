## 目标
- 让你能一键启动（含 Jaeger）并用 traceId 在 UI 里看到完整链路。

## 我会补齐的内容
- 新增一份可直接复制的配置样例（conf/config.yaml.example），覆盖 MySQL/Redis/Gateway/各 RPC 服务最小必填项。
- 新增一份本地观测说明文档：如何启动 Jaeger、启动各服务、如何用响应头 X-Trace-Id 在 Jaeger UI 搜索、如何验证跨服务传播。
- （可选）提供 docker-compose.yml（jaeger + mysql + redis）与一个启动脚本（Windows PowerShell 版本），减少手动步骤。

## 验证方式
- 启动 gateway + user_service/ticket_service/order_service 后，访问任一 /api/v1 接口：
  - 响应头包含 X-Trace-Id
  - Jaeger UI 能按 Trace ID 搜索到 trace，且包含 gateway 的 HTTP span + 下游 kitex span
- 跑 go test ./... 确保改动不破坏现有构建。

## 风险与回滚
- 文档/样例/compose 都是新增文件，不影响线上；如不需要可直接删除。
- 运行时开关全部通过环境变量控制（JAEGER_*），不写死在代码里。
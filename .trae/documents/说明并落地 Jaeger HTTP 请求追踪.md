## 先解释“HTTP 请求追踪有什么用”
- 把一次用户请求在“网关 HTTP → 下游 RPC → DB/缓存”等所有步骤的耗时串成一条链路（Trace），快速定位慢在哪一段。
- 出错时能看到是哪一个调用失败（哪个 Span）、错误信息/重试、上下游依赖关系，比只看日志更容易还原现场。
- 和日志/指标互补：指标看整体趋势；日志看细节；Trace 用于追单次请求的端到端路径，并用 traceId 关联日志。

## 当前仓库的落地点（按你的代码思路，做成可复用“单独封装”）
- 仓库现状：目前没有 Jaeger/OpenTracing/OpenTelemetry 相关代码或依赖；HTTP 用 Hertz（全局 `h.Use(...)` 中间件），RPC 用 Kitex（client/server 创建点集中）。
- 最佳 Hook 点：
  - HTTP 入口（Gateway）：在 `internal/gateway/http/router/router.go` 里最前面加一个 Tracing 中间件，创建 root span，把带 span 的 `context.Context` 传下去（你项目的 handler 已经把 `ctx` 透传给 Kitex client，链路打通的条件很好）。
  - RPC 出口（Gateway → 各微服务）：在 `internal/gateway/rpc/*_client.go` 创建 client 时加 tracer option，让 Kitex 自动从 `ctx` 注入传播信息。
  - RPC 入口（各微服务）：在 `cmd/user_service/main.go`、`cmd/ticket_service/main.go`、`cmd/order_service/main.go` 创建 server 时加 tracer option，提取上下文并创建 server span。

## 代码组织（你说的“单独写出来”）
- 新增一个统一 tracing 包（不散落在业务里）：
  - `common/observability/tracing/jaeger.go`：封装 `InitTracer(serviceName)`、采样、Collector Endpoint、关闭 flush。
  - `common/observability/tracing/propagation.go`：封装 HTTP header 的 Extract/Inject 载体（适配 Hertz header 结构）。
- Gateway 专用中间件：
  - `internal/gateway/http/middleware/tracing.go`：实现 `Tracing()`，对除 `/health` 外的请求创建 span（span 名用 `METHOD route`），把 `trace_id/span_id` 写入响应头便于排查。
- Kitex 接入层：
  - `internal/gateway/rpc/tracing_options.go`：提供 `ClientTracingOptions()` 之类的函数，让三个 client 创建处统一复用。
  - `internal/*_service`（或 `cmd/*/main.go` 旁）增加 `ServerTracingOptions()` 复用。

## 依赖与配置（让它可在不同环境跑）
- 依赖方案 A（贴近你给的示例）：使用 `opentracing-go` + `jaeger-client-go`。
- 配置用环境变量/配置文件，不硬编码：
  - `JAEGER_ENDPOINT`（如 `http://jaeger-collector:14268/api/traces`）
  - `JAEGER_SAMPLER`/`JAEGER_SAMPLER_PARAM`（开发 1.0，全量；生产用概率或限流）
  - `SERVICE_NAME`（每个服务不同：gateway/user_service/ticket_service/order_service）
- 修正你示例里的 CollectorEndpoint 字符串：去掉多余反引号和空格，避免初始化失败；同时避免把公网 IP 写死到仓库。

## 验证方式（接入后你能立刻看到效果）
- 启动 Jaeger（本地 all-in-one 或测试环境 collector）。
- 打一条链路：访问 Gateway 的某个 `/api/v1/*` 接口，确认 Jaeger UI 里出现同一个 trace，包含：HTTP root span + 下游 RPC spans。
- 用返回头里的 `trace_id` 在日志中检索（如果你希望我同步把 traceId 注入日志字段，也会在接入时一起做）。

## 我接下来会实际改哪些地方（确认后执行）
- 增加 tracing 封装包与 Hertz Tracing 中间件。
- 在 Gateway 的全局中间件注册处接入 Tracing。
- 在 Kitex client/server 创建处接入 tracer option，保证跨服务上下文传播。
- 本地跑一次编译与简单请求验证，确认 Jaeger UI 能看到完整链路。
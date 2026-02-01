# 在本项目查看远程 Jaeger

## 1. 先确认远程 Jaeger 地址

需要两类地址：

- Jaeger UI（查询与展示）：`http://<jaeger-host>:16686`
- Jaeger Collector（接收 trace，上报用，HTTP 推荐）：`http://<jaeger-host>:14268/api/traces`

如果你的环境只开放 Agent（UDP），则需要：

- Jaeger Agent：`<jaeger-host>:6831`（UDP）

## 2. 打开 Tracing 配置

在 `conf/config.yaml` 增加/修改：

```yaml
Tracing:
  Enabled: true
  Jaeger:
    CollectorEndpoint: "http://<jaeger-host>:14268/api/traces"
    AgentHost: ""
    AgentPort: 6831
    SamplerType: "const"
    SamplerParam: 1
    LogSpans: false
```

推荐优先填 `CollectorEndpoint`（HTTP），跨机房/跨网络比 UDP 更稳定。

## 3. 启动服务并发起一次请求

至少启动：

- `cmd/user_service`
- `cmd/ticket_service`
- `cmd/order_service`
- `cmd/gateway`

然后请求网关任意接口（例如登录/查询/下单）。只要请求能走通，Jaeger 里就会出现一条从 `api_gateway` 到各后端服务的调用链路。

## 4. 在 Jaeger UI 搜索链路

打开 `http://<jaeger-host>:16686`：

- Service 下拉选择：`api_gateway`（或 `user_service` / `ticket_service` / `order_service`）
- Operation 选择某个接口（例如 `POST /api/v1/user/login`）
- 点击 Find Traces

## 5. 常见排障

- UI 能打开但没数据：优先检查 Collector 地址是否能从服务机器访问（尤其是跨网段）
- 只有网关有 span：确认后端服务也打开了 `Tracing.Enabled=true`，并且都重启生效
- Operation 名称不对/都是 unknown：确认网关的 HTTP 请求确实经过 Hertz middleware（本项目已在路由注册处挂载）
- 远程 Jaeger 有鉴权：可使用 Jaeger client 的环境变量方式注入（例如 `JAEGER_USER` / `JAEGER_PASSWORD`），或在 Collector 前加反向代理做鉴权


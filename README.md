# 🎫 天极票务 - 智能审核系统

<div align="center">

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)
![React](https://img.shields.io/badge/React-18+-61DAFB?logo=react)
![Status](https://img.shields.io/badge/status-active-success.svg)

**一个基于规则引擎和机器学习的智能票务审核平台**

[功能特性](#-功能特性) • [快速开始](#-快速开始) • [技术架构](#-技术架构) • [API文档](#-api-文档) • [开发指南](#-开发指南)

</div>

---

## 📖 项目简介

天极票务是一个企业级的智能审核系统，支持**车票、酒店、机票、旅游门票**等多种业务类型的自动化和人工审核流程。系统采用**自动审核引擎 + 人工审核**的混合模式，通过规则引擎、机器学习模型和动态阈值管理，实现高效、准确的审核决策。

### 核心优势

- 🤖 **智能决策** - 规则引擎 + ML模型自动审核，人工审核准确率 > 95%
- ⚡ **高性能** - Kitex RPC框架，P99延迟 < 100ms
- 🔌 **易扩展** - 插件化架构，支持自定义规则和业务类型
- 📊 **可观测** - 内置监控告警，实时追踪审核指标
- 🎨 **现代化UI** - React + Ant Design，支持暗黑模式

---

## ✨ 功能特性

### 业务支持

| 业务类型 | 审核内容 | 自动审核率 |
|---------|---------|-----------|
| 🎫 **车票订单** | 高铁/动车/普通火车/汽车票退改签审核 | ~85% |
| 🏨 **酒店订单** | 酒店预订退款申请审核 | ~80% |
| 🏢 **酒店入驻** | 新商户资质审核（营业执照等） | ~60% |
| ✈️ **机票订单** | 国内/国际航班退改签审核 | ~75% |
| 🎡 **旅游门票** | 景区/演出/展览门票退款审核 | ~82% |

### 核心功能

#### 🤖 自动审核引擎
- **规则引擎** - 基于业务规则自动判决（金额阈值、时间窗口等）
- **ML模型** - 线性模型评分，支持离线训练、在线预测
- **动态阈值** - 根据历史数据自动调整决策阈值
- **黑名单/敏感词** - 插件化扩展，支持自定义检测逻辑

#### 👨‍💼 人工审核
- **任务队列** - 待审核任务自动分配
- **详情查看** - 完整业务信息展示（订单详情、用户历史等）
- **批注审核** - 支持通过/驳回，附带审核意见
- **操作日志** - 完整记录审核流转轨迹

#### 📈 监控告警
- **性能监控** - QPS、延迟、错误率实时追踪
- **准确率统计** - 自动审核 vs 人工审核对比分析
- **业务大盘** - 各业务类型审核分布、趋势分析
- **异常告警** - 延迟过高、准确率下降自动告警

---

## 🏗️ 技术架构

### 系统架构图

```
┌─────────────────┐
│   Web Frontend  │  React + TypeScript + Ant Design
│   (Port: 3000)  │
└────────┬────────┘
         │ HTTP/JSON
         ↓
┌─────────────────┐
│  API Gateway    │  Gin Framework (Go)
│   (Port: 8080)  │  - 协议转换 (HTTP → RPC)
└────────┬────────┘  - CORS处理
         │ Kitex RPC  - 鉴权中间件
         ↓
┌─────────────────┐
│  Audit Service  │  Cloudwego Kitex
│   (Port: 8889)  │  - 自动审核引擎
└────────┬────────┘  - 人工审核服务
         │            - 规则引擎 + ML模型
         ↓
┌─────────────────┬─────────────────┐
│  MySQL/GORM     │  Redis Cache    │
│  - 审核记录     │  - 规则缓存     │
│  - 业务详情     │  - 阈值管理     │
└─────────────────┴─────────────────┘
```

### 技术栈

**后端**
- **框架**: [Cloudwego Kitex](https://www.cloudwego.io/zh/docs/kitex/) (RPC) + [Gin](https://gin-gonic.com/) (HTTP Gateway)
- **语言**: Go 1.25+
- **数据库**: MySQL 8.0+ (GORM)
- **缓存**: Redis 7.0+
- **IDL**: Apache Thrift

**前端**
- **框架**: React 18 + TypeScript
- **UI库**: Ant Design 5.x
- **状态管理**: React Hooks
- **HTTP客户端**: Axios
- **构建工具**: Vite

**基础设施**
- **容器化**: Docker + Docker Compose
- **监控**: Prometheus + Grafana (可选)

---

## 🚀 快速开始

### 环境要求

- Go 1.25 或更高版本
- Node.js 18+ 和 npm/yarn
- MySQL 8.0+
- Redis 7.0+

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/piaowu.git
cd piaowu
```

### 2. 配置数据库

```bash
# 启动MySQL和Redis (使用Docker Compose)
docker-compose up -d mysql redis

# 或手动配置，修改配置文件
cp conf/config.example.yaml conf/config.yaml
# 编辑 config.yaml，填写数据库连接信息
```

**初始化数据库**
```bash
# 运行数据库迁移脚本
go run scripts/db_seed.go
```

### 3. 启动后端服务

#### 方式一：直接运行
```bash
# 安装依赖
go mod download

# 启动 Audit RPC 服务 (Port: 8889)
cd rpc/audit
go run .

# 新终端，启动 API Gateway (Port: 8080)
cd api  # 或 gateway
go run .
```

#### 方式二：使用脚本
```bash
# 构建并启动所有服务
./script/bootstrap.sh
```

### 4. 启动前端

```bash
cd frontend

# 安装依赖
npm install
# 或 yarn install

# 启动开发服务器 (Port: 3000)
npm run dev
```

### 5. 访问应用

打开浏览器访问：
- **前端应用**: http://localhost:3000
- **API网关**: http://localhost:8080/api/audit

**默认测试用户**: 系统目前无需登录，可直接使用

---

## 📁 项目结构

```
piaowu/
├── api/                    # API Gateway HTTP服务
│   └── main.go
├── gateway/                # 备用Gateway实现
│   └── main.go
├── rpc/                    # RPC服务
│   └── audit/              # 审核服务
│       ├── handler.go      # RPC接口实现
│       ├── service/        # 业务逻辑
│       │   ├── auto_audit.go       # 自动审核引擎
│       │   └── manual_audit.go     # 人工审核逻辑
│       └── component/              # 核心组件
│           ├── rule_engine/        # 规则引擎
│           ├── decision/           # 决策引擎
│           ├── ml/                 # 机器学习模型
│           ├── threshold/          # 动态阈值管理
│           ├── metrics/            # 监控指标
│           └── alert/              # 告警管理
├── common/                 # 公共代码
│   ├── model/              # 数据模型
│   │   └── audit/          # 审核相关模型
│   │       ├── audit_mains.go
│   │       ├── audit_flight_orders.go
│   │       ├── audit_scenic_orders.go
│   │       └── ...
│   ├── db/                 # 数据库连接
│   │   ├── mysql.go
│   │   └── redis.go
│   └── config/             # 配置管理
│       └── config.go
├── idl/                    # Thrift IDL定义
│   └── audit.thrift        # 审核服务接口定义
├── kitex_gen/              # Kitex自动生成代码
│   └── audit/
├── frontend/               # React前端
│   ├── src/
│   │   ├── pages/          # 页面组件
│   │   │   ├── ApplyAudit.tsx      # 提交审核
│   │   │   ├── AuditList.tsx       # 审核列表
│   │   │   └── AuditDashboard.tsx  # 审核大盘
│   │   ├── api/            # API客户端
│   │   │   └── index.ts
│   │   └── App.tsx         # 应用入口
│   └── package.json
├── scripts/                # 工具脚本
│   └── db_seed.go          # 数据库种子数据
├── conf/                   # 配置文件
│   └── config.yaml
├── go.mod
└── README.md
```

---

## 📡 API 文档

### 基础信息

- **Base URL**: `http://localhost:8080/api/audit`
- **Content-Type**: `application/json`

### 接口列表

#### 1. 提交审核

**请求**
```http
POST /submit
Content-Type: application/json

{
  "business_type": 4,              // 业务类型（1=车票, 2=酒店订单, 3=酒店入驻, 4=机票, 5=门票）
  "business_id": 40001,            // 业务主键ID
  "apply_reason": "机票退款申请",  // 申请原因
  
  "flight_order": {                // 机票订单详情（根据业务类型提供）
    "flight_order_id": 40001,
    "flight_type": 1,              // 1=国内, 2=国际
    "flight_no": "CA1234",
    "airline": "中国国际航空",
    "departure_airport": "北京首都国际机场",
    "arrival_airport": "上海浦东国际机场",
    "departure_time": "2026-02-05T08:30:00Z",
    "arrival_time": "2026-02-05T10:45:00Z",
    "cabin_class": "经济舱",
    "passenger_name": "张三",
    "passenger_id_card": "110101199001011234",
    "order_amount": 1280.0
  }
}
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "audit_id": 123,
    "business_type": 4,
    "business_id": 40001,
    "audit_status": 1,           // 1=待审核, 2=审核中, 3=通过, 4=驳回, 5=撤销
    "submit_time": "2026-02-02T10:30:00Z"
  }
}
```

#### 2. 查询审核列表

**请求**
```http
GET /list?page=1&page_size=10&business_type=4&audit_status=1
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 15,
    "list": [
      {
        "audit_id": 123,
        "business_type": 4,
        "business_id": 40001,
        "audit_status": 1,
        "audit_status_text": "待审核",
        "submit_user_name": "User_1",
        "submit_time": "2026-02-02T10:30:00Z"
      }
    ]
  }
}
```

#### 3. 查询审核详情

**请求**
```http
GET /record?audit_id=123
```

**响应**
```json
{
  "base_resp": { "code": 0, "msg": "Success" },
  "detail": {
    "task_info": { /* 基本信息 */ },
    "logs": [ /* 操作日志 */ ],
    "flight_order": { /* 机票详细信息 */ }
  }
}
```

#### 4. 处理审核

**请求**
```http
POST /process
Content-Type: application/json

{
  "audit_id": 123,
  "action": "approve",           // "approve" 或 "reject"
  "audit_remark": "审核通过，信息核实无误"
}
```

**响应**
```json
{
  "code": 0,
  "message": "success"
}
```

完整API文档请查看：[API详细文档](./docs/api.md)

---

## 🛠️ 开发指南

### 添加新的业务类型

**步骤 1**: 修改 IDL 定义

编辑 `idl/audit.thrift`：
```thrift
enum BizType {
    // ... 现有类型 ...
    NEW_TYPE = 6     // 新业务类型
}

// 定义新的详情结构
struct AuditNewTypeOrder {
    1: i64 order_id
    2: string field1
    // ... 其他字段
}
```

**步骤 2**: 重新生成代码

```bash
kitex -module piaowu idl/audit.thrift
```

**步骤 3**: 创建数据模型

在 `common/model/audit/` 创建 `audit_new_type_orders.go`

**步骤 4**: 实现业务逻辑

在 `rpc/audit/service/auto_audit.go` 的 `createSubRecord` 添加 case 分支

**步骤 5**: 更新前端

在 `frontend/src/api/index.ts` 添加类型定义和映射

### 自定义审核规则

编辑规则配置（JSON格式）或实现 `Plugin` 接口：

```go
type CustomPlugin struct {}

func (p *CustomPlugin) Name() string {
    return "CustomRule"
}

func (p *CustomPlugin) Execute(ctx context.Context, fact Fact) (*PluginResult, error) {
    // 自定义规则逻辑
    return &PluginResult{
        Action: "REVIEW",
        Reason: "触发自定义规则",
    }, nil
}
```

### 运行测试

```bash
# 单元测试
go test ./rpc/audit/service/... -v

# 集成测试
go test ./test/integration/... -v

# 前端测试
cd frontend
npm run test
```

---

## 🚢 部署说明

### Docker 部署

```bash
# 构建镜像
docker-compose build

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 生产环境配置

1. **修改配置文件** `conf/config.yaml`
   ```yaml
   server:
     port: 8889
   database:
     host: your-mysql-host
     port: 3306
     ...
   redis:
     addr: your-redis-host:6379
   ```

2. **编译生产版本**
   ```bash
   go build -o bin/audit_service rpc/audit/*.go
   go build -o bin/gateway api/main.go
   ```

3. **使用进程管理器**（如 systemd、supervisor）

---

## 📊 监控指标

访问 `http://localhost:8080/api/audit/metrics` 查看实时指标：

```json
{
  "qps": 150.5,
  "avg_latency_ms": 45.2,
  "p99_latency_ms": 98.5,
  "auto_audit_rate": 82.3,
  "accuracy": 96.8
}
```


/*
 * 配置结构体定义模块
 * 
 * 功能说明：
 * - 定义所有配置项的结构体
 * - 全局配置对象Cfg，供其他模块使用
 * - 与conf/config.yaml文件结构对应
 */
package config

// Cfg 全局配置对象
// 在ViperInit()函数中被填充，整个应用生命周期内使用
var Cfg = new(Config)

// Config 应用配置根结构体
// 包含所有子配置模块
type Config struct {
	Mysql  Mysql  // MySQL数据库配置
	Redis  Redis  // Redis缓存配置
	Server Server // 服务地址配置（Gateway、UserService等）
	Coupon Coupon // 优惠券相关配置（预留）
	JWT    JWT    // JWT Token配置
	RealName RealName // 实名认证配置
	AliPay AliPay // 支付宝支付配置
	Tracing Tracing // 链路追踪配置（Jaeger）
	Assistant Assistant // 智能助手配置（LLM/RAG/NLU）
}

// Mysql MySQL数据库配置结构体
type Mysql struct {
	Host     string // 数据库主机地址
	Port     int    // 数据库端口（默认3306）
	User     string // 数据库用户名
	Password string // 数据库密码
	Database string // 数据库名称
	LogLevel string // gorm日志级别：silent/error/warn/info
	AutoMigrate bool // 是否在启动时执行AutoMigrate
	StartupFixes bool // 是否在启动时执行一次性修复SQL
	ReadReplica MysqlReplica // 只读副本（读写分离：查询走只读库）
}

type MysqlReplica struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Enabled  bool
}

// Redis Redis缓存配置结构体
type Redis struct {
	Host     string // Redis主机地址
	Port     int    // Redis端口（默认6379）
	Password string // Redis密码（如果未设置则为空）
	Database int    // Redis数据库编号（0-15）
}

// Server 服务地址配置结构体
// 定义各个微服务的监听地址
type Server struct {
	Gateway     ServiceAddr // Gateway服务地址
	UserService ServiceAddr // User Service服务地址
	TicketService ServiceAddr
	OrderService  ServiceAddr
}

// ServiceAddr 服务地址结构体
// 用于定义服务的Host和Port
type ServiceAddr struct {
	Host string // 服务监听的主机（0.0.0.0表示所有网卡，127.0.0.1表示仅本地）
	Port int    // 服务监听的端口
}

// Coupon 优惠券配置结构体（预留）
// 用于防刷、加密等业务配置
type Coupon struct {
	AntiBrushLimit  int    // 防刷限制：单用户/设备时间窗口内最多领取次数
	AntiBrushExpire int    // 防刷过期时间（秒）
	AesKey          string // AES加密密钥（16/24/32字节）
}

// JWT JWT Token配置结构体
// 用于JWT Token的生成和验证
type JWT struct {
	Secret string // JWT签名密钥（建议使用32位以上的随机字符串）
	Expire int    // Token过期时间（秒），默认7200（2小时）
}

type RealName struct {
	SecretID  string // 云市场分配的密钥ID
	SecretKey string // 云市场分配的密钥Key
	DebugReturn bool // 调试：响应中返回第三方接口关键信息
}

type AliPay struct {
	AppId        string
	PrivateKey   string
	AliPublicKey string
	NotifyURL    string
	ReturnURL    string
	IsProduction bool
}

type Tracing struct {
	Enabled bool
	Jaeger  Jaeger
}

type Jaeger struct {
	CollectorEndpoint string
	AgentHost         string
	AgentPort         int
	SamplerType       string
	SamplerParam      float64
	LogSpans          bool
}

type Assistant struct {
	LLM LLM
	RAG RAG
}

type LLM struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	TimeoutSeconds int
}

type RAG struct {
	Enabled bool
	TopK    int
}

/*
 * RPC客户端管理模块
 * 
 * 功能说明：
 * - 统一管理Gateway对后端微服务的RPC客户端创建
 * - 封装客户端初始化逻辑，便于统一配置和管理
 * 
 * 扩展性：
 * - 后续可添加服务发现（Consul/Nacos）
 * - 可添加负载均衡、熔断等配置
 */
package rpc

import (
	"example_shop/common/config"
	"example_shop/kitex_gen/userapi/userservice"

	kclient "github.com/cloudwego/kitex/client"
	ktracer "github.com/kitex-contrib/tracer-opentracing"
)

// NewUserClient 创建User Service的RPC客户端
// 
// 参数：
//   - addr: User Service的服务地址，格式为 "host:port"
// 
// 返回值：
//   - userservice.Client: User Service的RPC客户端接口
//   - error: 如果创建失败，返回错误信息
// 
// 使用示例：
//   client, err := NewUserClient("127.0.0.1:8888")
//   if err != nil {
//       log.Fatal(err)
//   }
//   resp, err := client.Login(ctx, req)
func NewUserClient(addr string) (userservice.Client, error) {
	// 创建Kitex RPC客户端
	// 参数说明：
	//   - "user_service": 服务名称（用于服务发现和监控）
	//   - WithHostPorts: 指定服务地址（直连模式）
	// 
	// TODO: 后续可扩展为服务发现模式
	//   - WithResolver: 使用服务发现（Consul/Nacos）
	//   - WithLoadBalancer: 负载均衡策略
	//   - WithCircuitBreaker: 熔断器配置
	opts := []kclient.Option{kclient.WithHostPorts(addr)}
	if config.Cfg != nil && config.Cfg.Tracing.Enabled {
		opts = append(opts, kclient.WithSuite(ktracer.NewDefaultClientSuite()))
	}
	return userservice.NewClient("user_service", opts...)
}

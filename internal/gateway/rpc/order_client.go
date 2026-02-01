package rpc

import (
	"example_shop/common/config"
	"example_shop/kitex_gen/orderapi/orderservice"

	kclient "github.com/cloudwego/kitex/client"
	ktracer "github.com/kitex-contrib/tracer-opentracing"
)

func NewOrderClient(addr string) (orderservice.Client, error) {
	opts := []kclient.Option{kclient.WithHostPorts(addr)}
	if config.Cfg != nil && config.Cfg.Tracing.Enabled {
		opts = append(opts, kclient.WithSuite(ktracer.NewDefaultClientSuite()))
	}
	return orderservice.NewClient("order_service", opts...)
}

package rpc

import (
	"example_shop/common/config"
	"example_shop/kitex_gen/ticketapi/ticketservice"

	kclient "github.com/cloudwego/kitex/client"
	ktracer "github.com/kitex-contrib/tracer-opentracing"
)

func NewTicketClient(addr string) (ticketservice.Client, error) {
	opts := []kclient.Option{kclient.WithHostPorts(addr)}
	if config.Cfg != nil && config.Cfg.Tracing.Enabled {
		opts = append(opts, kclient.WithSuite(ktracer.NewDefaultClientSuite()))
	}
	return ticketservice.NewClient("ticket_service", opts...)
}

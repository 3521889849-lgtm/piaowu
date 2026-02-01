package main

import (
	"example_shop/common/config"
	initpkg "example_shop/common/init"
	"example_shop/internal/ticket_service/job"
	"example_shop/internal/user_service/handler/orderapi"
	"example_shop/kitex_gen/orderapi/orderservice"
	"fmt"
	"log"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/utils"
	"github.com/cloudwego/kitex/server"
	ktracer "github.com/kitex-contrib/tracer-opentracing"
)

func main() {
	shutdown := initpkg.Init("order_service")
	defer shutdown()

	stop := make(chan struct{})
	job.StartOrderCleanup(stop)
	defer close(stop)

	listenAddr := fmt.Sprintf("%s:%d", config.Cfg.Server.OrderService.Host, config.Cfg.Server.OrderService.Port)

	svr := orderservice.NewServer(
		orderapi.NewOrderServiceImpl(),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "order_service"}),
		server.WithServiceAddr(utils.NewNetAddr("tcp", listenAddr)),
		server.WithSuite(ktracer.NewDefaultServerSuite()),
	)

	log.Println("order_service started")
	if err := svr.Run(); err != nil {
		log.Println("order_service stopped:", err)
	}
}

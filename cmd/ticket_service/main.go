package main

import (
	"example_shop/common/config"
	initpkg "example_shop/common/init"
	"example_shop/internal/ticket_service/handler/ticketapi"
	"example_shop/kitex_gen/ticketapi/ticketservice"
	"fmt"
	"log"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/utils"
	"github.com/cloudwego/kitex/server"
	ktracer "github.com/kitex-contrib/tracer-opentracing"
)

func main() {
	shutdown := initpkg.Init("ticket_service")
	defer shutdown()

	listenAddr := fmt.Sprintf("%s:%d", config.Cfg.Server.TicketService.Host, config.Cfg.Server.TicketService.Port)

	svr := ticketservice.NewServer(
		ticketapi.NewTicketServiceImpl(),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "ticket_service"}),
		server.WithServiceAddr(utils.NewNetAddr("tcp", listenAddr)),
		server.WithSuite(ktracer.NewDefaultServerSuite()),
	)

	log.Println("ticket_service started")
	if err := svr.Run(); err != nil {
		log.Println("ticket_service stopped:", err)
	}
}

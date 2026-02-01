package handler

import (
	assistantHandler "example_shop/internal/gateway/http/handler/assistant"
	"example_shop/internal/gateway/http/handler/order"
	"example_shop/internal/gateway/http/handler/ticket"
	"example_shop/internal/gateway/http/handler/user"
	"example_shop/kitex_gen/orderapi/orderservice"
	"example_shop/kitex_gen/ticketapi/ticketservice"
	"example_shop/kitex_gen/userapi/userservice"
)

// App 聚合 Gateway 所有领域 HTTP Handler，并统一注入依赖（例如 RPC client）。
type App struct {
	User      *user.Handler
	Ticket    *ticket.Handler
	Order     *order.Handler
	Assistant *assistantHandler.Handler
}

// NewApp 构造一个网关 App 实例。
func NewApp(userClient userservice.Client, ticketClient ticketservice.Client, orderClient orderservice.Client) *App {
	return &App{
		User:      &user.Handler{UserClient: userClient},
		Ticket:    &ticket.Handler{TicketClient: ticketClient},
		Order:     &order.Handler{OrderClient: orderClient},
		Assistant: assistantHandler.New(userClient, ticketClient, orderClient),
	}
}

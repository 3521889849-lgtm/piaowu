/*
 * Gateway路由注册模块
 *
 * 功能说明：
 * - 定义所有HTTP API路由规则
 * - 将HTTP请求映射到对应的处理函数
 * - 统一管理API版本和路径前缀
 *
 * 路由规则：
 * - 所有API统一前缀：/api/v1
 * - RESTful风格：POST用于创建/操作，GET用于查询
 */
package router

import (
	"context"
	"example_shop/internal/gateway/http/handler"
	"example_shop/internal/gateway/http/middleware"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// RegisterRoutes 注册所有API路由
//
// 参数说明：
//   - h: Hertz服务器实例
//   - app: 包含所有RPC客户端的应用实例
//
// 路由分组策略：
//   - /api: 所有API的统一前缀
//   - /v1: API版本号，便于后续版本升级
//   - /user: 用户相关接口组
func RegisterRoutes(h *server.Hertz, app *handler.App) {
	h.Use(middleware.Tracing())
	h.Use(middleware.RateLimit(10))

	h.GET("/", serveWeb("index.html", "text/html; charset=utf-8"))
	h.GET("/styles.css", serveWeb("styles.css", "text/css; charset=utf-8"))
	h.GET("/app.js", serveWeb("app.js", "application/javascript; charset=utf-8"))

	// 创建路由分组：所有API都使用/api前缀
	api := h.Group("/api")
	// 创建版本分组：v1版本的所有接口
	v1 := api.Group("/v1")

	// 用户相关路由
	// POST /api/v1/user/register - 用户注册
	v1.POST("/user/register", app.User.Register)
	// POST /api/v1/user/login - 用户登录
	v1.POST("/user/login", app.User.Login)

	authed := v1.Group("", middleware.AuthRequired())

	// POST /api/v1/user/verify_realname - 实名认证
	authed.POST("/user/verify_realname", app.User.VerifyRealName)
	// GET /api/v1/user/info - 获取用户信息
	authed.GET("/user/info", app.User.GetUserInfo)
	// GET /api/v1/user/profile - 获取个人信息（当前登录用户）
	authed.GET("/user/profile", app.User.GetProfile)
	// POST /api/v1/user/profile - 更新个人信息（简化：仅支持更新real_name）
	authed.POST("/user/profile", app.User.UpdateProfile)
	// GET /api/v1/user/passengers - 常用乘车人（从历史订单提取）
	authed.GET("/user/passengers", app.User.ListPassengers)
	// GET /api/v1/station/suggest - 站点联想（票务查询）
	v1.GET("/station/suggest", app.Ticket.StationSuggest)
	// GET /api/v1/train/search - 车次查询（票务查询）
	v1.GET("/train/search", app.Ticket.SearchTrain)
	// GET /api/v1/train/detail - 车次详情（余票与最低价）
	v1.GET("/train/detail", app.Ticket.TrainDetail)
	// GET /api/v1/ticket/ws - WebSocket余票推送（票务查询）
	v1.GET("/ticket/ws", app.Ticket.TicketRemainWS)

	// 订单相关路由（携程买票流程：下单->支付->出票/取消）
	// POST /api/v1/order/create - 创建订单并锁座
	authed.POST("/order/create", app.Order.CreateOrder)
	// POST /api/v1/order/pay - 支付订单并出票（简化）
	authed.POST("/order/pay", app.Order.PayOrder)
	// POST /api/v1/pay/callback - 模拟第三方支付异步回调
	v1.POST("/pay/callback", app.Order.PayCallback)
	// POST /api/v1/pay/mock_notify - 触发一条本地模拟回调（便于 ApiPost 测试）
	authed.POST("/pay/mock_notify", app.Order.MockPayNotify)
	// POST /api/v1/order/cancel - 取消订单并释放锁座
	authed.POST("/order/cancel", app.Order.CancelOrder)
	// POST /api/v1/order/refund - 退票并释放已售区间占用
	authed.POST("/order/refund", app.Order.RefundOrder)
	// POST /api/v1/order/change - 改签（更换车次/区间）
	authed.POST("/order/change", app.Order.ChangeOrder)
	// GET /api/v1/order/info - 查询订单详情
	authed.GET("/order/info", app.Order.GetOrder)
	// GET /api/v1/order/list - 查询订单列表
	authed.GET("/order/list", app.Order.ListOrders)

	// 智能助手与知识库路由
	// POST /api/v1/assistant/chat - 智能助手对话（支持规划与工具调用）
	authed.POST("/assistant/chat", app.Assistant.Chat)
	// POST /api/v1/assistant/kb/upsert - 知识库文档写入/更新（用于 RAG 召回）
	authed.POST("/assistant/kb/upsert", app.Assistant.KBUpsert)
	// POST /api/v1/assistant/kb/search - 知识库文档检索（测试召回效果）
	authed.POST("/assistant/kb/search", app.Assistant.KBSearch)
}

var (
	webCacheMu  sync.RWMutex
	webCache    = map[string][]byte{}
	webCacheErr = map[string]error{}
)

func serveWeb(name, contentType string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		data, err := loadWebAsset(name)
		if err != nil {
			if strings.HasSuffix(name, ".html") {
				c.String(404, "web资源不存在："+name)
				return
			}
			c.Status(404)
			return
		}
		c.Header("Content-Type", contentType)
		c.Write(data)
	}
}

func loadWebAsset(name string) ([]byte, error) {
	webCacheMu.RLock()
	if data, ok := webCache[name]; ok {
		webCacheMu.RUnlock()
		return data, nil
	}
	if err, ok := webCacheErr[name]; ok {
		webCacheMu.RUnlock()
		return nil, err
	}
	webCacheMu.RUnlock()

	var (
		data []byte
		err  error
	)
	for _, base := range []string{"web", filepath.Join("..", "web"), filepath.Join("..", "..", "web"), filepath.Join("..", "..", "..", "web")} {
		path := filepath.Join(base, name)
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	webCacheMu.Lock()
	if err != nil {
		webCacheErr[name] = err
	} else {
		webCache[name] = data
	}
	webCacheMu.Unlock()
	return data, err
}

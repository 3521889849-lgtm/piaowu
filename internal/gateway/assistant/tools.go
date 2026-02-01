/*
 * 智能助手工具箱
 *
 * 功能说明：
 * - 定义可供 LLM 调用的本地函数（Tools）
 * - 统一工具接口：Name, RequiredSlots, Execute
 * - 注册与管理所有可用工具
 */
package assistant

import (
	"context"
	"errors"
	"example_shop/internal/gateway/http/dto"
	ticketlogic "example_shop/internal/gateway/ticket"
	kitexorder "example_shop/kitex_gen/orderapi"
	"example_shop/kitex_gen/orderapi/orderservice"
	kitexticket "example_shop/kitex_gen/ticketapi"
	"example_shop/kitex_gen/ticketapi/ticketservice"
	"strings"
	"time"
)

// Tool 工具接口
type Tool interface {
	Name() string                               // 工具名称（对应 NLU 意图）
	RequiredSlots() []string                    // 必填槽位（参数）
	Execute(ctx context.Context, in ToolInput) (any, error) // 执行逻辑
}

// ToolInput 工具入参
type ToolInput struct {
	UserID string            // 当前用户ID
	Slots  map[string]string // 提取到的槽位参数
}

// SearchTrainTool 查票工具
// 必填：departure_station, arrival_station
// 可选：travel_date (默认今天), sort, direction
type SearchTrainTool struct{}

func (t *SearchTrainTool) Name() string { return "SearchTrain" }
func (t *SearchTrainTool) RequiredSlots() []string {
	return []string{"departure_station", "arrival_station"}
}
func (t *SearchTrainTool) Execute(ctx context.Context, in ToolInput) (any, error) {
	req := dto.SearchTrainHTTPReq{
		DepartureStation: strings.TrimSpace(in.Slots["departure_station"]),
		ArrivalStation:   strings.TrimSpace(in.Slots["arrival_station"]),
		TravelDate:       strings.TrimSpace(in.Slots["travel_date"]),
		Sort:             strings.TrimSpace(in.Slots["sort"]),
		Direction:        strings.TrimSpace(in.Slots["direction"]),
	}
	// 默认查询今天的车票
	if req.TravelDate == "" {
		req.TravelDate = time.Now().Format("2006-01-02")
	}
	// 复用 Ticket 业务逻辑
	res := ticketlogic.New().SearchTrain(ctx, req)
	return res.Body, nil
}

// TrainDetailTool 车次详情工具
// 必填：train_id
type TrainDetailTool struct {
	TicketClient ticketservice.Client
}

func (t *TrainDetailTool) Name() string { return "TrainDetail" }
func (t *TrainDetailTool) RequiredSlots() []string { return []string{"train_id"} }
func (t *TrainDetailTool) Execute(ctx context.Context, in ToolInput) (any, error) {
	trainID := strings.TrimSpace(in.Slots["train_id"])
	if trainID == "" {
		return nil, errors.New("train_id为空")
	}
	return t.TicketClient.GetTrainDetail(ctx, &kitexticket.GetTrainDetailReq{TrainId: trainID})
}

// ListOrdersTool 查订单工具
// 依赖：UserID (从 ToolInput 获取)
type ListOrdersTool struct {
	OrderClient orderservice.Client
}

func (t *ListOrdersTool) Name() string { return "ListOrders" }
func (t *ListOrdersTool) RequiredSlots() []string { return nil }
func (t *ListOrdersTool) Execute(ctx context.Context, in ToolInput) (any, error) {
	if strings.TrimSpace(in.UserID) == "" {
		return nil, errors.New("user_id为空")
	}
	return t.OrderClient.ListOrders(ctx, &kitexorder.ListOrdersReq{UserId: in.UserID, Limit: 10})
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry 初始化并注册内置工具
func NewToolRegistry(ticketClient ticketservice.Client, orderClient orderservice.Client) *ToolRegistry {
	reg := &ToolRegistry{tools: map[string]Tool{}}
	reg.Register(&SearchTrainTool{})
	reg.Register(&TrainDetailTool{TicketClient: ticketClient})
	reg.Register(&ListOrdersTool{OrderClient: orderClient})
	return reg
}

// Register 注册一个新工具
func (r *ToolRegistry) Register(t Tool) {
	if t == nil || strings.TrimSpace(t.Name()) == "" {
		return
	}
	if r.tools == nil {
		r.tools = map[string]Tool{}
	}
	r.tools[t.Name()] = t
}

// Get 根据名称获取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	if r == nil || r.tools == nil {
		return nil, false
	}
	t, ok := r.tools[name]
	return t, ok
}

// MissingSlots 校验必填槽位是否缺失
func MissingSlots(required []string, slots map[string]string) []string {
	var missing []string
	for _, k := range required {
		if strings.TrimSpace(slots[k]) == "" {
			missing = append(missing, k)
		}
	}
	return missing
}


package coupon

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"example_shop/common/db"
	model "example_shop/common/model/attraction_ticket"
	couponpb "example_shop/kitex_gen/coupon"
)

// CouponServiceImpl 为 Kitex 生成的 coupon.CouponService 接口提供具体实现。
// 这里放置“景点列表/详情查询”等业务逻辑；main.go 只负责启动服务并注入该实现。
type CouponServiceImpl struct{}

// NewCouponServiceImpl 返回一个可直接注入到 couponservice.NewServer 的 handler。
// 预留工厂函数的好处是：未来可在这里注入依赖（例如 logger、cache、配置等）。
func NewCouponServiceImpl() *CouponServiceImpl {
	return &CouponServiceImpl{}
}

// spotExtFields 用于解析 spot_info.ext_fields(JSON) 中的扩展字段。
// 由于当前表结构没有“主题标签/销量/评分/评价”等独立字段，所以采用 ext_fields 承载。
type spotExtFields struct {
	Tags    []string       `json:"tags"`
	Sales   int64          `json:"sales"`
	Rating  float64        `json:"rating"`
	Reviews []reviewExtRaw `json:"reviews"`
}

// reviewExtRaw 为 ext_fields.reviews 的 JSON 原始结构。
// 在没有独立评价表的情况下，先以扩展字段返回（或作为占位数据）。
type reviewExtRaw struct {
	UserID    *int64   `json:"user_id"`
	UserName  *string  `json:"user_name"`
	Rating    *float64 `json:"rating"`
	Content   *string  `json:"content"`
	CreatedAt *string  `json:"created_at"`
}

// parseSpotExt 将 JSON 扩展字段解析为结构体。
// 解析失败时不返回错误，而是降级为空结构，保证查询接口稳定返回。
func parseSpotExt(ext *model.JSON) spotExtFields {
	if ext == nil || len(*ext) == 0 {
		return spotExtFields{}
	}
	var v spotExtFields
	_ = json.Unmarshal([]byte(*ext), &v)
	return v
}

// containsAllTags 判断 have 是否包含 want 中的全部标签（AND 逻辑）。
// 场景：用户指定多个主题标签时，需要景点同时满足这些标签。
func containsAllTags(have []string, want []string) bool {
	if len(want) == 0 {
		return true
	}
	if len(have) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(have))
	for _, t := range have {
		set[t] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}

// defaultPage 对页码进行兜底：nil 或 <=0 视为 1。
func defaultPage(p *int32) int {
	if p == nil || *p <= 0 {
		return 1
	}
	return int(*p)
}

// defaultPageSize 对每页数量进行兜底：nil 或 <=0 视为 10，并限制最大 50。
func defaultPageSize(ps *int32) int {
	if ps == nil || *ps <= 0 {
		return 10
	}
	if *ps > 50 {
		return 50
	}
	return int(*ps)
}

// Test 为历史占位方法，保持兼容。
func (s *CouponServiceImpl) Test(ctx context.Context, req *couponpb.EmptyReq) (*couponpb.BaseResp, error) {
	return &couponpb.BaseResp{
		Code: 200,
		Msg:  "空方法执行成功（无任何业务逻辑）",
	}, nil
}

// SpotList 景点列表查询：支持关键字/省市/标签过滤 + 分页 + 排序。
// 说明：
// - 省市/名称：直接在 DB 层过滤（WHERE）。
// - 标签：从 ext_fields.tags 解析后在内存过滤（当前表无独立 tags 字段）。
// - 最低价：从 ticket_type 表聚合 MIN(price) 得到（单位：分）。
// - 销量/评分：从 ext_fields 读取，缺省返回 0。
func (s *CouponServiceImpl) SpotList(ctx context.Context, req *couponpb.SpotListReq) (*couponpb.SpotListResp, error) {
	page := defaultPage(req.Page)
	pageSize := defaultPageSize(req.PageSize)

	// 1) 构建基础查询：spot_info 表
	q := db.MysqlDB.WithContext(ctx).Model(&model.SpotInfo{})
	if req.Keyword != nil {
		kw := strings.TrimSpace(*req.Keyword)
		if kw != "" {
			q = q.Where("spot_name LIKE ?", "%"+kw+"%")
		}
	}
	if req.Province != nil && strings.TrimSpace(*req.Province) != "" {
		q = q.Where("province = ?", strings.TrimSpace(*req.Province))
	}
	if req.City != nil && strings.TrimSpace(*req.City) != "" {
		q = q.Where("city = ?", strings.TrimSpace(*req.City))
	}

	// 2) 查询满足“名称/省市”的候选景点集合
	var spots []model.SpotInfo
	if err := q.Find(&spots).Error; err != nil {
		return &couponpb.SpotListResp{Base: &couponpb.BaseResp{Code: 500, Msg: err.Error()}, Total: 0, List: []*couponpb.SpotBrief{}}, nil
	}

	// 3) 预先计算每个景点的最低票价：ticket_type 表按 spot_id 分组聚合 MIN(price)
	type minPriceRow struct {
		SpotID   uint64
		MinPrice int64
	}
	var rows []minPriceRow
	_ = db.MysqlDB.WithContext(ctx).
		Model(&model.TicketType{}).
		Select("spot_id, MIN(price) AS min_price").
		Group("spot_id").
		Scan(&rows).Error
	minPriceMap := make(map[uint64]int64, len(rows))
	for _, r := range rows {
		minPriceMap[r.SpotID] = r.MinPrice
	}

	// 4) 标签过滤 + 组装响应列表项
	targetTags := req.Tags
	briefs := make([]*couponpb.SpotBrief, 0, len(spots))
	for _, sp := range spots {
		ext := parseSpotExt(sp.ExtFields)
		if !containsAllTags(ext.Tags, targetTags) {
			continue
		}
		mp := minPriceMap[sp.ID]
		sales := ext.Sales
		rating := ext.Rating
		tags := ext.Tags
		if len(tags) == 0 {
			tags = nil
		}
		briefs = append(briefs, &couponpb.SpotBrief{
			SpotId:   int64(sp.ID),
			SpotName: sp.SpotName,
			Province: sp.Province,
			City:     sp.City,
			Address:  sp.Address,
			CoverImg: sp.CoverImg,
			OpenTime: sp.OpenTime,
			MinPrice: mp,
			Sales:    &sales,
			Rating:   &rating,
			Tags:     tags,
		})
	}

	// 5) 排序：价格可直接用 min_price；销量/评分来自 ext_fields（内存排序）
	sortField := couponpb.SpotSortField_DEFAULT
	if req.SortField != nil {
		sortField = *req.SortField
	}
	sortOrder := couponpb.SortOrder_DESC
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	less := func(i, j int) bool { return briefs[i].SpotId < briefs[j].SpotId }
	switch sortField {
	case couponpb.SpotSortField_PRICE:
		less = func(i, j int) bool {
			if sortOrder == couponpb.SortOrder_ASC {
				return briefs[i].MinPrice < briefs[j].MinPrice
			}
			return briefs[i].MinPrice > briefs[j].MinPrice
		}
	case couponpb.SpotSortField_SALES:
		less = func(i, j int) bool {
			li := int64(0)
			lj := int64(0)
			if briefs[i].Sales != nil {
				li = *briefs[i].Sales
			}
			if briefs[j].Sales != nil {
				lj = *briefs[j].Sales
			}
			if sortOrder == couponpb.SortOrder_ASC {
				return li < lj
			}
			return li > lj
		}

	default:
		less = func(i, j int) bool {
			if sortOrder == couponpb.SortOrder_ASC {
				return briefs[i].SpotId < briefs[j].SpotId
			}
			return briefs[i].SpotId > briefs[j].SpotId
		}
	}
	sort.SliceStable(briefs, less)

	// 6) 分页切片（page 从 1 开始）
	total := int64(len(briefs))
	start := (page - 1) * pageSize
	if start >= len(briefs) {
		return &couponpb.SpotListResp{Base: &couponpb.BaseResp{Code: 200, Msg: "ok"}, Total: total, List: []*couponpb.SpotBrief{}}, nil
	}
	end := start + pageSize
	if end > len(briefs) {
		end = len(briefs)
	}

	return &couponpb.SpotListResp{Base: &couponpb.BaseResp{Code: 200, Msg: "ok"}, Total: total, List: briefs[start:end]}, nil
}

// SpotDetail 景点详情查询：返回景点基础信息 + 门票类型列表 + 扩展字段（销量/评分/标签/评价）。
func (s *CouponServiceImpl) SpotDetail(ctx context.Context, req *couponpb.SpotDetailReq) (*couponpb.SpotDetailResp, error) {
	// 1) 查询景点基础信息
	var sp model.SpotInfo
	if err := db.MysqlDB.WithContext(ctx).First(&sp, "id = ?", req.SpotId).Error; err != nil {
		return &couponpb.SpotDetailResp{Base: &couponpb.BaseResp{Code: 404, Msg: "spot not found"}}, nil
	}

	// 2) 查询该景点下的门票类型列表
	var tts []model.TicketType
	if err := db.MysqlDB.WithContext(ctx).Find(&tts, "spot_id = ?", sp.ID).Error; err != nil {
		return &couponpb.SpotDetailResp{Base: &couponpb.BaseResp{Code: 500, Msg: err.Error()}}, nil
	}
	infos := make([]*couponpb.TicketTypeInfo, 0, len(tts))
	for _, tt := range tts {
		stock := int32(tt.Stock)
		infos = append(infos, &couponpb.TicketTypeInfo{
			TicketTypeId:  int64(tt.ID),
			TicketName:    tt.TicketName,
			Price:         tt.Price,
			OriginalPrice: tt.OriginalPrice,
			Stock:         stock,
			TicketStatus:  tt.TicketStatus,
			RefundRule:    tt.RefundRule,
			UseRule:       tt.UseRule,
		})
	}

	// 3) 读取扩展字段（标签/销量/评分/评价）
	ext := parseSpotExt(sp.ExtFields)
	sales := ext.Sales
	rating := ext.Rating
	tags := ext.Tags
	if len(tags) == 0 {
		tags = nil
	}

	var reviews []*couponpb.ReviewInfo
	if len(ext.Reviews) > 0 {
		reviews = make([]*couponpb.ReviewInfo, 0, len(ext.Reviews))
		for _, r := range ext.Reviews {
			reviews = append(reviews, &couponpb.ReviewInfo{
				UserId:    r.UserID,
				UserName:  r.UserName,
				Rating:    r.Rating,
				Content:   r.Content,
				CreatedAt: r.CreatedAt,
			})
		}
	}

	// 4) 组装响应体
	detail := &couponpb.SpotDetail{
		SpotId:       int64(sp.ID),
		SpotName:     sp.SpotName,
		SpotDesc:     sp.SpotDesc,
		Province:     sp.Province,
		City:         sp.City,
		Address:      sp.Address,
		CoverImg:     sp.CoverImg,
		OpenTime:     sp.OpenTime,
		ContactPhone: sp.ContactPhone,
		Sales:        &sales,
		Rating:       &rating,
		Tags:         tags,
		TicketTypes:  infos,
		Reviews:      reviews,
	}

	return &couponpb.SpotDetailResp{Base: &couponpb.BaseResp{Code: 200, Msg: "ok"}, Data: detail}, nil
}

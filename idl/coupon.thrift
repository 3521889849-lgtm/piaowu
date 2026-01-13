// 必须的Thrift文档头部，解决「not document」核心问题
namespace go coupon

// 极简响应体（仅占位）
struct BaseResp {
    1: i32 code,
    2: string msg
}

// 极简入参（仅占位）
struct EmptyReq {
    1: i64 id
}

// 排序字段
enum SpotSortField {
    DEFAULT = 0,
    SALES = 1,
    RATING = 2,
    PRICE = 3
}

// 排序方式
enum SortOrder {
    ASC = 0,
    DESC = 1
}

// 景点列表查询请求
struct SpotListReq {
    1: optional string keyword,        // 景点名称模糊搜索
    2: optional string province,       // 省份
    3: optional string city,           // 城市
    4: optional list<string> tags,     // 主题标签（自然风光、人文古迹等）
    5: optional i32 page,              // 页码，从1开始
    6: optional i32 page_size,         // 每页数量
    7: optional SpotSortField sort_field,
    8: optional SortOrder sort_order
}

// 景点列表项
struct SpotBrief {
    1: i64 spot_id,
    2: string spot_name,
    3: string province,
    4: string city,
    5: string address,
    6: string cover_img,
    7: string open_time,
    8: i64 min_price,                  // 该景点门票最低价（单位：分）
    9: optional i64 sales,             // 销量（如无数据可返回0）
    10: optional double rating,        // 评分（如无数据可返回0）
    11: optional list<string> tags
}

struct SpotListResp {
    1: BaseResp base,
    2: i64 total,
    3: list<SpotBrief> list
}

// 门票类型信息
struct TicketTypeInfo {
    1: i64 ticket_type_id,
    2: string ticket_name,
    3: i64 price,                      // 单位：分
    4: i64 original_price,             // 单位：分
    5: i32 stock,
    6: string ticket_status,
    7: string refund_rule,
    8: string use_rule
}

// 用户评价（当前项目无独立表时可先返回占位/从扩展字段读取）
struct ReviewInfo {
    1: optional i64 user_id,
    2: optional string user_name,
    3: optional double rating,
    4: optional string content,
    5: optional string created_at
}

// 景点详情
struct SpotDetail {
    1: i64 spot_id,
    2: string spot_name,
    3: optional string spot_desc,
    4: string province,
    5: string city,
    6: string address,
    7: string cover_img,
    8: string open_time,
    9: optional string contact_phone,
    10: optional i64 sales,
    11: optional double rating,
    12: optional list<string> tags,
    13: list<TicketTypeInfo> ticket_types,
    14: optional list<ReviewInfo> reviews
}

struct SpotDetailReq {
    1: i64 spot_id
}

struct SpotDetailResp {
    1: BaseResp base,
    2: optional SpotDetail data
}

// Kitex必须的Service定义（仅占位，无业务逻辑）
service CouponService {
    BaseResp Test(1: EmptyReq req)

    // 景点列表查询（多条件 + 分页 + 排序）
    SpotListResp SpotList(1: SpotListReq req)

    // 景点详情（含门票类型、规则、评价等）
    SpotDetailResp SpotDetail(1: SpotDetailReq req)
}
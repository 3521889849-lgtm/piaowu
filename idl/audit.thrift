namespace go audit

// ---------------------- 枚举定义 ----------------------

// 业务类型枚举
enum BizType {
    TICKET_ORDER = 1,    // 车票订单
    HOTEL_ORDER = 2,     // 酒店订单
    MERCHANT_ENTRY = 3,  // 商户入驻
    FLIGHT_ORDER = 4,    // 机票订单
    SCENIC_ORDER = 5     // 旅游门票
}

// 审核类型枚举
enum AuditType {
    AUTO = 1,    // 系统自动审核
    MANUAL = 2   // 人工审核
}

// 审核状态枚举
enum AuditStatus {
    PENDING = 1,    // 待审核
    PROCESSING = 2, // 审核中
    PASSED = 3,     // 审核通过
    REJECTED = 4,   // 审核驳回
    CANCELLED = 5   // 已撤销
}

// ---------------------- 基础结构 ----------------------

// 基础响应结构
struct BaseResp {
    1: i32 code,      // 状态码：0=成功，非0=失败
    2: string msg     // 响应信息
}

// 分页请求参数
struct Pagination {
    1: i32 page_num,  // 当前页码，默认1
    2: i32 page_size  // 每页大小，默认10
}

// ---------------------- 请求/响应结构 ----------------------

// 1. 提交审核请求
struct ApplyAuditReq {
    1: BizType biz_type,         // 业务类型
    2: string biz_id,            // 业务ID（如订单号、商户ID）
    3: string submitter_id,      // 提交人ID
    4: string content,           // 审核内容文本（或JSON序列化数据）
    5: list<string> attachments, // 附件链接列表
    6: map<string, string> extra // 扩展字段
}

struct ApplyAuditResp {
    1: BaseResp base_resp,
    2: i64 audit_id              // 生成的审核单ID
}

// 2. 获取人工审核任务列表请求
struct FetchManualTasksReq {
    1: string auditor_id,        // 审核员ID（可选，为空则拉取公共池）
    2: list<BizType> biz_types,  // 筛选业务类型
    3: AuditStatus status,       // 筛选状态（通常是 PENDING 或 PROCESSING）
    4: Pagination pagination     // 分页参数
}

struct AuditTask {
    1: i64 audit_id,             // 审核单ID
    2: BizType biz_type,         // 业务类型
    3: string biz_id,            // 业务ID
    4: string submitter_id,      // 提交人ID
    5: string content,           // 审核内容
    6: string apply_time,        // 提交时间（ISO8601格式字符串）
    7: AuditStatus status,       // 当前状态
    8: i32 priority              // 优先级（数值越高越优先）
}

struct FetchManualTasksResp {
    1: BaseResp base_resp,
    2: list<AuditTask> tasks,    // 任务列表
    3: i64 total                 // 总数
}

// 3. 处理人工审核结果请求
struct ProcessManualAuditReq {
    1: i64 audit_id,             // 审核单ID
    2: string auditor_id,        // 审核员ID
    3: bool is_passed,           // 是否通过：true=通过，false=驳回
    4: string remark,            // 审核备注/驳回原因
    5: string opinion            // 详细审核意见
}

struct ProcessManualAuditResp {
    1: BaseResp base_resp
}

// 4. 查询审核记录请求
struct GetAuditRecordReq {
    1: i64 audit_id,             // 审核单ID（优先使用）
    2: string biz_id,            // 业务ID（配合biz_type使用）
    3: BizType biz_type
}

// 审核流转日志
struct AuditLog {
    1: string operator_id,       // 操作人ID
    2: string operation,         // 操作类型（提交/通过/驳回等）
    3: string remark,            // 备注
    4: string create_time        // 操作时间
}

struct AuditDetail {
    1: AuditTask task_info,      // 任务基本信息
    2: list<AuditLog> logs,      // 流转日志
    3: string audit_result       // 最终审核结果说明
}

struct GetAuditRecordResp {
    1: BaseResp base_resp,
    2: AuditDetail detail        // 审核详情
}

// ---------------------- 服务接口定义 ----------------------

service AuditService {
    // 1. 提交审核请求（业务入口）
    ApplyAuditResp ApplyAudit(1: ApplyAuditReq req),

    // 2. 获取需人工处理的任务列表（运营后台）
    FetchManualTasksResp FetchManualTasks(1: FetchManualTasksReq req),

    // 3. 提交人工审核结果（运营后台）
    ProcessManualAuditResp ProcessManualAudit(1: ProcessManualAuditReq req),

    // 4. 查询审核历史/详情
    GetAuditRecordResp GetAuditRecord(1: GetAuditRecordReq req)
}

import axios, { AxiosError } from 'axios';
import { message } from 'antd';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/audit',
  timeout: 60000, // 增加到60秒
  headers: {
    'Content-Type': 'application/json',
  },
});


// 请求重试配置
const MAX_RETRIES = 2;
const RETRY_DELAY = 1000;

// 添加请求拦截器
api.interceptors.request.use(
  (config) => {
    // 可以在这里添加token等认证信息
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 添加响应拦截器处理错误
api.interceptors.response.use(
  (response) => {
    const res = response.data;
    // 处理业务逻辑错误码
    if (res && res.code !== undefined && res.code !== 0) {
      message.error(res.message || '操作失败');
      return Promise.reject(new Error(res.message || 'Error'));
    }
    return response;
  },
  async (error: AxiosError) => {
    const config: any = error.config;

    // 错误日志记录
    console.error('[API 错误]', {
      url: config?.url,
      method: config?.method,
      status: error.response?.status,
      data: error.response?.data,
      message: error.message
    });

    // 重试逻辑（仅对网络错误和5xx错误重试）
    if (!config || !config.retry) {
      config.retry = 0;
    }

    const shouldRetry = (
      (!error.response || error.response.status >= 500) &&
      config.retry < MAX_RETRIES
    );

    if (shouldRetry) {
      config.retry += 1;
      await new Promise(resolve => setTimeout(resolve, RETRY_DELAY * config.retry));
      return api(config);
    }

    if (error.response) {
      const status = error.response.status;
      const data: any = error.response.data;
      
      switch (status) {
        case 500:
          message.error({
            content: data?.message || '服务器内部错误，请稍后再试',
            duration: 5,
          });
          break;
        case 404:
          message.error('请求的资源不存在');
          break;
        case 403:
          message.error('您没有权限执行此操作');
          break;
        case 401:
          message.error('登录已过期，请重新登录');
          break;
        case 400:
          message.error(data?.message || '请求参数错误');
          break;
        default:
          message.error(data?.message || `请求失败: ${status}`);
      }
    } else if (error.request) {
      message.error('网络连接失败，请检查您的网络设置');
    } else {
      message.error('请求配置错误');
    }
    return Promise.reject(error);
  }
);


// 响应结构
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data?: T;
}

// 车票订单信息
export interface TicketOrderInfo {
  ticket_order_id: number;
  ticket_type: number;
  departure_station: string;
  arrival_station: string;
  departure_time: string;
  passenger_name: string;
  passenger_id_card: string;
  order_amount: number;
  ticket_extra?: string;
}

// 酒店订单信息
export interface HotelOrderInfo {
  hotel_id: number;
  hotel_name: string;
  hotel_address: string;
  room_type?: string;
  check_in_time?: string;
  check_out_time?: string;
  guest_name?: string;
  guest_id_card?: string;
  order_amount: number;
  hotel_extra?: string;
}

// 酒店入驻信息
export interface HotelSettleInfo {
  hotel_id: number;
  hotel_name: string;
  hotel_address: string;
  business_license: string;
  hotel_extra?: string;
}

// 提交审核请求
export interface SubmitAuditRequest {
  business_type: number;
  business_id: number;
  apply_reason: string;
  ticket_order?: TicketOrderInfo;
  hotel_order?: HotelOrderInfo;
  hotel_settle?: HotelSettleInfo;
  flight_order?: FlightOrderInfo;
  scenic_order?: ScenicOrderInfo;
}

// 机票订单信息
export interface FlightOrderInfo {
  flight_order_id: number;
  flight_type: number;
  flight_no: string;
  airline: string;
  departure_airport: string;
  arrival_airport: string;
  departure_time: string;
  arrival_time: string;
  cabin_class: string;
  passenger_name: string;
  passenger_id_card: string;
  order_amount: number;
}

// 旅游门票信息
export interface ScenicOrderInfo {
  scenic_order_id: number;
  ticket_type: number;
  scenic_name: string;
  scenic_address: string;
  ticket_name: string;
  visit_date: string;
  ticket_quantity: number;
  unit_price: number;
  order_amount: number;
  contact_name: string;
  contact_phone: string;
}

// 提交审核响应
export interface SubmitAuditResponse {
  audit_id: number;
  business_type: number;
  business_id: number;
  audit_status: number;
  submit_time: string;
}

// 审核详情
export interface AuditDetail {
  audit_id: number;
  business_type: number;
  business_id: number;
  audit_status: number;
  audit_status_text: string;
  submit_user_id: number;
  submit_user_name: string;
  audit_user_id: number;
  audit_user_name: string;
  audit_remark: string;
  audit_time?: string;
  submit_time: string;
  ticket_order?: {
    ticket_order_id: number;
    ticket_type: number;
    ticket_type_text: string;
    departure_station: string;
    arrival_station: string;
    departure_time: string;
    passenger_name: string;
    passenger_id_card: string;
    order_amount: number;
    apply_reason: string;
  };
  hotel_order?: {
    hotel_id: number;
    hotel_name: string;
    hotel_address: string;
    room_type: string;
    check_in_time?: string;
    check_out_time?: string;
    guest_name: string;
    guest_id_card: string;
    order_amount: number;
    apply_reason: string;
  };
  hotel_settle?: {
    hotel_id: number;
    hotel_name: string;
    hotel_address: string;
    business_license: string;
    apply_reason: string;
  };
  flight_order?: {
    flight_order_id: number;
    flight_type: number;
    flight_no: string;
    airline: string;
    departure_airport: string;
    arrival_airport: string;
    departure_time: string;
    arrival_time: string;
    cabin_class: string;
    passenger_name: string;
    order_amount: number;
    apply_reason: string;
  };
  scenic_order?: {
    scenic_order_id: number;
    ticket_type: number;
    scenic_name: string;
    scenic_address: string;
    ticket_name: string;
    visit_date: string;
    ticket_quantity: number;
    unit_price: number;
    order_amount: number;
    contact_name: string;
    contact_phone: string;
    apply_reason: string;
  };
}

// 审核列表响应
export interface AuditListResponse {
  total: number;
  list: AuditDetail[];
}

// 查询审核请求
export interface QueryAuditRequest {
  audit_id?: number;
  business_type?: number;
  business_id?: number;
  audit_status?: number;
  page?: number;
  page_size?: number;
}

// 审核操作请求
export interface AuditActionRequest {
  audit_id: number;
  action: 'approve' | 'reject';
  audit_remark?: string;
}

// API 方法
export const submitAudit = async (req: SubmitAuditRequest): Promise<SubmitAuditResponse> => {
  const { data } = await api.post<ApiResponse<SubmitAuditResponse>>('/submit', req);
  return data.data!;
};

export const queryAuditList = async (req: QueryAuditRequest): Promise<AuditListResponse> => {
  const params = new URLSearchParams();
  if (req.audit_id) params.append('audit_id', req.audit_id.toString());
  if (req.business_type) params.append('business_type', req.business_type.toString());
  if (req.business_id) params.append('business_id', req.business_id.toString());
  if (req.audit_status) params.append('audit_status', req.audit_status.toString());
  if (req.page) params.append('page', req.page.toString());
  if (req.page_size) params.append('page_size', req.page_size.toString());
  
  const { data } = await api.get<ApiResponse<AuditListResponse>>(`/list?${params.toString()}`);
  return data.data!;
};

export const processAudit = async (req: AuditActionRequest): Promise<void> => {
  await api.post<ApiResponse>('/process', req);
};

// 常量映射
export const BusinessTypeMap: Record<number, string> = {
  1: '车票订单',
  2: '酒店订单',
  3: '酒店入驻',
  4: '机票订单',
  5: '旅游门票',
};

export const AuditStatusMap: Record<number, string> = {
  1: '待审核',
  2: '审核中',
  3: '通过',
  4: '驳回',
  5: '撤销',
};

export const TicketTypeMap: Record<number, string> = {
  1: '高铁',
  2: '动车',
  3: '普通火车',
  4: '汽车票',
};


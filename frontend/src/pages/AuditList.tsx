import React, { useEffect, useState, useMemo } from 'react';
import { 
  Table, Tag, Button, Modal, Form, Input, Radio, message, Space, 
  Descriptions, Card, Select, Row, Col, Badge, Skeleton, Typography
} from 'antd';
import { 
  ReloadOutlined, EyeOutlined, CheckOutlined, CloseOutlined,
  SearchOutlined, ClearOutlined
} from '@ant-design/icons';
import { 
  queryAuditList, 
  processAudit,
  type AuditDetail,
  BusinessTypeMap, 
  AuditStatusMap
} from '../api';
import dayjs from 'dayjs';

const { Text } = Typography;

const AuditList: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<AuditDetail[]>([]);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(10);
  const [modalOpen, setModalOpen] = useState(false);
  const [currentTask, setCurrentTask] = useState<AuditDetail | null>(null);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [detailData, setDetailData] = useState<AuditDetail | null>(null);
  const [form] = Form.useForm();
  const [filterForm] = Form.useForm();
  const [filters, setFilters] = useState<any>({});

  // 简化统计
  const stats = useMemo(() => {
    const pending = data.filter(d => d.audit_status === 1).length;
    const approved = data.filter(d => d.audit_status === 3).length;
    const rejected = data.filter(d => d.audit_status === 4).length;
    return { pending, approved, rejected };
  }, [data]);

  const loadData = async (page = currentPage, filterParams = filters) => {
    setLoading(true);
    try {
      const res = await queryAuditList({ page, page_size: pageSize, ...filterParams });
      setData(res.list || []);
      setTotal(res.total);
      setCurrentPage(page);
    } catch (e) {
      console.error('加载数据失败:', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadData(); }, []);

  const handleProcess = (task: AuditDetail) => {
    setCurrentTask(task);
    form.resetFields();
    setModalOpen(true);
  };

  const [detailLoading, setDetailLoading] = useState(false);

  const handleView = async (task: AuditDetail) => {
    setDetailLoading(true);
    setDetailModalOpen(true);
    try {
      const res = await queryAuditRecord({ audit_id: task.audit_id });
      setDetailData(res.data);
    } catch (e) {
      console.error('获取详情失败:', e);
      message.error('获取详情失败');
    } finally {
      setDetailLoading(false);
    }
  };

  const submitProcess = async () => {
    try {
      const values = await form.validateFields();
      await processAudit({
        audit_id: currentTask!.audit_id,
        action: values.action,
        audit_remark: values.audit_remark || ''
      });
      message.success('处理成功');
      setModalOpen(false);
      loadData();
    } catch (e) {
      console.error('处理失败:', e);
    }
  };

  const handleFilter = () => {
    filterForm.validateFields().then(values => {
      const params: any = {};
      if (values.business_type) params.business_type = values.business_type;
      if (values.audit_status) params.audit_status = values.audit_status;
      if (values.audit_id) params.audit_id = values.audit_id;
      setFilters(params);
      loadData(1, params);
    });
  };

  const handleClearFilter = () => {
    filterForm.resetFields();
    setFilters({});
    loadData(1, {});
  };

  const columns = useMemo(() => [
    { 
      title: 'ID', 
      dataIndex: 'audit_id', 
      width: 80,
      fixed: 'left' as const,
    },
    { 
      title: '业务类型', 
      dataIndex: 'business_type', 
      width: 100,
      render: (t: number) => <Tag color="blue">{BusinessTypeMap[t] || `类型${t}`}</Tag> 
    },
    { 
      title: '状态', 
      dataIndex: 'audit_status', 
      width: 90,
      render: (status: number, record: AuditDetail) => {
        const colorMap: Record<number, string> = {
          1: 'orange', 2: 'processing', 3: 'success', 4: 'error', 5: 'default'
        };
        return <Badge status={colorMap[status] as any} text={record.audit_status_text} />;
      }
    },
    { title: '提交人', dataIndex: 'submit_user_name', width: 100 },
    { 
      title: '提交时间', 
      dataIndex: 'submit_time', 
      width: 160,
      render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm')
    },
    { 
      title: '审核人', 
      dataIndex: 'audit_user_name', 
      width: 100,
      render: (name: string) => name || '-'
    },
    {
      title: '操作',
      width: 140,
      fixed: 'right' as const,
      render: (_: any, record: AuditDetail) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleView(record)}>
            详情
          </Button>
          {record.audit_status === 1 && (
            <Button type="primary" size="small" onClick={() => handleProcess(record)}>
              审核
            </Button>
          )}
        </Space>
      )
    }
  ], []);

  return (
    <div>
      {loading && data.length === 0 ? (
        <Card><Skeleton active paragraph={{ rows: 10 }} /></Card>
      ) : (
        <Card
          title={
            <Space split={<span style={{ color: '#d9d9d9' }}>|</span>}>
              <Text>审核任务</Text>
              <Text type="secondary">
                待审核 <Text strong style={{ color: '#faad14' }}>{stats.pending}</Text>
              </Text>
              <Text type="secondary">
                已通过 <Text strong style={{ color: '#52c41a' }}>{stats.approved}</Text>
              </Text>
              <Text type="secondary">
                已驳回 <Text strong style={{ color: '#ff4d4f' }}>{stats.rejected}</Text>
              </Text>
              <Text type="secondary">共 {total} 条</Text>
            </Space>
          }
          extra={
            <Button icon={<ReloadOutlined />} onClick={() => loadData()}>刷新</Button>
          }
        >
          {/* 简洁筛选栏 */}
          <div style={{ marginBottom: 16, padding: 12, background: '#fafafa', borderRadius: 6 }}>
            <Form form={filterForm} layout="inline" size="small">
              <Form.Item name="audit_id" style={{ marginBottom: 0 }}>
                <Input placeholder="审核ID" style={{ width: 100 }} allowClear />
              </Form.Item>
              <Form.Item name="business_type" style={{ marginBottom: 0 }}>
                <Select placeholder="业务类型" style={{ width: 120 }} allowClear>
                  {Object.entries(BusinessTypeMap).map(([k, v]) => (
                    <Select.Option key={k} value={Number(k)}>{v}</Select.Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item name="audit_status" style={{ marginBottom: 0 }}>
                <Select placeholder="审核状态" style={{ width: 120 }} allowClear>
                  {Object.entries(AuditStatusMap).map(([k, v]) => (
                    <Select.Option key={k} value={Number(k)}>{v}</Select.Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item style={{ marginBottom: 0 }}>
                <Space size="small">
                  <Button type="primary" icon={<SearchOutlined />} onClick={handleFilter}>查询</Button>
                  <Button icon={<ClearOutlined />} onClick={handleClearFilter}>重置</Button>
                </Space>
              </Form.Item>
            </Form>
          </div>
        
          <Table 
            columns={columns} 
            dataSource={data} 
            rowKey="audit_id" 
            loading={loading}
            size="small"
            scroll={{ x: 900 }}
            pagination={{ 
              current: currentPage,
              total, 
              pageSize, 
              showSizeChanger: false,
              showTotal: (t) => `共 ${t} 条`,
              onChange: (page) => loadData(page)
            }}
          />
        </Card>
      )}

      {/* 审核处理弹窗 */}
      <Modal
        title={`处理审核 #${currentTask?.audit_id}`}
        open={modalOpen}
        onOk={submitProcess}
        okText="确认"
        cancelText="取消"
        onCancel={() => setModalOpen(false)}
        width={500}
      >
        {currentTask && (
          <>
            <Descriptions column={2} size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label="业务类型">
                <Tag color="blue">{BusinessTypeMap[currentTask.business_type]}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="提交人">{currentTask.submit_user_name}</Descriptions.Item>
            </Descriptions>

            <Form form={form} layout="vertical">
              <Form.Item name="action" label="审核结果" rules={[{ required: true, message: '请选择' }]}>
                <Radio.Group buttonStyle="solid">
                  <Radio.Button value="approve" style={{ color: '#52c41a' }}>
                    <CheckOutlined /> 通过
                  </Radio.Button>
                  <Radio.Button value="reject" style={{ color: '#ff4d4f' }}>
                    <CloseOutlined /> 驳回
                  </Radio.Button>
                </Radio.Group>
              </Form.Item>
              <Form.Item name="audit_remark" label="备注">
                <Input.TextArea rows={3} placeholder="审核意见（选填）" maxLength={500} showCount />
              </Form.Item>
            </Form>
          </>
        )}
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title={`审核详情 #${detailData?.audit_id}`}
        open={detailModalOpen}
        footer={<Button onClick={() => setDetailModalOpen(false)}>关闭</Button>}
        width={700}
        onCancel={() => setDetailModalOpen(false)}
      >
        {detailData && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="审核ID">{detailData.audit_id}</Descriptions.Item>
              <Descriptions.Item label="业务ID">{detailData.business_id}</Descriptions.Item>
              <Descriptions.Item label="业务类型">
                <Tag color="blue">{BusinessTypeMap[detailData.business_type]}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Badge 
                  status={detailData.audit_status === 3 ? 'success' : (detailData.audit_status === 4 ? 'error' : 'processing')} 
                  text={detailData.audit_status_text}
                />
              </Descriptions.Item>
              <Descriptions.Item label="提交人">{detailData.submit_user_name}</Descriptions.Item>
              <Descriptions.Item label="提交时间">{dayjs(detailData.submit_time).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>
              <Descriptions.Item label="审核人">{detailData.audit_user_name || '-'}</Descriptions.Item>
              <Descriptions.Item label="审核时间">{detailData.audit_time ? dayjs(detailData.audit_time).format('YYYY-MM-DD HH:mm') : '-'}</Descriptions.Item>
              {detailData.audit_remark && (
                <Descriptions.Item label="备注" span={2}>{detailData.audit_remark}</Descriptions.Item>
              )}
            </Descriptions>

            {/* 车票订单 */}
            {detailData.ticket_order && (
              <Card title="车票订单" size="small" type="inner">
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="订单ID">{detailData.ticket_order.ticket_order_id}</Descriptions.Item>
                  <Descriptions.Item label="车票类型"><Tag>{detailData.ticket_order.ticket_type_text}</Tag></Descriptions.Item>
                  <Descriptions.Item label="出发站">{detailData.ticket_order.departure_station}</Descriptions.Item>
                  <Descriptions.Item label="到达站">{detailData.ticket_order.arrival_station}</Descriptions.Item>
                  <Descriptions.Item label="乘客">{detailData.ticket_order.passenger_name}</Descriptions.Item>
                  <Descriptions.Item label="金额"><Text type="danger" strong>¥{detailData.ticket_order.order_amount.toFixed(2)}</Text></Descriptions.Item>
                </Descriptions>
              </Card>
            )}

            {/* 酒店订单 */}
            {detailData.hotel_order && (
              <Card title="酒店订单" size="small" type="inner">
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="酒店">{detailData.hotel_order.hotel_name}</Descriptions.Item>
                  <Descriptions.Item label="房型">{detailData.hotel_order.room_type}</Descriptions.Item>
                  <Descriptions.Item label="入住人">{detailData.hotel_order.guest_name}</Descriptions.Item>
                  <Descriptions.Item label="金额"><Text type="danger" strong>¥{detailData.hotel_order.order_amount.toFixed(2)}</Text></Descriptions.Item>
                </Descriptions>
              </Card>
            )}

            {/* 酒店入驻 */}
            {detailData.hotel_settle && (
              <Card title="酒店入驻" size="small" type="inner">
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="酒店名称">{detailData.hotel_settle.hotel_name}</Descriptions.Item>
                  <Descriptions.Item label="执照编号">{detailData.hotel_settle.business_license}</Descriptions.Item>
                  <Descriptions.Item label="地址" span={2}>{detailData.hotel_settle.hotel_address}</Descriptions.Item>
                </Descriptions>
              </Card>
            )}

            {/* 机票订单 */}
            {detailData.flight_order && (
              <Card title="机票订单" size="small" type="inner">
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="航班号">{detailData.flight_order.flight_no}</Descriptions.Item>
                  <Descriptions.Item label="航空公司">{detailData.flight_order.airline}</Descriptions.Item>
                  <Descriptions.Item label="出发机场">{detailData.flight_order.departure_airport}</Descriptions.Item>
                  <Descriptions.Item label="到达机场">{detailData.flight_order.arrival_airport}</Descriptions.Item>
                  <Descriptions.Item label="起飞时间">{dayjs(detailData.flight_order.departure_time).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>
                  <Descriptions.Item label="舱位">{detailData.flight_order.cabin_class}</Descriptions.Item>
                  <Descriptions.Item label="乘客">{detailData.flight_order.passenger_name}</Descriptions.Item>
                  <Descriptions.Item label="金额"><Text type="danger" strong>¥{detailData.flight_order.order_amount.toFixed(2)}</Text></Descriptions.Item>
                </Descriptions>
              </Card>
            )}

            {/* 旅游门票 */}
            {detailData.scenic_order && (
              <Card title="旅游门票" size="small" type="inner">
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="景区/场馆">{detailData.scenic_order.scenic_name}</Descriptions.Item>
                  <Descriptions.Item label="门票名称">{detailData.scenic_order.ticket_name}</Descriptions.Item>
                  <Descriptions.Item label="游玩日期">{dayjs(detailData.scenic_order.visit_date).format('YYYY-MM-DD')}</Descriptions.Item>
                  <Descriptions.Item label="数量">{detailData.scenic_order.ticket_quantity}</Descriptions.Item>
                  <Descriptions.Item label="联系人">{detailData.scenic_order.contact_name}</Descriptions.Item>
                  <Descriptions.Item label="金额"><Text type="danger" strong>¥{detailData.scenic_order.order_amount.toFixed(2)}</Text></Descriptions.Item>
                  <Descriptions.Item label="地址" span={2}>{detailData.scenic_order.scenic_address}</Descriptions.Item>
                </Descriptions>
              </Card>
            )}
          </Space>
        )}
      </Modal>
    </div>
  );
};

export default AuditList;

import React, { useEffect, useState, useMemo } from 'react';
import {
  Table, Tag, Button, Modal, Form, Input, Radio, message, Space,
  Descriptions, Card, Select, Badge, Typography, Tooltip,
  Divider, Row, Col, theme
} from 'antd';
import {
  ReloadOutlined, EyeOutlined, CheckCircleOutlined, CloseCircleOutlined,
  SearchOutlined, ClearOutlined, FilterOutlined
} from '@ant-design/icons';
import {
  queryAuditList,
  processAudit,
  queryAuditRecord,
  type AuditDetail,
  BusinessTypeMap,
  AuditStatusMap
} from '../api';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const { Text, Title } = Typography;

const AuditList: React.FC = () => {
  const { token } = theme.useToken();
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

  const handleView = async (task: AuditDetail) => {
    setDetailModalOpen(true);
    try {
      const res = await queryAuditRecord({ audit_id: task.audit_id });
      setDetailData(res);
    } catch (e) {
      console.error('获取详情失败:', e);
      message.error('获取详情失败');
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
    filterForm.validateFields().then((values: any) => {
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
      title: '审核 ID',
      dataIndex: 'audit_id',
      width: 100,
      fixed: 'left' as const,
      render: (t: number) => <Text code copyable>{t}</Text>
    },
    {
      title: '业务类型',
      dataIndex: 'business_type',
      width: 120,
      render: (t: number) => {
        const config: any = {
          1: { color: 'blue', icon: '🎫' },
          2: { color: 'green', icon: '🏨' },
          3: { color: 'purple', icon: '🏢' },
          4: { color: 'gold', icon: '✈️' },
          5: { color: 'magenta', icon: '🎡' },
        };
        const item = config[t] || { color: 'default', icon: '📄' };
        return (
          <Tag color={item.color} style={{ border: 0, padding: '2px 8px' }}>
            {item.icon} {BusinessTypeMap[t] || `Type ${t}`}
          </Tag>
        );
      }
    },
    {
      title: '状态',
      dataIndex: 'audit_status',
      width: 100,
      render: (status: number) => {
        const statusConfig: any = {
          1: { status: 'warning', color: '#faad14', text: '待审核' },
          2: { status: 'processing', color: '#1890ff', text: '审核中' },
          3: { status: 'success', color: '#52c41a', text: '已通过' },
          4: { status: 'error', color: '#ff4d4f', text: '已驳回' },
          5: { status: 'default', color: '#d9d9d9', text: '已撤销' },
        };
        const cfg = statusConfig[status] || statusConfig[5];
        return <Badge status={cfg.status} text={cfg.text} />;
      }
    },
    { title: '提交人', dataIndex: 'submit_user_name', width: 120 },
    {
      title: '提交时间',
      dataIndex: 'submit_time',
      width: 160,
      render: (t: string) => (
        <Tooltip title={t}>
          <span style={{ color: token.colorTextSecondary }}>{dayjs(t).fromNow()}</span>
        </Tooltip>
      )
    },
    {
      title: '操作',
      width: 120,
      fixed: 'right' as const,
      render: (_: any, record: AuditDetail) => (
        <Space size={0}>
          <Tooltip title="查看详情">
            <Button type="text" shape="circle" icon={<EyeOutlined />} onClick={() => handleView(record)} />
          </Tooltip>
          {record.audit_status === 1 && (
            <Tooltip title="立即审核">
              <Button type="link" size="small" onClick={() => handleProcess(record)}>
                审核
              </Button>
            </Tooltip>
          )}
        </Space>
      )
    }
  ], [token.colorTextSecondary]);

  return (
    <div style={{ maxWidth: 1600, margin: '0 auto' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>审核任务列表</Title>
          <Text type="secondary">管理所有提交的审核请求</Text>
        </div>
        <div>
          <Button icon={<ReloadOutlined />} onClick={() => loadData()}>刷新</Button>
        </div>
      </div>

      <Card
        bordered={false}
        className="glass-card"
        style={{ marginBottom: 24, padding: 0 }}
        styles={{ body: { padding: '24px' } }}
      >
        {/* 高级筛选栏 - 适配暗色模式 */}
        <div style={{ marginBottom: 24, padding: '16px', background: token.colorBgElevated, borderRadius: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <Form form={filterForm} layout="inline" size="middle" style={{ gap: 8, flexWrap: 'wrap' }}>
            <Form.Item name="audit_id" style={{ marginBottom: 8 }}>
              <Input prefix={<SearchOutlined style={{ color: token.colorTextSecondary }} />} placeholder="搜索 ID" style={{ width: 140 }} allowClear />
            </Form.Item>
            <Form.Item name="business_type" style={{ marginBottom: 8 }}>
              <Select placeholder="业务类型" style={{ width: 140 }} allowClear>
                {Object.entries(BusinessTypeMap).map(([k, v]) => (
                  <Select.Option key={k} value={Number(k)}>{v}</Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="audit_status" style={{ marginBottom: 8 }}>
              <Select placeholder="审核状态" style={{ width: 140 }} allowClear>
                {Object.entries(AuditStatusMap).map(([k, v]) => (
                  <Select.Option key={k} value={Number(k)}>{v}</Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item style={{ marginBottom: 8 }}>
              <Button type="primary" icon={<FilterOutlined />} onClick={handleFilter}>筛选</Button>
            </Form.Item>
            <Form.Item style={{ marginBottom: 8 }}>
              <Button type="text" icon={<ClearOutlined />} onClick={handleClearFilter} style={{ color: token.colorTextSecondary }}>重置</Button>
            </Form.Item>
          </Form>
        </div>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="audit_id"
          loading={loading}
          size="middle"
          scroll={{ x: 1000 }}
          pagination={{
            current: currentPage,
            total,
            pageSize,
            showSizeChanger: true,
            showTotal: (t) => <Text type="secondary">共 {t} 条记录</Text>,
            onChange: (page) => loadData(page)
          }}
        />
      </Card>

      {/* 审核处理弹窗 */}
      <Modal
        title={null}
        open={modalOpen}
        onOk={submitProcess}
        okText="提交结果"
        cancelText="取消"
        onCancel={() => setModalOpen(false)}
        width={480}
        centered
      >
        {currentTask && (
          <>
            <div style={{ marginBottom: 24 }}>
              <Title level={4} style={{ marginTop: 0 }}>处理审核 #{currentTask.audit_id}</Title>
              <Text type="secondary">请仔细核对业务信息后进行操作</Text>
            </div>

            <div style={{ background: token.colorBgElevated, padding: 16, borderRadius: 8, marginBottom: 24 }}>
              <Descriptions column={1} size="small">
                <Descriptions.Item label="业务类型">{BusinessTypeMap[currentTask.business_type]}</Descriptions.Item>
                <Descriptions.Item label="提交人">{currentTask.submit_user_name}</Descriptions.Item>
                <Descriptions.Item label="提交时间">{dayjs(currentTask.submit_time).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
              </Descriptions>
            </div>

            <Form form={form} layout="vertical">
              <Form.Item name="action" label={<Text strong>审核结论</Text>} rules={[{ required: true, message: '请选择审核结果' }]}>
                <Radio.Group buttonStyle="solid" style={{ width: '100%' }}>
                  <Row gutter={16}>
                    <Col span={12}>
                      <Radio.Button value="approve" style={{ width: '100%', textAlign: 'center', height: 40, lineHeight: '40px', background: 'transparent' }}>
                        <Space><CheckCircleOutlined style={{ color: '#52c41a' }} /> 通过</Space>
                      </Radio.Button>
                    </Col>
                    <Col span={12}>
                      <Radio.Button value="reject" style={{ width: '100%', textAlign: 'center', height: 40, lineHeight: '40px', background: 'transparent' }}>
                        <Space><CloseCircleOutlined style={{ color: '#ff4d4f' }} /> 驳回</Space>
                      </Radio.Button>
                    </Col>
                  </Row>
                </Radio.Group>
              </Form.Item>
              <Form.Item name="audit_remark" label="审核意见">
                <Input.TextArea rows={4} placeholder="请输入详细的审核意见..." maxLength={500} showCount style={{ borderRadius: 8 }} />
              </Form.Item>
            </Form>
          </>
        )}
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title={null}
        open={detailModalOpen}
        footer={<div style={{ textAlign: 'center' }}><Button type="primary" onClick={() => setDetailModalOpen(false)} style={{ minWidth: 100 }}>关闭</Button></div>}
        width={720}
        onCancel={() => setDetailModalOpen(false)}
        centered
      >
        {detailData && (
          <div>
            <div style={{ background: token.colorBgElevated, padding: '24px 32px', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
                <div>
                  <Title level={4} style={{ margin: 0 }}>审核单据详情</Title>
                  <Text type="secondary">ID: {detailData.audit_id} · {dayjs(detailData.submit_time).format('YYYY-MM-DD')}</Text>
                </div>
                <Tag color="blue" style={{ fontSize: 14, padding: '4px 12px' }}>{BusinessTypeMap[detailData.business_type]}</Tag>
              </div>
            </div>

            <div style={{ padding: '24px 32px', maxHeight: '60vh', overflowY: 'auto' }}>
              <Descriptions bordered column={2} size="middle" labelStyle={{ width: 120 }}>
                <Descriptions.Item label="状态">
                  <Badge
                    status={detailData.audit_status === 3 ? 'success' : (detailData.audit_status === 4 ? 'error' : 'processing')}
                    text={detailData.audit_status_text}
                  />
                </Descriptions.Item>
                <Descriptions.Item label="业务 ID">{detailData.business_id}</Descriptions.Item>
                <Descriptions.Item label="提交人">{detailData.submit_user_name}</Descriptions.Item>
                <Descriptions.Item label="提交时间">{dayjs(detailData.submit_time).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>
                <Descriptions.Item label="审核人">{detailData.audit_user_name || '-'}</Descriptions.Item>
                <Descriptions.Item label="审核时间">{detailData.audit_time ? dayjs(detailData.audit_time).format('YYYY-MM-DD HH:mm') : '-'}</Descriptions.Item>
                {detailData.audit_remark && (
                  <Descriptions.Item label="备注" span={2}>{detailData.audit_remark}</Descriptions.Item>
                )}
              </Descriptions>

              <Divider style={{ margin: '32px 0 16px' }}>
                <Text type="secondary" style={{ fontSize: 13 }}>业务详情数据</Text>
              </Divider>

              {/* 动态渲染业务详情卡片 */}
              {detailData.ticket_order && (
                <BusinessCard title="车票订单" data={detailData.ticket_order} />
              )}
              {detailData.hotel_order && (
                <BusinessCard title="酒店订单" data={detailData.hotel_order} />
              )}
              {detailData.hotel_settle && (
                <BusinessCard title="酒店入驻" data={detailData.hotel_settle} />
              )}
              {detailData.flight_order && (
                <BusinessCard title="机票订单" data={detailData.flight_order} />
              )}
              {detailData.scenic_order && (
                <BusinessCard title="旅游门票" data={detailData.scenic_order} />
              )}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

// 辅助组件：通用业务卡片
const BusinessCard: React.FC<{ title: string, data: any }> = ({ title, data }) => {
  const { token } = theme.useToken();
  return (
    <Card title={title} type="inner" size="small" style={{ background: token.colorBgContainer, border: `1px solid ${token.colorBorderSecondary}` }}>
      <Descriptions column={2} size="small">
        {Object.entries(data).map(([key, value]) => {
          const label = key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
          return (
            <Descriptions.Item key={key} label={label}>
              {typeof value === 'number' && key.includes('amount') ? (
                <Text type="danger" strong>¥{(value as number).toFixed(2)}</Text>
              ) : (
                String(value)
              )}
            </Descriptions.Item>
          );
        })}
      </Descriptions>
    </Card>
  );
};

export default AuditList;

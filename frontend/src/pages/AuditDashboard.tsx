import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Empty, Spin, Progress, Typography, List, Tag, Button, Timeline, Avatar, Space, Tooltip } from 'antd';
import { 
  CheckCircleOutlined, CloseCircleOutlined, 
  ClockCircleOutlined, SyncOutlined,
  FileTextOutlined, StopOutlined,
  PlusOutlined, UnorderedListOutlined,
  RightOutlined, CalendarOutlined,
  ThunderboltOutlined, FieldTimeOutlined
} from '@ant-design/icons';
import { queryAuditList, BusinessTypeMap } from '../api';
import { useNavigate } from 'react-router-dom';

const { Title, Text } = Typography;

// 业务类型图标和颜色配置
const bizTypeConfig: Record<number, { icon: string; color: string; bg: string }> = {
  1: { icon: '🎫', color: '#1890ff', bg: '#e6f7ff' },
  2: { icon: '🏨', color: '#52c41a', bg: '#f6ffed' },
  3: { icon: '🏢', color: '#722ed1', bg: '#f9f0ff' },
  4: { icon: '✈️', color: '#fa8c16', bg: '#fff7e6' },
  5: { icon: '🎡', color: '#eb2f96', bg: '#fff0f6' },
};

// 审核状态配置
const statusConfig: Record<number, { text: string; color: string; icon: React.ReactNode }> = {
  1: { text: '待审核', color: 'warning', icon: <ClockCircleOutlined /> },
  2: { text: '审核中', color: 'processing', icon: <SyncOutlined spin /> },
  3: { text: '已通过', color: 'success', icon: <CheckCircleOutlined /> },
  4: { text: '已驳回', color: 'error', icon: <CloseCircleOutlined /> },
  5: { text: '已撤销', color: 'default', icon: <StopOutlined /> },
};

const AuditDashboard: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [recentList, setRecentList] = useState<any[]>([]);
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    processing: 0,
    approved: 0,
    rejected: 0,
    cancelled: 0,
    byType: {} as Record<number, number>,
    todayTotal: 0,
    todayPending: 0,
    todayApproved: 0,
    todayRejected: 0,
  });

  const loadStats = async () => {
    try {
      setLoading(true);
      let allRecords: any[] = [];
      let page = 1;
      const pageSize = 100;
      
      while (true) {
        const res = await queryAuditList({ page, page_size: pageSize });
        allRecords = allRecords.concat(res.list);
        if (allRecords.length >= res.total || res.list.length < pageSize) break;
        page++;
      }
      
      // 获取今天的日期范围
      const today = new Date();
      today.setHours(0, 0, 0, 0);
      
      const newStats = {
        total: allRecords.length,
        pending: 0,
        processing: 0,
        approved: 0,
        rejected: 0,
        cancelled: 0,
        byType: {} as Record<number, number>,
        todayTotal: 0,
        todayPending: 0,
        todayApproved: 0,
        todayRejected: 0,
      };

      allRecords.forEach(item => {
        const createTime = new Date(item.created_at);
        const isToday = createTime >= today;
        
        if (isToday) {
          newStats.todayTotal++;
        }
        
        switch (item.audit_status) {
          case 1: 
            newStats.pending++; 
            if (isToday) newStats.todayPending++;
            break;
          case 2: newStats.processing++; break;
          case 3: 
            newStats.approved++; 
            if (isToday) newStats.todayApproved++;
            break;
          case 4: 
            newStats.rejected++; 
            if (isToday) newStats.todayRejected++;
            break;
          case 5: newStats.cancelled++; break;
        }
        if (!newStats.byType[item.business_type]) {
          newStats.byType[item.business_type] = 0;
        }
        newStats.byType[item.business_type]++;
      });

      setStats(newStats);
      // 获取最近5条记录
      setRecentList(allRecords.slice(0, 5));
    } catch (e) {
      console.error('加载统计数据失败:', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStats();
    const timer = setInterval(loadStats, 30000);
    return () => clearInterval(timer);
  }, []);

  // 计算通过率和驳回率
  const completedTotal = stats.approved + stats.rejected;
  const passRate = completedTotal > 0 ? (stats.approved / completedTotal) * 100 : 0;
  const rejectRate = completedTotal > 0 ? (stats.rejected / completedTotal) * 100 : 0;

  // 格式化时间
  const formatTime = (timeStr: string) => {
    const date = new Date(timeStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    if (days < 7) return `${days}天前`;
    return date.toLocaleDateString();
  };

  if (loading && stats.total === 0) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
        <div style={{ marginTop: 16, color: '#999' }}>加载中...</div>
      </div>
    );
  }

  return (
    <div>
      <Row justify="space-between" align="middle" style={{ marginBottom: 20 }}>
        <Col>
          <Title level={4} style={{ margin: 0 }}>数据概览</Title>
        </Col>
        <Col>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/apply')}>
              发起审核
            </Button>
            <Button icon={<UnorderedListOutlined />} onClick={() => navigate('/list')}>
              审核列表
            </Button>
          </Space>
        </Col>
      </Row>
      
      {/* 今日数据速览 */}
      <Card 
        size="small" 
        style={{ marginBottom: 16, background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' }}
        bodyStyle={{ padding: '16px 24px' }}
      >
        <Row gutter={24} align="middle">
          <Col>
            <CalendarOutlined style={{ fontSize: 32, color: 'rgba(255,255,255,0.8)' }} />
          </Col>
          <Col flex={1}>
            <Text style={{ color: 'rgba(255,255,255,0.8)', fontSize: 13 }}>今日数据</Text>
            <div style={{ color: '#fff', fontSize: 13, marginTop: 4 }}>
              新增 <Text strong style={{ color: '#fff', fontSize: 18 }}>{stats.todayTotal}</Text> 条 · 
              待处理 <Text strong style={{ color: '#ffd666', fontSize: 18 }}>{stats.todayPending}</Text> 条 · 
              已通过 <Text strong style={{ color: '#95de64', fontSize: 18 }}>{stats.todayApproved}</Text> 条 · 
              已驳回 <Text strong style={{ color: '#ff7875', fontSize: 18 }}>{stats.todayRejected}</Text> 条
            </div>
          </Col>
          <Col>
            <Tooltip title="待处理任务需要尽快审核">
              {stats.todayPending > 0 ? (
                <Tag color="warning" icon={<ThunderboltOutlined />}>需处理</Tag>
              ) : (
                <Tag color="success" icon={<CheckCircleOutlined />}>已完成</Tag>
              )}
            </Tooltip>
          </Col>
        </Row>
      </Card>

      {/* 核心指标 */}
      <Row gutter={[12, 12]}>
        <Col xs={12} sm={8} md={4}>
          <Card size="small" hoverable>
            <Statistic 
              title="累计总量" 
              value={stats.total} 
              prefix={<FileTextOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} md={4}>
          <Card size="small" hoverable style={{ borderLeft: '3px solid #faad14' }}>
            <Statistic 
              title="待审核" 
              value={stats.pending} 
              prefix={<ClockCircleOutlined style={{ color: '#faad14' }} />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} md={4}>
          <Card size="small" hoverable>
            <Statistic 
              title="审核中" 
              value={stats.processing} 
              prefix={<SyncOutlined spin style={{ color: '#1890ff' }} />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} md={4}>
          <Card size="small" hoverable style={{ borderLeft: '3px solid #52c41a' }}>
            <Statistic 
              title="已通过" 
              value={stats.approved} 
              prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} md={4}>
          <Card size="small" hoverable style={{ borderLeft: '3px solid #ff4d4f' }}>
            <Statistic 
              title="已驳回" 
              value={stats.rejected} 
              prefix={<CloseCircleOutlined style={{ color: '#ff4d4f' }} />}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} md={4}>
          <Card size="small" hoverable>
            <Statistic 
              title="已撤销" 
              value={stats.cancelled} 
              prefix={<StopOutlined style={{ color: '#d9d9d9' }} />}
              valueStyle={{ color: '#999' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {/* 审核效率 */}
        <Col xs={24} md={8}>
          <Card title={<><FieldTimeOutlined /> 审核效率</>} size="small" style={{ height: '100%' }}>
            <Row gutter={16}>
              <Col span={12}>
                <div style={{ textAlign: 'center' }}>
                  <Progress 
                    type="circle" 
                    percent={Number(passRate.toFixed(1))} 
                    strokeColor="#52c41a"
                    size={80}
                  />
                  <div style={{ marginTop: 8, color: '#666', fontSize: 13 }}>通过率</div>
                </div>
              </Col>
              <Col span={12}>
                <div style={{ textAlign: 'center' }}>
                  <Progress 
                    type="circle" 
                    percent={Number(rejectRate.toFixed(1))} 
                    strokeColor="#ff4d4f"
                    size={80}
                  />
                  <div style={{ marginTop: 8, color: '#666', fontSize: 13 }}>驳回率</div>
                </div>
              </Col>
            </Row>
            <div style={{ marginTop: 16, padding: '12px', background: '#f5f5f5', borderRadius: 6, fontSize: 12, color: '#666' }}>
              <div>📊 已完成审核: <Text strong>{completedTotal}</Text> 条</div>
              <div style={{ marginTop: 4 }}>📈 待处理积压: <Text strong style={{ color: stats.pending > 10 ? '#ff4d4f' : '#52c41a' }}>{stats.pending}</Text> 条</div>
            </div>
          </Card>
        </Col>

        {/* 业务类型分布 */}
        <Col xs={24} md={8}>
          <Card title="业务类型分布" size="small" style={{ height: '100%' }}>
            {Object.keys(stats.byType).length === 0 ? (
              <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
              <div>
                {Object.entries(stats.byType).map(([type, count]) => {
                  const typeNum = Number(type);
                  const config = bizTypeConfig[typeNum] || { icon: '📄', color: '#666', bg: '#f5f5f5' };
                  const typeName = BusinessTypeMap[typeNum] || `类型${type}`;
                  const percent = stats.total > 0 ? ((count as number) / stats.total) * 100 : 0;
                  
                  return (
                    <div key={type} style={{ 
                      display: 'flex', 
                      alignItems: 'center', 
                      padding: '10px 12px',
                      background: config.bg,
                      borderRadius: 8,
                      marginBottom: 8,
                      cursor: 'pointer',
                      transition: 'all 0.2s'
                    }}
                    onClick={() => navigate(`/list?business_type=${type}`)}
                    >
                      <Avatar style={{ background: config.color, marginRight: 12 }} size={36}>
                        {config.icon}
                      </Avatar>
                      <div style={{ flex: 1 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                          <Text strong>{typeName}</Text>
                          <Text type="secondary">{count}条 ({percent.toFixed(0)}%)</Text>
                        </div>
                        <Progress 
                          percent={percent} 
                          showInfo={false} 
                          strokeColor={config.color}
                          size="small"
                        />
                      </div>
                      <RightOutlined style={{ color: '#999', marginLeft: 8 }} />
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </Col>

        {/* 最近审核记录 */}
        <Col xs={24} md={8}>
          <Card 
            title="最近审核记录" 
            size="small"
            style={{ height: '100%' }}
            extra={<Button type="link" size="small" onClick={() => navigate('/list')}>查看全部</Button>}
          >
            {recentList.length === 0 ? (
              <Empty description="暂无记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
              <Timeline
                items={recentList.map(item => {
                  const typeConfig = bizTypeConfig[item.business_type] || { icon: '📄', color: '#666' };
                  const status = statusConfig[item.audit_status] || { text: '未知', color: 'default' };
                  return {
                    color: status.color === 'warning' ? 'orange' : 
                           status.color === 'success' ? 'green' : 
                           status.color === 'error' ? 'red' : 
                           status.color === 'processing' ? 'blue' : 'gray',
                    children: (
                      <div 
                        style={{ cursor: 'pointer' }} 
                        onClick={() => navigate(`/list?id=${item.id}`)}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space size={4}>
                            <span>{typeConfig.icon}</span>
                            <Text strong style={{ fontSize: 13 }}>
                              {BusinessTypeMap[item.business_type] || '未知类型'}
                            </Text>
                            <Tag color={status.color} style={{ marginLeft: 4 }}>{status.text}</Tag>
                          </Space>
                        </div>
                        <div style={{ fontSize: 12, color: '#999', marginTop: 2 }}>
                          ID: {item.business_id} · {formatTime(item.created_at)}
                        </div>
                      </div>
                    )
                  };
                })}
              />
            )}
          </Card>
        </Col>
      </Row>

      {/* 快捷提示 */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Row gutter={24} align="middle">
          <Col flex={1}>
            <Space split={<span style={{ color: '#d9d9d9' }}>|</span>}>
              <Text type="secondary">💡 小提示</Text>
              <Text type="secondary">点击业务类型可快速筛选</Text>
              <Text type="secondary">数据每30秒自动刷新</Text>
              {stats.pending > 0 && (
                <Text type="warning">⚠️ 有 {stats.pending} 条待审核任务</Text>
              )}
            </Space>
          </Col>
          <Col>
            <Text type="secondary" style={{ fontSize: 12 }}>
              最后更新: {new Date().toLocaleTimeString()}
            </Text>
          </Col>
        </Row>
      </Card>
    </div>
  );
};

export default AuditDashboard;

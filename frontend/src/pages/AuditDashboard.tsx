import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Empty, Spin, Typography, Tag, Button, Timeline, theme } from 'antd';
import {
  CheckCircleFilled, CloseCircleFilled,
  ClockCircleFilled,
  PlusOutlined, UnorderedListOutlined,
  CalendarOutlined,
  ThunderboltFilled, RiseOutlined,
  PieChartOutlined
} from '@ant-design/icons';
import { queryAuditList, BusinessTypeMap } from '../api';
import { useNavigate } from 'react-router-dom';
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer,
  PieChart, Pie, Cell, Legend
} from 'recharts';

const { Title, Text } = Typography;

// 业务类型图标和颜色配置
const bizTypeConfig: Record<number, { icon: string; color: string; bg: string; borderColor: string }> = {
  1: { icon: '🎫', color: '#60a5fa', bg: 'rgba(59, 130, 246, 0.1)', borderColor: 'rgba(59, 130, 246, 0.2)' }, // Blue
  2: { icon: '🏨', color: '#34d399', bg: 'rgba(16, 185, 129, 0.1)', borderColor: 'rgba(16, 185, 129, 0.2)' }, // Green
  3: { icon: '🏢', color: '#a78bfa', bg: 'rgba(139, 92, 246, 0.1)', borderColor: 'rgba(139, 92, 246, 0.2)' }, // Purple
  4: { icon: '✈️', color: '#fbbf24', bg: 'rgba(245, 158, 11, 0.1)', borderColor: 'rgba(245, 158, 11, 0.2)' }, // Amber
  5: { icon: '🎡', color: '#f472b6', bg: 'rgba(236, 72, 153, 0.1)', borderColor: 'rgba(236, 72, 153, 0.2)' }, // Pink
};

const AuditDashboard: React.FC = () => {
  const navigate = useNavigate();
  const { token } = theme.useToken();
  const [loading, setLoading] = useState(true);
  const [recentList, setRecentList] = useState<any[]>([]);
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    processing: 0,
    approved: 0,
    rejected: 0,
    cancelled: 0,
    byType: [] as { name: string; value: number; type: number }[],
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

      // 为了演示效果，只拉取前100条
      const res = await queryAuditList({ page, page_size: pageSize });
      allRecords = res.list || [];

      // 获取今天的日期范围
      const today = new Date();
      today.setHours(0, 0, 0, 0);

      const newStats = {
        total: res.total, // 使用API返回的总数
        pending: 0,
        processing: 0,
        approved: 0,
        rejected: 0,
        cancelled: 0,
        byType: [] as { name: string; value: number; type: number }[],
        todayTotal: 0,
        todayPending: 0,
        todayApproved: 0,
        todayRejected: 0,
      };

      const typeMap: Record<number, number> = {};

      allRecords.forEach(item => {
        const createTime = new Date(item.submit_time);
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

        if (!typeMap[item.business_type]) {
          typeMap[item.business_type] = 0;
        }
        typeMap[item.business_type]++;
      });

      // 转换类型数据用于图表
      newStats.byType = Object.entries(typeMap).map(([type, value]) => ({
        name: BusinessTypeMap[Number(type)] || `Unknown ${type}`,
        value,
        type: Number(type)
      })).sort((a, b) => b.value - a.value);

      setStats(newStats);
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

  // Mock Trend Data (因为API不支持历史趋势)
  const trendData = [
    { name: 'Mon', value: 40 },
    { name: 'Tue', value: 30 },
    { name: 'Wed', value: 55 },
    { name: 'Thu', value: 80 },
    { name: 'Fri', value: 65 },
    { name: 'Sat', value: 45 },
    { name: 'Sun', value: stats.todayTotal > 0 ? stats.todayTotal : 60 },
  ];

  const pieColors = ['#60a5fa', '#34d399', '#a78bfa', '#fbbf24', '#f472b6'];

  // 格式化时间
  const formatTime = (timeStr: string) => {
    if (!timeStr) return '-';
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
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 1600, margin: '0 auto', color: token.colorText }}>
      {/* 顶部 Header 区域 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>数据概览</Title>
          <Text type="secondary">欢迎回来，今日已处理 {stats.todayApproved + stats.todayRejected} 条审核任务</Text>
        </div>
        <div style={{ display: 'flex', gap: 12 }}>
          <Button icon={<UnorderedListOutlined />} onClick={() => navigate('/')}>审核列表</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/apply')}>发起申请</Button>
        </div>
      </div>

      {/* 核心指标卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} className="glass-card" style={{ height: '100%' }}>
            <Statistic
              title={<span style={{ color: token.colorTextSecondary }}>待处理任务</span>}
              value={stats.pending}
              prefix={<ClockCircleFilled style={{ color: '#fbbf24' }} />}
              suffix={<Tag color="warning" style={{ marginLeft: 8, background: 'rgba(245, 158, 11, 0.2)', border: 'none' }}>{stats.todayPending} 新增</Tag>}
              valueStyle={{ fontWeight: 600, color: token.colorText }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} className="glass-card" style={{ height: '100%' }}>
            <Statistic
              title={<span style={{ color: token.colorTextSecondary }}>审核通过</span>}
              value={stats.approved}
              prefix={<CheckCircleFilled style={{ color: '#34d399' }} />}
              valueStyle={{ fontWeight: 600, color: token.colorText }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} className="glass-card" style={{ height: '100%' }}>
            <Statistic
              title={<span style={{ color: token.colorTextSecondary }}>审核驳回</span>}
              value={stats.rejected}
              prefix={<CloseCircleFilled style={{ color: '#f87171' }} />}
              valueStyle={{ fontWeight: 600, color: token.colorText }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} className="glass-card" style={{ height: '100%' }}>
            <Statistic
              title={<span style={{ color: token.colorTextSecondary }}>累计申请总数</span>}
              value={stats.total}
              prefix={<ThunderboltFilled style={{ color: '#60a5fa' }} />}
              valueStyle={{ fontWeight: 600, color: token.colorText }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[24, 24]}>
        {/* 左侧图表区域 */}
        <Col xs={24} lg={16}>
          <Card
            title={<><RiseOutlined style={{ marginRight: 8 }} />近7日审核趋势</>}
            bordered={false}
            className="glass-card"
            style={{ marginBottom: 24 }}
          >
            <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={trendData}>
                  <defs>
                    <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#60a5fa" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#60a5fa" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={token.colorBorderSecondary} />
                  <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fill: token.colorTextSecondary }} />
                  <YAxis axisLine={false} tickLine={false} tick={{ fill: token.colorTextSecondary }} />
                  <RechartsTooltip
                    contentStyle={{
                      backgroundColor: token.colorBgElevated,
                      borderColor: token.colorBorder,
                      borderRadius: 8,
                      boxShadow: token.boxShadowSecondary,
                      color: token.colorText
                    }}
                    itemStyle={{ color: token.colorText }}
                    labelStyle={{ color: token.colorTextSecondary }}
                  />
                  <Area type="monotone" dataKey="value" stroke="#60a5fa" strokeWidth={3} fillOpacity={1} fill="url(#colorValue)" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </Card>

          <Card
            title={<><UnorderedListOutlined style={{ marginRight: 8 }} />最近活动</>}
            bordered={false}
            className="glass-card"
            extra={<Button type="link" onClick={() => navigate('/')}>查看全部</Button>}
          >
            {recentList.length === 0 ? (
              <Empty description="暂无记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
              <Timeline
                items={recentList.map(item => {
                  const typeConfig = bizTypeConfig[item.business_type] || { icon: '📄', color: token.colorTextSecondary, bg: 'transparent', borderColor: token.colorBorder };
                  return {
                    color: item.audit_status === 1 ? 'gold' : item.audit_status === 3 ? 'green' : item.audit_status === 4 ? 'red' : 'gray',
                    children: (
                      <div
                        style={{ cursor: 'pointer', paddingBottom: 12 }}
                        onClick={() => navigate(`/?id=${item.audit_id}`)}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Tag style={{ border: `1px solid ${typeConfig.borderColor}`, background: typeConfig.bg, color: typeConfig.color }}>
                            {typeConfig.icon} {BusinessTypeMap[item.business_type]}
                          </Tag>
                          <Text type="secondary" style={{ fontSize: 12 }}>{formatTime(item.submit_time)}</Text>
                        </div>
                        <div style={{ marginTop: 4 }}>
                          <Text style={{ fontSize: 13 }}>业务ID: {item.business_id}</Text>
                        </div>
                      </div>
                    )
                  };
                })}
              />
            )}
          </Card>
        </Col>

        {/* 右侧分布区域 */}
        <Col xs={24} lg={8}>
          <Card
            title={<><PieChartOutlined style={{ marginRight: 8 }} />业务分布</>}
            bordered={false}
            className="glass-card"
            style={{ marginBottom: 24 }}
          >
            <div style={{ height: 260 }}>
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={stats.byType}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="value"
                    stroke="none"
                  >
                    {stats.byType.map((_entry, index) => (
                      <Cell key={`cell-${index}`} fill={pieColors[index % pieColors.length]} />
                    ))}
                  </Pie>
                  <RechartsTooltip contentStyle={{
                    backgroundColor: token.colorBgElevated,
                    borderColor: token.colorBorder,
                    borderRadius: 8,
                    color: token.colorText
                  }}
                    itemStyle={{ color: token.colorText }}
                  />
                  <Legend verticalAlign="bottom" height={36} formatter={(value) => <span style={{ color: token.colorTextSecondary }}>{value}</span>} />
                </PieChart>
              </ResponsiveContainer>
            </div>

            <div style={{ marginTop: 16 }}>
              {stats.byType.slice(0, 3).map((item, idx) => (
                <div key={idx} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8, fontSize: 13 }}>
                  <span>
                    <span style={{
                      display: 'inline-block', width: 8, height: 8, borderRadius: '50%',
                      backgroundColor: pieColors[idx % pieColors.length], marginRight: 8
                    }} />
                    <Text>{item.name}</Text>
                  </span>
                  <Text strong>{item.value}</Text>
                </div>
              ))}
            </div>
          </Card>

          {/* 待处理总数卡片 - 修复布局错误 */}
          <Card
            bordered={false}
            className="glass-card"
            style={{
              background: 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)', // Darker Blue
              borderRadius: 12,
              border: 'none',
              overflow: 'hidden',
              position: 'relative'
            }}
            bodyStyle={{ padding: 24 }}
          >
            <div style={{ position: 'relative', zIndex: 1, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14, color: 'rgba(255,255,255,0.8)', marginBottom: 8 }}>待处理总数</div>
                <div style={{ fontSize: 36, fontWeight: 700, color: 'white', lineHeight: 1 }}>{stats.pending}</div>
                <div style={{ fontSize: 13, color: 'rgba(255,255,255,0.7)', marginTop: 8 }}>
                  较昨日 <span style={{ color: 'white', fontWeight: 500 }}>+{stats.todayPending}</span>
                </div>
              </div>
              <CalendarOutlined style={{ fontSize: 64, color: 'rgba(255,255,255,0.1)', transform: 'rotate(-15deg)', marginRight: -10, marginBottom: -10 }} />
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default AuditDashboard;

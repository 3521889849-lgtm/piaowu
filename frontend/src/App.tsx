import React from 'react';
import { Layout, Menu, Button, Avatar, Dropdown, Typography, theme, ConfigProvider } from 'antd';
import {
  UnorderedListOutlined,
  PlusCircleOutlined,
  DashboardOutlined,
  UserOutlined,
  LogoutOutlined,
  BellOutlined
} from '@ant-design/icons';
import { BrowserRouter, Routes, Route, Link, Navigate, useLocation } from 'react-router-dom';
import { themeConfig } from './styles/theme';
import AuditList from './pages/AuditList';
import ApplyAudit from './pages/ApplyAudit';
import AuditDashboard from './pages/AuditDashboard';
import Login from './pages/Login';
import Error500 from './pages/Error500';
import { AuthProvider, useAuth } from './context/AuthContext';

const { Header, Content, Sider } = Layout;
const { Text } = Typography;

const RequireAuth: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  return <>{children}</>;
};

const MainLayout: React.FC = () => {
  const { token } = theme.useToken();
  const { logout, user } = useAuth();
  const location = useLocation();

  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: <Link to="/dashboard">数据概览</Link> },
    { key: '/', icon: <UnorderedListOutlined />, label: <Link to="/">审核任务</Link> },
    { key: '/apply', icon: <PlusCircleOutlined />, label: <Link to="/apply">提交申请</Link> },
  ];

  const userMenu = {
    items: [
      { key: 'profile', icon: <UserOutlined />, label: '个人中心' },
      { type: 'divider' as const },
      { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: logout }
    ]
  };

  return (
    <Layout style={{ minHeight: '100vh', background: token.colorBgLayout }}>
      <Sider
        width={240}
        style={{
          background: token.colorBgContainer,
          borderRight: `1px solid ${token.colorBorderSecondary}`,
          position: 'fixed',
          height: '100vh',
          zIndex: 10,
          left: 0,
          top: 0,
        }}
        breakpoint="lg"
        collapsedWidth={0}
      >
        <div style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderBottom: `1px solid ${token.colorBorderSecondary}`
        }}>
          <div style={{
            fontSize: 20,
            fontWeight: 700,
            background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            display: 'flex',
            alignItems: 'center',
            gap: 8
          }}>
            <span style={{ fontSize: 24 }}>🎫</span> 天极票务
          </div>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          style={{ borderRight: 0, padding: 16, background: 'transparent' }}
          items-style={{ borderRadius: 8 }}
        />

        <div style={{ position: 'absolute', bottom: 24, left: 0, width: '100%', padding: '0 24px', textAlign: 'center' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>v1.0.0 Enterprise</Text>
        </div>
      </Sider>

      <Layout style={{ marginLeft: 240, background: 'transparent' }}>
        <Header style={{
          padding: '0 24px',
          background: token.colorBgContainer, // Use token
          opacity: 0.9, // Slight transparency
          backdropFilter: 'blur(8px)',
          display: 'flex',
          justifyContent: 'flex-end',
          alignItems: 'center',
          position: 'sticky',
          top: 0,
          zIndex: 9,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          height: 64
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Button type="text" shape="circle" icon={<BellOutlined />} />

            <Dropdown menu={userMenu} placement="bottomRight" arrow>
              <div style={{
                display: 'flex',
                alignItems: 'center',
                cursor: 'pointer',
                padding: '4px 8px',
                borderRadius: 20,
                background: token.colorBgElevated,
                border: `1px solid ${token.colorBorderSecondary}`,
                transition: 'all 0.3s'
              }}
                className="user-dropdown-trigger"
              >
                <Avatar size="small" style={{ backgroundColor: token.colorPrimary, marginRight: 8, verticalAlign: 'middle' }}>
                  {user?.charAt(0).toUpperCase()}
                </Avatar>
                <Text strong style={{ marginRight: 4 }}>{user}</Text>
              </div>
            </Dropdown>
          </div>
        </Header>

        <Content style={{ margin: '24px 24px', overflow: 'initial' }}>
          <Routes>
            <Route path="/" element={<AuditList />} />
            <Route path="/dashboard" element={<AuditDashboard />} />
            <Route path="/apply" element={<ApplyAudit />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  );
};

const App: React.FC = () => {
  return (
    <ConfigProvider theme={themeConfig}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/500" element={<Error500 />} />
            <Route path="/*" element={<RequireAuth><MainLayout /></RequireAuth>} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ConfigProvider>
  );
};

export default App;

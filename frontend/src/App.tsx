import React from 'react';
import { Layout, Menu, Button, Avatar, Dropdown, Typography, theme } from 'antd';
import {
  UnorderedListOutlined,
  PlusCircleOutlined,
  DashboardOutlined,
  UserOutlined,
  LogoutOutlined
} from '@ant-design/icons';
import { BrowserRouter, Routes, Route, Link, Navigate, useLocation } from 'react-router-dom';
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
  const { token: { colorBgContainer, borderRadiusLG } } = theme.useToken();
  const { logout, user } = useAuth();
  const location = useLocation();

  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: <Link to="/dashboard">数据概览</Link> },
    { key: '/', icon: <UnorderedListOutlined />, label: <Link to="/">审核任务</Link> },
    { key: '/apply', icon: <PlusCircleOutlined />, label: <Link to="/apply">提交申请</Link> },
  ];

  const userMenu = {
    items: [
      { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: logout }
    ]
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider 
        width={200} 
        style={{ background: '#001529' }}
        breakpoint="lg" 
        collapsedWidth={0}
      >
        <div style={{ 
          height: 48, 
          margin: 16, 
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center'
        }}>
          <Text style={{ color: '#fff', fontSize: 16, fontWeight: 600 }}>
            🎫 天极票务
          </Text>
        </div>
        <Menu 
          theme="dark" 
          mode="inline" 
          selectedKeys={[location.pathname]}
          items={menuItems}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <Layout>
        <Header style={{ 
          padding: '0 24px', 
          background: colorBgContainer, 
          display: 'flex', 
          justifyContent: 'flex-end', 
          alignItems: 'center',
          borderBottom: '1px solid #f0f0f0'
        }}>
          <Dropdown menu={userMenu} placement="bottomRight">
            <Button type="text" style={{ height: 'auto', padding: '4px 8px' }}>
              <Avatar size="small" icon={<UserOutlined />} style={{ marginRight: 8 }} />
              <Text>{user}</Text>
            </Button>
          </Dropdown>
        </Header>
        <Content style={{ margin: 16, overflow: 'auto' }}>
          <div style={{ 
            padding: 20, 
            minHeight: '100%',
            background: colorBgContainer, 
            borderRadius: borderRadiusLG 
          }}>
            <Routes>
              <Route path="/" element={<AuditList />} />
              <Route path="/dashboard" element={<AuditDashboard />} />
              <Route path="/apply" element={<ApplyAudit />} />
            </Routes>
          </div>
        </Content>
      </Layout>
    </Layout>
  );
};

const App: React.FC = () => (
  <AuthProvider>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/500" element={<Error500 />} />
        <Route path="/*" element={<RequireAuth><MainLayout /></RequireAuth>} />
      </Routes>
    </BrowserRouter>
  </AuthProvider>
);

export default App;

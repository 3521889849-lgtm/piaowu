import React from 'react';
import { Form, Input, Button, Card, message } from 'antd';
import { useAuth } from '../context/AuthContext';
import { useNavigate } from 'react-router-dom';

const Login: React.FC = () => {
  const { login } = useAuth();
  const navigate = useNavigate();

  const onFinish = (values: any) => {
    if (values.username === 'admin' && values.password === '123456') {
      login(values.username);
      message.success('登录成功');
      navigate('/');
    } else {
      message.error('用户名或密码错误 (请尝试 admin/123456)');
    }
  };

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', background: '#f0f2f5' }}>
      <Card title="登录审核管理系统" style={{ width: 350 }}>
        <Form onFinish={onFinish} layout="vertical">
          <Form.Item 
            name="username" 
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="请输入用户名 (admin)" />
          </Form.Item>
          <Form.Item 
            name="password" 
            label="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder="请输入密码 (123456)" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block size="large">
              进入系统
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default Login;

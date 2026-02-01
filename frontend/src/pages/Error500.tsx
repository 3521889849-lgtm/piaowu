import React from 'react';
import { Result, Button, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';

const { Paragraph, Text } = Typography;

const Error500: React.FC = () => {
  const navigate = useNavigate();

  return (
    <Result
      status="500"
      title="500"
      subTitle="服务器内部错误，请稍后再试"
      extra={
        <Button type="primary" onClick={() => navigate('/')}>
          返回首页
        </Button>
      }
    >
      <div className="desc">
        <Paragraph>
          <Text
            strong
            style={{
              fontSize: 16,
            }}
          >
            可能的原因提示：
          </Text>
        </Paragraph>
        <Paragraph>
          <ul>
            <li>服务器正在维护中</li>
            <li>系统负载过高</li>
            <li>后端服务连接失败</li>
          </ul>
        </Paragraph>
        <Paragraph>
          <Text
            strong
            style={{
              fontSize: 16,
            }}
          >
            建议的解决方案：
          </Text>
        </Paragraph>
        <Paragraph>
          <ul>
            <li>请稍后刷新页面重试</li>
            <li>检查您的网络连接</li>
            <li>如果问题持续存在，请联系技术支持</li>
          </ul>
        </Paragraph>
        <Paragraph>
          <Text strong>联系技术支持：</Text> support@example.com
        </Paragraph>
      </div>
    </Result>
  );
};

export default Error500;

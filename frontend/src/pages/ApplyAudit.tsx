import React, { useState } from 'react';
import { Form, Input, Select, Button, message, Card, DatePicker, InputNumber, Row, Col, Divider, Typography } from 'antd';
import { SendOutlined, FileAddOutlined } from '@ant-design/icons';
import { submitAudit, BusinessTypeMap, TicketTypeMap, type SubmitAuditRequest } from '../api';
import dayjs from 'dayjs';

const { Text } = Typography;

const ApplyAudit: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [businessType, setBusinessType] = useState<number>();
  const [form] = Form.useForm();

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const request: SubmitAuditRequest = {
        business_type: values.business_type,
        business_id: values.business_id,
        apply_reason: values.apply_reason,
      };

      if (values.business_type === 1) {
        request.ticket_order = {
          ticket_order_id: values.ticket_order_id,
          ticket_type: values.ticket_type,
          departure_station: values.departure_station,
          arrival_station: values.arrival_station,
          departure_time: values.departure_time.format('YYYY-MM-DDTHH:mm:ss') + 'Z',
          passenger_name: values.passenger_name,
          passenger_id_card: values.passenger_id_card,
          order_amount: values.order_amount,
        };
      } else if (values.business_type === 2) {
        request.hotel_order = {
          hotel_id: values.hotel_id,
          hotel_name: values.hotel_name,
          hotel_address: values.hotel_address,
          room_type: values.room_type,
          check_in_time: values.check_in_time?.format('YYYY-MM-DDTHH:mm:ss') + 'Z',
          check_out_time: values.check_out_time?.format('YYYY-MM-DDTHH:mm:ss') + 'Z',
          guest_name: values.guest_name,
          guest_id_card: values.guest_id_card,
          order_amount: values.order_amount,
        };
      } else if (values.business_type === 3) {
        request.hotel_settle = {
          hotel_id: values.hotel_id,
          hotel_name: values.hotel_name,
          hotel_address: values.hotel_address,
          business_license: values.business_license,
        };
      } else if (values.business_type === 4) {
        request.flight_order = {
          flight_order_id: values.flight_order_id,
          flight_type: values.flight_type,
          flight_no: values.flight_no,
          airline: values.airline,
          departure_airport: values.departure_airport,
          arrival_airport: values.arrival_airport,
          departure_time: values.departure_time.format('YYYY-MM-DDTHH:mm:ss') + 'Z',
          arrival_time: values.arrival_time.format('YYYY-MM-DDTHH:mm:ss') + 'Z',
          cabin_class: values.cabin_class,
          passenger_name: values.passenger_name,
          passenger_id_card: values.passenger_id_card,
          order_amount: values.order_amount,
        };
      } else if (values.business_type === 5) {
        request.scenic_order = {
          scenic_order_id: values.scenic_order_id,
          ticket_type: values.scenic_ticket_type,
          scenic_name: values.scenic_name,
          scenic_address: values.scenic_address,
          ticket_name: values.ticket_name,
          visit_date: values.visit_date.format('YYYY-MM-DDTHH:mm:ss') + 'Z',
          ticket_quantity: values.ticket_quantity,
          unit_price: values.unit_price,
          order_amount: values.order_amount,
          contact_name: values.contact_name,
          contact_phone: values.contact_phone,
        };
      }

      const res = await submitAudit(request);
      message.success(`提交成功！审核ID: ${res.audit_id}`);
      form.resetFields();
      setBusinessType(undefined);
    } catch (e) {
      console.error('提交失败:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleTypeChange = (type: number) => {
    setBusinessType(type);
    form.resetFields([
      'ticket_order_id', 'ticket_type', 'departure_station', 'arrival_station',
      'departure_time', 'passenger_name', 'passenger_id_card',
      'hotel_id', 'hotel_name', 'hotel_address', 'room_type',
      'check_in_time', 'check_out_time', 'guest_name', 'guest_id_card',
      'business_license', 'order_amount',
      'flight_order_id', 'flight_type', 'flight_no', 'airline',
      'departure_airport', 'arrival_airport', 'arrival_time', 'cabin_class',
      'scenic_order_id', 'scenic_ticket_type', 'scenic_name', 'scenic_address',
      'ticket_name', 'visit_date', 'ticket_quantity', 'unit_price',
      'contact_name', 'contact_phone'
    ]);
  };

  const loadTemplate = () => {
    const templates: Record<number, any> = {
      1: {
        business_id: 10001, apply_reason: '车票退款审核',
        ticket_order_id: 10001, ticket_type: 1,
        departure_station: '北京南站', arrival_station: '上海虹桥站',
        departure_time: dayjs().add(2, 'day'),
        passenger_name: '张三', passenger_id_card: '110101199001011234',
        order_amount: 553.5,
      },
      2: {
        business_id: 20001, apply_reason: '酒店退款审核',
        hotel_id: 5001, hotel_name: '北京希尔顿酒店',
        hotel_address: '北京市朝阳区建国路1号', room_type: '豪华大床房',
        check_in_time: dayjs().add(2, 'day').hour(14),
        check_out_time: dayjs().add(3, 'day').hour(12),
        guest_name: '李四', guest_id_card: '110101199002021234',
        order_amount: 888.0,
      },
      3: {
        business_id: 30001, apply_reason: '新酒店入驻申请',
        hotel_id: 6001, hotel_name: '上海外滩酒店',
        hotel_address: '上海市黄浦区中山东一路500号',
        business_license: '91310000MA1234567X',
      },
      4: {
        business_id: 40001, apply_reason: '机票退款审核',
        flight_order_id: 40001, flight_type: 1,
        flight_no: 'CA1234', airline: '中国国际航空',
        departure_airport: '北京首都国际机场', arrival_airport: '上海浦东国际机场',
        departure_time: dayjs().add(3, 'day').hour(8).minute(30),
        arrival_time: dayjs().add(3, 'day').hour(10).minute(45),
        cabin_class: '经济舱',
        passenger_name: '王五', passenger_id_card: '110101199003031234',
        order_amount: 1280.0,
      },
      5: {
        business_id: 50001, apply_reason: '门票退款审核',
        scenic_order_id: 50001, scenic_ticket_type: 1,
        scenic_name: '故宫博物院', scenic_address: '北京市东城区景山前街4号',
        ticket_name: '成人票', visit_date: dayjs().add(5, 'day'),
        ticket_quantity: 2, unit_price: 60.0, order_amount: 120.0,
        contact_name: '赵六', contact_phone: '13800138000',
      }
    };
    if (businessType && templates[businessType]) {
      form.setFieldsValue(templates[businessType]);
    }
  };

  return (
    <Card
      title={<><FileAddOutlined /> 提交审核申请</>}
      extra={businessType && <Button type="link" size="small" onClick={loadTemplate}>填充示例</Button>}
      bordered={false}
      className="glass-card"
    >
      <Form form={form} layout="vertical" onFinish={onFinish} style={{ maxWidth: 700 }}>
        {/* 基础信息 */}
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="business_type" label="业务类型" rules={[{ required: true, message: '请选择' }]}>
              <Select onChange={handleTypeChange} placeholder="选择业务类型">
                {Object.entries(BusinessTypeMap).map(([k, v]) => (
                  <Select.Option key={k} value={Number(k)}>{v}</Select.Option>
                ))}
              </Select>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="business_id" label="业务ID" rules={[{ required: true, message: '请输入' }]}>
              <InputNumber style={{ width: '100%' }} placeholder="业务ID" min={1} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item name="apply_reason" label="申请原因" rules={[{ required: true, message: '请输入' }]}>
          <Input.TextArea rows={2} placeholder="申请原因" maxLength={255} showCount />
        </Form.Item>

        {/* 车票订单 */}
        {businessType === 1 && (
          <>
            <Divider><Text type="secondary">🎫 车票信息</Text></Divider>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="ticket_order_id" label="订单ID" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} placeholder="订单ID" min={1} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="ticket_type" label="车票类型" rules={[{ required: true }]}>
                  <Select placeholder="选择类型">
                    {Object.entries(TicketTypeMap).map(([k, v]) => (
                      <Select.Option key={k} value={Number(k)}>{v}</Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="departure_station" label="出发站" rules={[{ required: true }]}>
                  <Input placeholder="出发站" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="arrival_station" label="到达站" rules={[{ required: true }]}>
                  <Input placeholder="到达站" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="departure_time" label="发车时间" rules={[{ required: true }]}>
                  <DatePicker showTime style={{ width: '100%' }} format="YYYY-MM-DD HH:mm" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="order_amount" label="金额" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="passenger_name" label="乘客姓名" rules={[{ required: true }]}>
                  <Input placeholder="乘客姓名" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="passenger_id_card" label="身份证号" rules={[{ required: true }, { pattern: /^\d{17}[\dXx]$/, message: '格式错误' }]}>
                  <Input placeholder="身份证号" />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        {/* 酒店订单 */}
        {businessType === 2 && (
          <>
            <Divider><Text type="secondary">🏨 酒店订单</Text></Divider>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="hotel_id" label="酒店ID" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} placeholder="酒店ID" min={1} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="hotel_name" label="酒店名称" rules={[{ required: true }]}>
                  <Input placeholder="酒店名称" />
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item name="hotel_address" label="酒店地址" rules={[{ required: true }]}>
                  <Input placeholder="酒店地址" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="room_type" label="房间类型">
                  <Input placeholder="房间类型（选填）" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="order_amount" label="金额" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="check_in_time" label="入住时间">
                  <DatePicker showTime style={{ width: '100%' }} format="YYYY-MM-DD HH:mm" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="check_out_time" label="退房时间">
                  <DatePicker showTime style={{ width: '100%' }} format="YYYY-MM-DD HH:mm" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="guest_name" label="入住人">
                  <Input placeholder="入住人姓名（选填）" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="guest_id_card" label="身份证号" rules={[{ pattern: /^\d{17}[\dXx]$/, message: '格式错误' }]}>
                  <Input placeholder="身份证号（选填）" />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        {/* 酒店入驻 */}
        {businessType === 3 && (
          <>
            <Divider><Text type="secondary">🏢 酒店入驻</Text></Divider>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="hotel_id" label="酒店ID" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} placeholder="酒店ID" min={1} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="hotel_name" label="酒店名称" rules={[{ required: true }]}>
                  <Input placeholder="酒店名称" />
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item name="hotel_address" label="酒店地址" rules={[{ required: true }]}>
                  <Input placeholder="酒店地址" />
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item name="business_license" label="营业执照" rules={[{ required: true }]}>
                  <Input placeholder="营业执照编号" />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        {/* 机票订单 */}
        {businessType === 4 && (
          <>
            <Divider><Text type="secondary">✈️ 机票信息</Text></Divider>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="flight_order_id" label="订单ID" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} placeholder="订单ID" min={1} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="flight_type" label="航班类型" rules={[{ required: true }]}>
                  <Select placeholder="选择类型">
                    <Select.Option value={1}>国内航班</Select.Option>
                    <Select.Option value={2}>国际航班</Select.Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="flight_no" label="航班号" rules={[{ required: true }]}>
                  <Input placeholder="如：CA1234" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="airline" label="航空公司" rules={[{ required: true }]}>
                  <Input placeholder="航空公司" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="departure_airport" label="出发机场" rules={[{ required: true }]}>
                  <Input placeholder="出发机场" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="arrival_airport" label="到达机场" rules={[{ required: true }]}>
                  <Input placeholder="到达机场" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="departure_time" label="起飞时间" rules={[{ required: true }]}>
                  <DatePicker showTime style={{ width: '100%' }} format="YYYY-MM-DD HH:mm" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="arrival_time" label="降落时间" rules={[{ required: true }]}>
                  <DatePicker showTime style={{ width: '100%' }} format="YYYY-MM-DD HH:mm" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="cabin_class" label="舱位等级" rules={[{ required: true }]}>
                  <Select placeholder="选择舱位">
                    <Select.Option value="经济舱">经济舱</Select.Option>
                    <Select.Option value="商务舱">商务舱</Select.Option>
                    <Select.Option value="头等舱">头等舱</Select.Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="order_amount" label="金额" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="passenger_name" label="乘客姓名" rules={[{ required: true }]}>
                  <Input placeholder="乘客姓名" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="passenger_id_card" label="身份证号" rules={[{ required: true }, { pattern: /^\d{17}[\dXx]$/, message: '格式错误' }]}>
                  <Input placeholder="身份证号" />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        {/* 旅游门票 */}
        {businessType === 5 && (
          <>
            <Divider><Text type="secondary">🎡 门票信息</Text></Divider>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="scenic_order_id" label="订单ID" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} placeholder="订单ID" min={1} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="scenic_ticket_type" label="门票类型" rules={[{ required: true }]}>
                  <Select placeholder="选择类型">
                    <Select.Option value={1}>景区门票</Select.Option>
                    <Select.Option value={2}>演出票</Select.Option>
                    <Select.Option value={3}>展览票</Select.Option>
                    <Select.Option value={4}>游乐园票</Select.Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="scenic_name" label="景区/场馆名称" rules={[{ required: true }]}>
                  <Input placeholder="景区/场馆名称" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="ticket_name" label="门票名称" rules={[{ required: true }]}>
                  <Input placeholder="如：成人票、学生票" />
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item name="scenic_address" label="景区地址" rules={[{ required: true }]}>
                  <Input placeholder="景区地址" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="visit_date" label="游玩日期" rules={[{ required: true }]}>
                  <DatePicker style={{ width: '100%' }} format="YYYY-MM-DD" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="ticket_quantity" label="门票数量" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} placeholder="数量" min={1} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="unit_price" label="单价" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="order_amount" label="总金额" rules={[{ required: true }]}>
                  <InputNumber style={{ width: '100%' }} prefix="¥" min={0.01} precision={2} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="contact_name" label="联系人姓名" rules={[{ required: true }]}>
                  <Input placeholder="联系人姓名" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="contact_phone" label="联系电话" rules={[{ required: true }, { pattern: /^1[3-9]\d{9}$/, message: '手机号格式错误' }]}>
                  <Input placeholder="联系电话" />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        <Form.Item style={{ marginTop: 24 }}>
          <Button type="primary" htmlType="submit" loading={loading} icon={<SendOutlined />} size="large" disabled={!businessType}>
            提交申请
          </Button>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default ApplyAudit;

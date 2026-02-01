# 酒店审核接口文档

## 接口概述

本文档提供酒店审核相关的HTTP接口说明。外部系统直接将审核数据写入数据库，本接口仅提供审核查询和审核处理功能。

## 基础信息

- **Base URL**: `http://your-domain:8080/api/v1`
- **Content-Type**: `application/json`
- **字符编码**: `UTF-8`

## 数据流程说明

1. **外部系统提交审核**: 外部系统（如酒店订单系统）直接将审核数据写入 `audit_mains` 和 `audit_hotel_orders` 表
2. **审核员查询待审核任务**: 通过本接口查询待审核列表
3. **审核员处理审核**: 通过本接口进行审核通过或驳回操作
4. **外部系统查询审核结果**: 通过本接口查询审核状态和结果

## 接口列表

### 1. 查询审核详情

**接口地址**: `GET /audit/{audit_id}`

**接口描述**: 根据审核ID查询审核详情

**请求参数**:

| 参数 | 类型 | 位置 | 必填 | 说明 |
|------|------|------|------|------|
| audit_id | int | path | 是 | 审核ID |

**请求示例**:
```
GET /api/v1/audit/12345
```

**响应示例**:

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "audit_id": 12345,
    "business_type": 2,
    "business_id": 10001,
    "audit_status": 1,
    "audit_status_text": "待审核",
    "submit_user_id": 1001,
    "submit_user_name": "张三",
    "audit_user_id": 0,
    "audit_user_name": "",
    "audit_remark": "",
    "audit_time": null,
    "submit_time": "2024-01-20T10:30:00Z",
    "hotel_order": {
      "hotel_id": 5001,
      "hotel_name": "北京希尔顿酒店",
      "hotel_address": "北京市朝阳区建国路1号",
      "room_type": "豪华大床房",
      "check_in_time": "2024-01-20T14:00:00Z",
      "check_out_time": "2024-01-22T12:00:00Z",
      "guest_name": "张三",
      "guest_id_card": "1101**********1234",
      "order_amount": 2888.00,
      "apply_reason": "订单金额异常，需要人工审核"
    }
  }
}
```

---

### 2. 查询审核列表

**接口地址**: `GET /audit/list`

**接口描述**: 根据条件查询审核列表

**请求参数**:

| 参数 | 类型 | 位置 | 必填 | 说明 |
|------|------|------|------|------|
| business_type | int | query | 否 | 业务类型：1=车票订单，2=酒店订单，3=酒店入驻 |
| business_id | int | query | 否 | 业务ID |
| audit_status | int | query | 否 | 审核状态：1=待审核，2=审核中，3=通过，4=驳回，5=撤销 |
| page | int | query | 否 | 页码，默认1 |
| page_size | int | query | 否 | 每页数量，默认10，最大100 |

**请求示例**:
```
GET /api/v1/audit/list?business_type=2&audit_status=1&page=1&page_size=10
```

**响应示例**:

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "total": 25,
    "list": [
      {
        "audit_id": 12345,
        "business_type": 2,
        "business_id": 10001,
        "audit_status": 1,
        "audit_status_text": "待审核",
        "submit_user_id": 1001,
        "submit_user_name": "张三",
        "submit_time": "2024-01-20T10:30:00Z",
        "hotel_order": {
          "hotel_id": 5001,
          "hotel_name": "北京希尔顿酒店",
          "order_amount": 2888.00
        }
      }
    ]
  }
}
```

---

### 3. 处理审核（审核员操作）

**接口地址**: `POST /audit/process`

**接口描述**: 审核员处理审核，可以通过或驳回

**请求头**:
```
Content-Type: application/json
X-User-ID: 审核员ID
X-User-Name: 审核员名称
```

**请求参数**:

```json
{
  "audit_id": 12345,
  "action": "approve",
  "audit_remark": "审核通过，订单信息无误"
}
```

**参数说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| audit_id | int | 是 | 审核ID |
| action | string | 是 | 操作：approve=通过，reject=驳回 |
| audit_remark | string | 否 | 审核备注，最多500字符 |

**响应示例**:

```json
{
  "code": 0,
  "message": "审核处理成功"
}
```

---

## 审核状态说明

| 状态码 | 状态名称 | 说明 |
|--------|----------|------|
| 1 | 待审核 | 审核申请已提交，等待审核员处理 |
| 2 | 审核中 | 审核员已接单，正在审核 |
| 3 | 通过 | 审核通过 |
| 4 | 驳回 | 审核驳回 |
| 5 | 撤销 | 审核已撤销 |

## 业务类型说明

| 类型码 | 业务类型 | 说明 |
|--------|----------|------|
| 1 | 车票订单 | 车票订单审核 |
| 2 | 酒店订单 | 酒店订单审核 |
| 3 | 酒店入驻 | 酒店入驻审核 |

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 外部系统集成说明

### 数据库表结构

外部系统需要直接写入以下表：

#### 1. audit_mains（审核主表）

```sql
INSERT INTO audit_mains (
    business_type,
    business_id,
    audit_status,
    submit_user_id,
    submit_user_name,
    created_at,
    updated_at
) VALUES (
    2,                    -- 业务类型：2=酒店订单
    10001,                -- 业务ID（订单ID）
    1,                    -- 审核状态：1=待审核
    1001,                 -- 提交人ID
    '张三',               -- 提交人名称
    NOW(),                -- 创建时间
    NOW()                 -- 更新时间
);
```

#### 2. audit_hotel_orders（酒店审核明细表）

```sql
INSERT INTO audit_hotel_orders (
    audit_main_id,
    business_rel_id,
    hotel_id,
    hotel_name,
    hotel_address,
    room_type,
    check_in_time,
    check_out_time,
    guest_name,
    guest_id_card,
    order_amount,
    apply_reason,
    created_at,
    updated_at
) VALUES (
    12345,                           -- 审核主表ID（上一步插入返回的ID）
    10001,                           -- 关联业务ID
    5001,                            -- 酒店ID
    '北京希尔顿酒店',                -- 酒店名称
    '北京市朝阳区建国路1号',         -- 酒店地址
    '豪华大床房',                    -- 房间类型
    '2024-01-20 14:00:00',          -- 入住时间
    '2024-01-22 12:00:00',          -- 退房时间
    '张三',                          -- 入住人姓名
    '1101**********1234',           -- 入住人身份证号（脱敏）
    2888.00,                         -- 订单金额
    '订单金额异常，需要人工审核',   -- 申请原因
    NOW(),                           -- 创建时间
    NOW()                            -- 更新时间
);
```

### Go语言集成示例

```go
package main

import (
    "example_shop/common/model/audit"
    "time"
    "gorm.io/gorm"
)

// 外部系统提交审核数据
func SubmitAuditToDatabase(db *gorm.DB, orderID uint64, hotelInfo HotelInfo) error {
    // 开启事务
    tx := db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // 1. 插入审核主表
    auditMain := &audit.AuditMain{
        BusinessType:   2,              // 酒店订单
        BusinessId:     orderID,
        AuditStatus:    1,              // 待审核
        SubmitUserId:   hotelInfo.UserID,
        SubmitUserName: hotelInfo.UserName,
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }
    
    if err := tx.Create(auditMain).Error; err != nil {
        tx.Rollback()
        return err
    }

    // 2. 插入酒店审核明细表
    hotelOrder := &audit.AuditHotelOrder{
        AuditMainId:   auditMain.ID,
        BusinessRelId: orderID,
        HotelId:       hotelInfo.HotelID,
        HotelName:     hotelInfo.HotelName,
        HotelAddress:  hotelInfo.HotelAddress,
        RoomType:      hotelInfo.RoomType,
        CheckInTime:   hotelInfo.CheckInTime,
        CheckOutTime:  hotelInfo.CheckOutTime,
        GuestName:     hotelInfo.GuestName,
        GuestIdCard:   maskIDCard(hotelInfo.GuestIDCard),
        OrderAmount:   hotelInfo.OrderAmount,
        ApplyReason:   hotelInfo.ApplyReason,
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }
    
    if err := tx.Create(hotelOrder).Error; err != nil {
        tx.Rollback()
        return err
    }

    // 提交事务
    return tx.Commit().Error
}

// 身份证号脱敏
func maskIDCard(idCard string) string {
    if len(idCard) < 8 {
        return idCard
    }
    return idCard[:4] + "**********" + idCard[len(idCard)-4:]
}
```

## 调用示例

### cURL示例

查询审核详情：
```bash
curl -X GET http://localhost:8080/api/v1/audit/12345
```

查询待审核列表：
```bash
curl -X GET "http://localhost:8080/api/v1/audit/list?business_type=2&audit_status=1&page=1&page_size=10"
```

审核通过：
```bash
curl -X POST http://localhost:8080/api/v1/audit/process \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 2001" \
  -H "X-User-Name: 审核员王五" \
  -d '{
    "audit_id": 12345,
    "action": "approve",
    "audit_remark": "审核通过，订单信息无误"
  }'
```

### Go语言查询示例

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

// 查询审核详情
func GetAuditDetail(auditID uint64) {
    url := fmt.Sprintf("http://localhost:8080/api/v1/audit/%d", auditID)
    
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("请求失败:", err)
        return
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Println("响应:", result)
}

// 处理审核
func ProcessAudit(auditID uint64, action, remark string, auditorID uint64, auditorName string) {
    url := "http://localhost:8080/api/v1/audit/process"
    
    data := map[string]interface{}{
        "audit_id":     auditID,
        "action":       action,
        "audit_remark": remark,
    }
    
    jsonData, _ := json.Marshal(data)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-User-ID", fmt.Sprintf("%d", auditorID))
    req.Header.Set("X-User-Name", auditorName)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("请求失败:", err)
        return
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Println("响应:", result)
}
```

## 注意事项

1. **数据提交方式**: 外部系统直接将审核数据写入数据库，不通过HTTP接口提交
2. **身份证号脱敏**: 外部系统在写入数据库时应该对身份证号进行脱敏处理
3. **事务处理**: 插入审核主表和明细表时应该使用数据库事务，保证数据一致性
4. **审核员认证**: 处理审核接口需要通过请求头传递审核员信息
5. **时间格式**: 所有时间字段使用ISO 8601格式或MySQL DATETIME格式
6. **金额精度**: 金额字段保留2位小数

## 联系方式

如有问题，请联系技术支持团队。
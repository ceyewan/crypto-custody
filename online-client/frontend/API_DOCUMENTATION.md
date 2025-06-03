# 在线系统 API 开发文档

## 概述

本文档描述了在线系统的 RESTful API 接口，包括用户管理、账户管理和交易功能。

## 基础信息

- **基础URL**: `http://localhost:8080` (根据实际部署环境调整)
- **数据格式**: JSON
- **认证方式**: JWT Token (通过 Authorization 头部传递)

## 通用响应格式

### 成功响应
```json
{
  "code": 200,
  "message": "操作成功",
  "data": {} // 具体数据内容
}
```

### 错误响应
```json
{
  "code": 400,
  "message": "错误信息描述"
}
```

## 用户权限说明

- **Guest**: 访客权限 (最低权限)
- **Officer**: 警员权限 (可管理账户和交易)
- **Admin**: 管理员权限 (可管理所有功能)

---

# 用户管理 API

## 1. 用户登录

**接口**: `POST /api/login`
**权限**: 无需认证
**描述**: 用户登录获取 JWT Token

### 请求参数
```json
{
  "username": "string", // 用户名 (必填)
  "password": "string"  // 密码 (必填)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin"
    }
  }
}
```

## 2. 用户注册

**接口**: `POST /api/register`
**权限**: 无需认证
**描述**: 注册新用户账户

### 请求参数
```json
{
  "username": "string", // 用户名 (必填)
  "password": "string", // 密码 (必填)
  "email": "string"     // 邮箱 (必填，格式验证)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "注册成功",
  "data": {
    "id": 2,
    "username": "newuser",
    "email": "newuser@example.com",
    "role": "guest"
  }
}
```

## 3. 验证Token

**接口**: `POST /api/check-auth`
**权限**: 无需认证
**描述**: 验证 JWT Token 是否有效

### 请求参数
```json
{
  "token": "string" // JWT Token (必填)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "令牌有效",
  "data": {
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com", 
      "role": "admin"
    }
  },
  "valid": true
}
```

## 4. 获取当前用户信息

**接口**: `GET /api/users/profile`
**权限**: 需要JWT认证
**描述**: 获取当前登录用户的详细信息

### 请求头部
```
Authorization: <token>
```

### 响应数据
```json
{
  "code": 200,
  "message": "获取当前用户信息成功",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin"
  }
}
```

## 5. 用户登出

**接口**: `POST /api/users/logout`
**权限**: 需要JWT认证
**描述**: 登出当前用户，将Token加入黑名单

### 请求头部
```
Authorization: <token>
```

### 响应数据
```json
{
  "code": 200,
  "message": "登出成功"
}
```

## 6. 修改密码

**接口**: `POST /api/users/change-password`
**权限**: 需要JWT认证
**描述**: 用户修改自己的密码

### 请求头部
```
Authorization: <token>
```

### 请求参数
```json
{
  "oldPassword": "string", // 旧密码 (必填)
  "newPassword": "string"  // 新密码 (必填，最少6位)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "密码修改成功"
}
```

## 7. 获取所有用户 (管理员)

**接口**: `GET /api/users/admin/users`
**权限**: 管理员
**描述**: 获取系统中所有用户列表

### 请求头部
```
Authorization: <token>
```

### 响应数据
```json
{
  "code": 200,
  "message": "获取用户列表成功",
  "data": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin"
    },
    {
      "id": 2,
      "username": "officer1",
      "email": "officer1@example.com", 
      "role": "officer"
    }
  ]
}
```

## 8. 获取指定用户信息 (管理员)

**接口**: `GET /api/users/admin/users/{id}`
**权限**: 管理员
**描述**: 根据用户ID获取用户详细信息

### 请求头部
```
Authorization: <token>
```

### 路径参数
- `id`: 用户ID (整数)

### 响应数据
```json
{
  "code": 200,
  "message": "获取用户信息成功",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin"
  }
}
```

## 9. 更新用户角色 (管理员)

**接口**: `PUT /api/users/admin/users/{id}/role`
**权限**: 管理员
**描述**: 修改指定用户的角色

### 请求头部
```
Authorization: Bearer <token>
```

### 路径参数
- `id`: 用户ID (整数)

### 请求参数
```json
{
  "role": "string" // 角色: "admin", "officer", "guest"
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "用户角色更新成功"
}
```

## 10. 更新用户名 (管理员)

**接口**: `PUT /api/users/admin/users/{id}/username`
**权限**: 管理员
**描述**: 修改指定用户的用户名

### 请求头部
```
Authorization: Bearer <token>
```

### 路径参数
- `id`: 用户ID (整数)

### 请求参数
```json
{
  "username": "string" // 新用户名 (必填)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "用户名更新成功"
}
```

## 11. 管理员修改用户密码

**接口**: `PUT /api/users/admin/users/{id}/password`
**权限**: 管理员
**描述**: 管理员直接修改用户密码

### 请求头部
```
Authorization: Bearer <token>
```

### 路径参数
- `id`: 用户ID (整数)

### 请求参数
```json
{
  "newPassword": "string" // 新密码 (必填，最少6位)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "用户密码修改成功"
}
```

## 12. 删除用户 (管理员)

**接口**: `DELETE /api/users/admin/users/{id}`
**权限**: 管理员
**描述**: 删除指定用户账户

### 请求头部
```
Authorization: Bearer <token>
```

### 路径参数
- `id`: 用户ID (整数)

### 响应数据
```json
{
  "code": 200,
  "message": "用户删除成功"
}
```

---

# 账户管理 API

## 1. 根据地址查询账户

**接口**: `GET /api/accounts/address/{address}`
**权限**: 公开接口
**描述**: 通过账户地址查询账户信息

### 路径参数
- `address`: 账户地址 (字符串)

### 响应数据
```json
{
  "code": 200,
  "message": "查询账户成功",
  "data": {
    "address": "0x1234567890abcdef...",
    "coinType": "ETH",
    "balance": "1.500000000000000000",
    "importedBy": "admin",
    "description": "主账户"
  }
}
```

## 2. 获取用户账户列表 (警员+)

**接口**: `GET /api/accounts/officer/`
**权限**: 警员及以上
**描述**: 获取当前用户导入的所有账户

### 请求头部
```
Authorization: Bearer <token>
```

### 响应数据
```json
{
  "code": 200,
  "message": "查询账户列表成功",
  "data": [
    {
      "address": "0x1234567890abcdef...",
      "coinType": "ETH",
      "balance": "1.500000000000000000",
      "importedBy": "officer1",
      "description": "账户1"
    },
    {
      "address": "0xabcdef1234567890...",
      "coinType": "ETH", 
      "balance": "0.250000000000000000",
      "importedBy": "officer1",
      "description": "账户2"
    }
  ]
}
```

## 3. 创建账户 (警员+)

**接口**: `POST /api/accounts/officer/create`
**权限**: 警员及以上
**描述**: 创建新的账户记录

### 请求头部
```
Authorization: Bearer <token>
```

### 请求参数
```json
{
  "address": "string",     // 账户地址 (必填)
  "coinType": "string",    // 币种类型 (必填)
  "description": "string"  // 描述信息 (可选)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "创建账户成功"
}
```

## 4. 批量导入账户 (警员+)

**接口**: `POST /api/accounts/officer/import`
**权限**: 警员及以上
**描述**: 批量导入多个账户

### 请求头部
```
Authorization: Bearer <token>
```

### 请求参数
```json
{
  "accounts": [
    {
      "address": "string",     // 账户地址 (必填)
      "coinType": "string",    // 币种类型 (必填)
      "description": "string"  // 描述信息 (可选)
    },
    {
      "address": "string",
      "coinType": "string", 
      "description": "string"
    }
  ]
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "批量导入账户成功"
}
```

## 5. 获取所有账户 (管理员)

**接口**: `GET /api/accounts/admin/all`
**权限**: 管理员
**描述**: 获取系统中的所有账户信息

### 请求头部
```
Authorization: Bearer <token>
```

### 响应数据
```json
{
  "code": 200,
  "message": "查询所有账户成功",
  "data": {
    "accounts": [
      {
        "address": "0x1234567890abcdef...",
        "coinType": "ETH",
        "balance": "1.500000000000000000", 
        "importedBy": "admin",
        "description": "账户1"
      },
      {
        "address": "0xabcdef1234567890...",
        "coinType": "ETH",
        "balance": "0.250000000000000000",
        "importedBy": "officer1", 
        "description": "账户2"
      }
    ],
    "total": 2
  }
}
```

---

# 交易管理 API

## 1. 获取账户余额

**接口**: `GET /api/transaction/balance/{address}`
**权限**: 公开接口
**描述**: 查询指定地址的ETH余额

### 路径参数
- `address`: 账户地址 (字符串)

### 响应数据
```json
{
  "code": 200,
  "message": "获取余额成功",
  "data": {
    "address": "0x1234567890abcdef...",
    "balance": "1.500000000000000000"
  }
}
```

## 2. 准备交易 (警员+)

**接口**: `POST /api/transaction/tx/prepare`
**权限**: 警员及以上
**描述**: 准备一笔交易，返回交易ID和消息哈希用于签名

### 请求头部
```
Authorization: Bearer <token>
```

### 请求参数
```json
{
  "fromAddress": "string", // 发送方地址 (必填)
  "toAddress": "string",   // 接收方地址 (必填)
  "amount": 1.5            // 转账金额 (必填，大于0)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "交易准备成功",
  "data": {
    "transactionId": 1,
    "messageHash": "0xabcdef1234567890..."
  }
}
```

## 3. 签名并发送交易 (警员+)

**接口**: `POST /api/transaction/tx/sign-send`
**权限**: 警员及以上
**描述**: 使用签名完成交易并发送到区块链

### 请求头部
```
Authorization: Bearer <token>
```

### 请求参数
```json
{
  "messageHash": "string", // 消息哈希 (必填)
  "signature": "string"    // 签名数据 (必填)
}
```

### 响应数据
```json
{
  "code": 200,
  "message": "交易签名并发送成功",
  "data": {
    "transactionId": 1,
    "txHash": "0x1234567890abcdef..."
  }
}
```

## 4. 获取交易列表 (警员+)

**接口**: `GET /api/transaction/list`
**权限**: 警员及以上
**描述**: 获取当前用户的交易列表

### 请求头部
```
Authorization: Bearer <token>
```

### 查询参数
- `fromAddress`: 发送方地址 (可选)
- `toAddress`: 接收方地址 (可选)
- `status`: 交易状态 (可选: prepared, signed, sent, confirmed, failed)

### 响应数据
```json
{
  "code": 200,
  "message": "获取交易列表成功",
  "data": {
    "transactions": [
      {
        "id": 1,
        "fromAddress": "0x1234567890abcdef...",
        "toAddress": "0xabcdef1234567890...",
        "amount": "1.500000000000000000",
        "status": "confirmed",
        "txHash": "0x9876543210fedcba...",
        "messageHash": "0xfedcba0987654321...",
        "createdAt": "2023-01-01T10:00:00.000Z",
        "updatedAt": "2023-01-01T10:05:00.000Z"
      }
    ],
    "total": 1
  }
}
```

## 5. 获取所有交易 (管理员)

**接口**: `GET /api/transaction/admin/all`
**权限**: 管理员
**描述**: 获取系统中的所有交易记录

### 请求头部
```
Authorization: Bearer <token>
```

### 查询参数
- `fromAddress`: 发送方地址 (可选)
- `toAddress`: 接收方地址 (可选)
- `status`: 交易状态 (可选: prepared, signed, sent, confirmed, failed)

### 响应数据
```json
{
  "code": 200,
  "message": "获取所有交易成功",
  "data": {
    "transactions": [
      {
        "id": 1,
        "fromAddress": "0x1234567890abcdef...",
        "toAddress": "0xabcdef1234567890...",
        "amount": "1.500000000000000000",
        "status": "confirmed",
        "txHash": "0x9876543210fedcba...",
        "messageHash": "0xfedcba0987654321...",
        "createdBy": "officer1",
        "createdAt": "2023-01-01T10:00:00.000Z",
        "updatedAt": "2023-01-01T10:05:00.000Z"
      }
    ],
    "total": 1
  }
}
```

## 6. 获取交易详情

**接口**: `GET /api/transaction/{id}`
**权限**: 警员及以上
**描述**: 根据交易ID获取交易详细信息

### 请求头部
```
Authorization: Bearer <token>
```

### 路径参数
- `id`: 交易ID (整数)

### 响应数据
```json
{
  "code": 200,
  "message": "获取交易详情成功",
  "data": {
    "id": 1,
    "fromAddress": "0x1234567890abcdef...",
    "toAddress": "0xabcdef1234567890...",
    "amount": "1.500000000000000000",
    "status": "confirmed",
    "txHash": "0x9876543210fedcba...",
    "messageHash": "0xfedcba0987654321...",
    "createdBy": "officer1",
    "createdAt": "2023-01-01T10:00:00.000Z",
    "updatedAt": "2023-01-01T10:05:00.000Z"
  }
}
```

## 7. 获取交易统计 (警员+)

**接口**: `GET /api/transaction/stats`
**权限**: 警员及以上
**描述**: 获取当前用户的交易统计信息

### 请求头部
```
Authorization: Bearer <token>
```

### 响应数据
```json
{
  "code": 200,
  "message": "获取交易统计成功",
  "data": {
    "count": 25,
    "confirmed": 20,
    "pending": 3,
    "failed": 2
  }
}
```

## 8. 获取所有交易统计 (管理员)

**接口**: `GET /api/transaction/admin/stats`
**权限**: 管理员
**描述**: 获取系统中所有交易的统计信息

### 请求头部
```
Authorization: Bearer <token>
```

### 响应数据
```json
{
  "code": 200,
  "message": "获取所有交易统计成功",
  "data": {
    "count": 150,
    "confirmed": 120,
    "pending": 20,
    "failed": 10
  }
}
```

---

# 错误码说明

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权/Token无效 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

# 使用示例

## JavaScript/TypeScript 示例

### 用户登录
```javascript
const login = async (username, password) => {
  const response = await fetch('/api/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      username,
      password
    })
  });
  
  const data = await response.json();
  if (data.code === 200) {
    // 保存token
    localStorage.setItem('token', data.data.token);
    return data.data;
  }
  throw new Error(data.message);
};
```

### 获取账户列表
```javascript
const getAccounts = async () => {
  const token = localStorage.getItem('token');
  const response = await fetch('/api/accounts/officer/', {
    headers: {
      'Authorization': token
    }
  });
  
  const data = await response.json();
  if (data.code === 200) {
    return data.data;
  }
  throw new Error(data.message);
};
```

### 准备交易
```javascript
const prepareTransaction = async (fromAddress, toAddress, amount) => {
  const token = localStorage.getItem('token');
  const response = await fetch('/api/transaction/tx/prepare', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': token
    },
    body: JSON.stringify({
      fromAddress,
      toAddress,
      amount
    })
  });
  
  const data = await response.json();
  if (data.code === 200) {
    return data.data;
  }
  throw new Error(data.message);
};
```

---

# 注意事项

1. **认证**: 需要认证的接口必须在请求头中包含有效的JWT Token
2. **权限**: 不同接口需要不同的用户权限，请确保当前用户具有相应权限
3. **参数验证**: 所有必填参数都会进行严格验证，请确保参数格式正确
4. **错误处理**: 请根据返回的状态码和错误消息进行相应的错误处理
5. **Token过期**: JWT Token有过期时间，过期后需要重新登录获取新Token

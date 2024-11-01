# Crypto Custody API Documentation

## Base URL
```
/api/v1
```

## 角色说明
- SYSTEM_ADMIN: 系统管理员
- CASE_ADMIN: 案件管理员
- AUDIT_ADMIN: 审计管理员
- USER: 普通用户

## 认证方式
除了公开接口外，其他所有接口都需要在请求头中携带认证信息：
```
Authorization: Bearer <token>
```

## API 接口列表

### 1. 健康检查
```
GET /health
```
响应：
```json
{
    "status": "ok"
}
```

### 2. 认证相关接口

#### 2.1 用户注册
```
POST /api/v1/auth/register
```
请求体：
```json
{
    "username": "string",
    "password": "string",
    "email": "string",
    "phone": "string"
}
```

#### 2.2 用户登录
```
POST /api/v1/auth/login
```
请求体：
```json
{
    "username": "string",
    "password": "string"
}
```

#### 2.3 刷新令牌
```
POST /api/v1/auth/refresh
```

### 3. 用户管理接口

#### 3.1 获取个人信息
```
GET /api/v1/users/me
```

#### 3.2 更新个人信息
```
PUT /api/v1/users/me
```
请求体：
```json
{
    "email": "string",
    "phone": "string",
    "oldPassword": "string",
    "newPassword": "string"
}
```

#### 3.3 获取用户列表（系统管理员）
```
GET /api/v1/admin/users
```
查询参数：
```
page: 页码
size: 每页数量
username: 用户名搜索
role: 角色筛选
```

#### 3.4 创建用户（系统管理员）
```
POST /api/v1/admin/users
```
请求体：
```json
{
    "username": "string",
    "password": "string",
    "email": "string",
    "phone": "string",
    "roles": ["string"]
}
```

#### 3.5 更新用户角色（系统管理员）
```
PUT /api/v1/admin/users/:id/roles
```
请求体：
```json
{
    "roles": ["string"]
}
```

#### 3.6 删除用户（系统管理员）
```
DELETE /api/v1/admin/users/:id
```

### 4. 案件管理接口

#### 4.1 创建案件（系统管理员）
```
POST /api/v1/cases
```
请求体：
```json
{
    "name": "string",
    "description": "string",
    "type": "string",
    "status": "string"
}
```

#### 4.2 获取案件列表
```
GET /api/v1/cases
```
查询参数：
```
page: 页码
size: 每页数量
name: 案件名称搜索
type: 案件类型
status: 案件状态
startTime: 开始时间
endTime: 结束时间
```

#### 4.3 获取案件详情
```
GET /api/v1/cases/:id
```

#### 4.4 更新案件（系统管理员/案件管理员）
```
PUT /api/v1/cases/:id
```
请求体：
```json
{
    "name": "string",
    "description": "string",
    "type": "string",
    "status": "string"
}
```

#### 4.5 删除案件（系统管理员）
```
DELETE /api/v1/cases/:id
```

#### 4.6 分配案件权限（系统管理员）
```
POST /api/v1/cases/:id/permissions
```
请求体：
```json
{
    "userId": "string",
    "permissions": ["READ", "WRITE", "DELETE"]
}
```

#### 4.7 撤销案件权限（系统管理员）
```
DELETE /api/v1/cases/:id/permissions/:userId
```

### 5. 钱包管理接口

#### 5.1 创建钱包（系统管理员/案件管理员）
```
POST /api/v1/wallets
```
请求体：
```json
{
    "caseId": "string",
    "type": "string",
    "description": "string"
}
```

#### 5.2 获取钱包列表
```
GET /api/v1/wallets
```
查询参数：
```
page: 页码
size: 每页数量
caseId: 案件ID
address: 钱包地址
type: 钱包类型
```

#### 5.3 获取钱包详情
```
GET /api/v1/wallets/:address
```

#### 5.4 删除钱包（系统管理员）
```
DELETE /api/v1/wallets/:address
```

#### 5.5 发起转账（系统管理员/案件管理员）
```
POST /api/v1/wallets/:address/transfer
```
请求体：
```json
{
    "toAddress": "string",
    "amount": "string",
    "currency": "string",
    "description": "string"
}
```

### 6. 日志管理接口

#### 6.1 获取系统操作日志
```
GET /api/v1/logs/system
```
查询参数：
```
page: 页码
size: 每页数量
userId: 操作用户ID
action: 操作类型
startTime: 开始时间
endTime: 结束时间
```

#### 6.2 获取案件操作日志
```
GET /api/v1/logs/cases
```
查询参数：
```
page: 页码
size: 每页数量
caseId: 案件ID
userId: 操作用户ID
action: 操作类型
startTime: 开始时间
endTime: 结束时间
```

#### 6.3 获取交易操作日志
```
GET /api/v1/logs/transactions
```
查询参数：
```
page: 页码
size: 每页数量
caseId: 案件ID
walletAddress: 钱包地址
userId: 操作用户ID
status: 交易状态
startTime: 开始时间
endTime: 结束时间
```

#### 6.4 删除日志（系统管理员）
```
DELETE /api/v1/logs/:id
```

## 错误码说明

```
200: 成功
400: 请求参数错误
401: 未认证
403: 权限不足
404: 资源不存在
500: 服务器内部错误
```

## 通用响应格式
```json
{
    "code": 200,
    "message": "success",
    "data": {}
}
```

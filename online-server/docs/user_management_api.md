# 用户管理系统 API 文档

## 概述

用户管理系统提供了完整的用户认证、授权和管理功能，包括用户注册、登录、权限控制以及管理员对用户的管理功能。

本文档主要描述API接口的使用方法、参数要求和返回值格式。

## 基础信息

- 基础URL: `/api`
- 所有请求和响应均使用JSON格式
- 需要认证的API需要在请求头中包含`Authorization`字段，值为登录时获取的令牌
- 所有响应都包含以下字段：
  - `code`: 状态码，200表示成功，其他值表示错误
  - `message`: 状态描述信息
  - `data`: 响应数据（可选）

## 公开API

### 1. 用户注册

- **路径**: `/register`
- **方法**: POST
- **描述**: 创建新用户账号
- **请求体**:
  ```json
  {
    "username": "string", // 用户名，必填
    "password": "string", // 密码，必填
    "email": "string"     // 邮箱，必填，必须是有效邮箱格式
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "message": "注册成功",
    "data": {
      "id": 1,
      "username": "string",
      "email": "string"
    }
  }
  ```
- **错误响应**:
  - 400: 请求参数不正确
  - 400: 用户名已存在
  - 400: 邮箱已被使用

### 2. 用户登录

- **路径**: `/login`
- **方法**: POST
- **描述**: 用户登录并获取认证令牌
- **请求体**:
  ```json
  {
    "username": "string", // 用户名，必填
    "password": "string"  // 密码，必填
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "message": "登录成功",
    "data": {
      "token": "string",
      "user": {
        "id": 1,
        "username": "string",
        "email": "string",
        "role": "string"
      }
    }
  }
  ```
- **错误响应**:
  - 400: 请求参数不正确
  - 401: 用户名或密码错误

### 3. 验证令牌

- **路径**: `/check-auth`
- **方法**: POST
- **描述**: 验证令牌是否有效
- **请求体**:
  ```json
  {
    "token": "string" // 令牌，必填
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "message": "令牌有效",
    "valid": true,
    "data": {
      "user": {
        "id": 1,
        "username": "string",
        "email": "string",
        "role": "string"
      }
    }
  }
  ```
- **错误响应**:
  - 400: 请求参数不正确
  - 401: 令牌无效

## 需要认证的API

这些API需要在请求头中包含有效的`Authorization`令牌。

### 1. 用户登出

- **路径**: `/users/logout`
- **方法**: POST
- **描述**: 使当前令牌失效
- **请求头**: `Authorization: 令牌值`
- **响应**:
  ```json
  {
    "code": 200,
    "message": "登出成功"
  }
  ```

### 2. 获取当前用户信息

- **路径**: `/users/profile`
- **方法**: GET
- **描述**: 获取当前登录用户的详细信息
- **请求头**: `Authorization: 令牌值`
- **响应**:
  ```json
  {
    "code": 200,
    "message": "获取当前用户信息成功",
    "data": {
      "id": 1,
      "username": "string",
      "email": "string",
      "role": "string",
      "createdAt": "string"
    }
  }
  ```

### 3. 修改密码

- **路径**: `/users/change-password`
- **方法**: POST
- **描述**: 修改当前用户密码
- **请求头**: `Authorization: 令牌值`
- **请求体**:
  ```json
  {
    "oldPassword": "string", // 原密码，必填
    "newPassword": "string"  // 新密码，必填，至少6个字符
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "message": "密码修改成功"
  }
  ```
- **错误响应**:
  - 400: 请求参数不正确
  - 400: 原密码不正确

## 管理员API

以下API仅限管理员使用，需要在请求头中包含具有管理员权限的`Authorization`令牌。

### 1. 获取所有用户

- **路径**: `/users/admin/users`
- **方法**: GET
- **描述**: 获取系统中所有用户的列表
- **请求头**: `Authorization: 管理员令牌值`
- **响应**:
  ```json
  {
    "code": 200,
    "message": "获取用户列表成功",
    "data": [
      {
        "id": 1,
        "username": "string",
        "email": "string",
        "role": "string",
        "createdAt": "string"
      },
      // ...更多用户
    ]
  }
  ```
- **错误响应**:
  - 403: 权限不足
  - 500: 系统错误

### 2. 获取指定用户

- **路径**: `/users/admin/users/:id`
- **方法**: GET
- **描述**: 获取指定用户的详细信息
- **请求头**: `Authorization: 管理员令牌值`
- **参数**: `id` - 用户ID
- **响应**:
  ```json
  {
    "code": 200,
    "message": "获取用户信息成功",
    "data": {
      "id": 1,
      "username": "string",
      "email": "string",
      "role": "string",
      "createdAt": "string"
    }
  }
  ```
- **错误响应**:
  - 403: 权限不足
  - 404: 用户不存在

### 3. 更新用户角色

- **路径**: `/users/admin/users/:id/role`
- **方法**: PUT
- **描述**: 更新非管理员用户的角色
- **请求头**: `Authorization: 管理员令牌值`
- **参数**: `id` - 用户ID
- **请求体**:
  ```json
  {
    "role": "string" // 新角色，必填，有效值: "admin", "officer", "guest"
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "message": "用户角色更新成功"
  }
  ```
- **错误响应**:
  - 400: 无效的角色值
  - 403: 权限不足
  - 400: 不允许修改管理员用户的角色

### 4. 更新用户名

- **路径**: `/users/admin/users/:id/username`
- **方法**: PUT
- **描述**: 更新非管理员用户的用户名
- **请求头**: `Authorization: 管理员令牌值`
- **参数**: `id` - 用户ID
- **请求体**:
  ```json
  {
    "username": "string" // 新用户名，必填
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "message": "用户名更新成功"
  }
  ```
- **错误响应**:
  - 400: 用户名已被使用
  - 403: 权限不足
  - 400: 不允许修改管理员用户的用户名

### 5. 删除用户

- **路径**: `/users/admin/users/:id`
- **方法**: DELETE
- **描述**: 删除指定用户
- **请求头**: `Authorization: 管理员令牌值`
- **参数**: `id` - 用户ID
- **响应**:
  ```json
  {
    "code": 200,
    "message": "用户删除成功"
  }
  ```
- **错误响应**:
  - 403: 权限不足
  - 500: 删除用户失败

## 角色与权限

系统定义了三种用户角色，不同角色具有不同的权限：

1. **管理员 (admin)**
   - 可以访问所有API
   - 可以管理所有非管理员用户
   - 可以修改非管理员用户的角色和用户名

2. **警员 (officer)**
   - 可以访问基本用户功能
   - 可以访问需要警员权限的特定功能

3. **游客 (guest)**
   - 只能访问基本用户功能

## 安全说明

1. 所有密码都经过哈希处理后存储
2. JWT令牌具有24小时有效期
3. 已登出的令牌会被加入黑名单，无法重复使用
4. 系统会记录所有关键操作的日志

## 错误码说明

- 200: 请求成功
- 400: 请求参数错误或业务错误
- 401: 未授权或令牌无效
- 403: 权限不足
- 404: 资源不存在
- 500: 服务器内部错误
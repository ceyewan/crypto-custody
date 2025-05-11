# 以太坊交易管理系统

本系统提供基于RBAC（基于角色的访问控制）的用户管理和以太坊交易功能，支持用户注册、登录、权限管理以及区块链交易操作。系统采用Go语言开发，使用Gin框架提供Web API服务。

## 功能特性

- **用户管理**：注册、登录、登出、密码修改
- **角色权限控制**：支持管理员(admin)、警员(officer)和游客(guest)三种角色
- **以太坊账户管理**：创建、查询账户
- **交易管理**：预备交易、签名交易、发送交易、查询交易状态
- **区块链集成**：连接以太坊网络，支持Sepolia测试网

## 系统架构

```
├── ethereum/       # 以太坊相关功能
│   ├── client.go   # 以太坊客户端
│   ├── service.go  # 以太坊服务
│   └── transaction_manager.go  # 交易管理器
├── handlers/       # API处理函数
│   ├── account.go  # 账户相关处理
│   ├── ethereum.go # 以太坊交易相关处理
│   ├── role.go     # 角色管理相关处理
│   └── user.go     # 用户相关处理
├── model/          # 数据模型
│   ├── accounts.go # 账户模型
│   ├── transactions.go # 交易模型
│   └── users.go    # 用户模型
├── routes/         # API路由
│   ├── account.go  # 账户相关路由
│   ├── ethereum.go # 以太坊相关路由
│   ├── role.go     # 角色相关路由
│   └── user.go     # 用户相关路由
├── servers/        # 服务实现
│   └── eth_service.go  # 以太坊交易服务
├── utils/          # 工具函数
│   ├── database.go # 数据库操作
│   ├── hash.go     # 密码哈希
│   ├── helpers.go  # 辅助函数
│   ├── jwt.go      # JWT令牌
│   ├── logger.go   # 日志
│   └── middleware.go  # 中间件
├── main.go         # 入口文件
└── go.mod          # Go模块定义
```

## 角色与权限

系统定义了三种角色，每种角色具有不同的权限：

1. **管理员(admin)**
   - 可以管理所有用户（查看、创建、修改、删除）
   - 可以管理角色和权限
   - 可以执行所有交易操作
   - 可以访问系统管理功能

2. **警员(officer)**
   - 可以创建和管理账户
   - 可以执行交易操作（准备、签名、发送）
   - 可以查看交易历史和状态

3. **游客(guest)**
   - 只能查看公开信息
   - 可以查看自己的账户信息
   - 不能执行交易操作

## API接口说明

### 用户管理

- **POST /api/login**：用户登录
- **POST /api/register**：用户注册
- **GET /api/users/profile**：获取当前用户信息
- **POST /api/users/logout**：用户登出
- **POST /api/users/change-password**：修改密码
- **GET /api/users/**：获取所有用户（仅管理员）
- **GET /api/users/:id**：获取特定用户（仅管理员）

### 角色管理

- **GET /api/roles/**：获取所有角色
- **POST /api/admin/roles/check-permission**：检查权限
- **POST /api/admin/roles/change-user-role**：修改用户角色（仅管理员）

### 账户管理

- **GET /api/accounts/:address**：查询账户余额
- **GET /api/accounts/**：获取账户列表（需认证）
- **POST /api/accounts/**：创建账户（警员或管理员）
- **POST /api/accounts/packTransferData**：打包交易数据（警员或管理员）
- **POST /api/accounts/sendTransaction**：发送交易（警员或管理员）
- **GET /api/accounts/admin/transferAll**：批量转账（仅管理员）
- **GET /api/accounts/admin/updateBalance**：更新所有账户余额（仅管理员）

### 以太坊交易

- **GET /api/ethereum/balance/:address**：获取地址余额
- **GET /api/ethereum/transaction/:id**：获取交易状态
- **GET /api/ethereum/transactions/:address**：获取用户交易历史
- **POST /api/ethereum/prepare**：准备交易（警员或管理员）
- **POST /api/ethereum/sign**：签名交易（警员或管理员）
- **POST /api/ethereum/send/:id**：发送交易（警员或管理员）
- **POST /api/ethereum/admin/check-pending**：检查待处理交易（仅管理员）

## 交易流程

系统的交易过程分为两步：

1. **打包交易**：
   - 用户通过API提交发送方地址、接收方地址和金额
   - 系统生成交易数据并返回交易哈希，等待签名

2. **发送交易**：
   - 用户提供交易哈希和签名
   - 系统验证签名并将交易提交到以太坊网络
   - 交易上链后，系统会跟踪并更新交易状态

## 数据库设计

系统使用SQLite数据库，主要包含以下表：

1. **users**：存储用户信息
   - ID, Username, Password, Email, Role, CreatedAt, UpdatedAt, DeletedAt

2. **accounts**：存储以太坊账户信息
   - ID, Address, Balance, CreatedAt, UpdatedAt, DeletedAt

3. **transactions**：存储交易记录
   - ID, FromAddress, ToAddress, Value, Nonce, GasLimit, GasPrice, Data, TxHash, Status, MessageHash, Signature, BlockNumber, BlockHash, Error, SubmittedAt, ConfirmedAt, LastCheckedAt, RetryCount, TransactionJSON, CreatedAt, UpdatedAt, DeletedAt

## 安全措施

- 使用bcrypt算法对密码进行哈希处理
- 使用JWT令牌进行身份验证
- 实现基于角色的访问控制(RBAC)
- 交易过程中的签名验证
- 防止SQL注入和XSS攻击

## 部署指南

1. 克隆代码库
2. 安装依赖：`go mod tidy`
3. 运行应用：`go run main.go`
4. 默认服务运行在http://localhost:8080

## 注意事项

- 初始管理员账户：用户名`admin`，密码`admin123`
- 系统默认使用Sepolia测试网，如需更改网络配置，请修改`ethereum/client.go`中的配置
- 所有交易都在测试网上执行，请确保有足够的测试网ETH

## 客户端开发指南

基于本系统开发客户端应用时，需要实现以下功能：

1. **用户认证**：
   - 实现登录注册界面
   - 保存JWT令牌用于后续API调用
   - 处理令牌过期和刷新逻辑

2. **账户管理**：
   - 显示用户账户列表
   - 提供创建新账户功能
   - 显示账户余额

3. **交易功能**：
   - 交易表单（指定发送方、接收方和金额）
   - 签名机制（可使用Web3或类似库）
   - 交易状态显示和历史记录

4. **权限控制**：
   - 根据用户角色显示或隐藏功能
   - 实现管理员专属功能（如用户管理）

## 技术栈

- Go 1.16+
- Gin Web框架
- GORM ORM库
- SQLite数据库
- go-ethereum (以太坊Go客户端)
- JWT认证
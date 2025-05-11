# 以太坊交易管理系统

本系统提供基于RBAC（基于角色的访问控制）的用户管理和以太坊交易功能，支持用户注册、登录、权限管理以及区块链交易操作。系统采用Go语言开发，使用Gin框架提供Web API服务。

## 功能特性

- **用户管理**：注册、登录、登出、密码修改
- **角色权限控制**：支持管理员(admin)、警员(officer)和游客(guest)三种角色
- **以太坊账户管理**：创建、查询账户
- **交易管理**：预备交易、签名交易、发送交易、查询交易状态
- **区块链集成**：连接以太坊网络，支持Sepolia测试网

## 系统架构设计

系统采用了清晰的分层架构设计，遵循关注点分离原则，确保每一层都有明确的职责：

```
├── dto/            # 数据传输对象层
│   └── transaction.go  # 交易相关的数据传输对象
├── model/          # 数据模型层
│   ├── accounts.go # 账户模型
│   └── users.go    # 用户模型
├── ethereum/       # 以太坊交互层
│   ├── client.go   # 以太坊客户端(使用懒汉式加载)
│   └── service.go  # 以太坊服务(封装区块链交互)
├── service/        # 服务层
│   ├── account_service.go  # 账户服务(处理数据库操作)
│   ├── eth_service.go      # 以太坊服务(调用ethereum包)
│   └── user_service.go     # 用户服务(处理用户相关逻辑)
├── handlers/       # 处理器层
│   ├── account.go  # 账户处理器(处理数据库查询)
│   ├── ethereum.go # 以太坊处理器(处理区块链交互)
│   └── user.go     # 用户处理器(处理用户管理)
├── routes/         # 路由层
│   ├── account.go  # 账户相关路由
│   ├── ethereum.go # 以太坊相关路由
│   └── user.go     # 用户相关路由
├── utils/          # 工具层
│   ├── database.go # 数据库操作
│   ├── hash.go     # 密码哈希
│   └── jwt.go      # JWT令牌认证
├── main.go         # 应用入口
└── go.mod          # Go模块定义
```

### 分层详解

#### 1. 数据传输对象层 (DTO)
- 位于 `dto` 目录
- 定义了在不同层之间传递数据的结构
- 包括 `TransactionRequest`、`TransactionResponse`、`SignatureRequest` 等
- 职责：规范化数据格式，减少各层之间的耦合

#### 2. 数据模型层 (Model)
- 位于 `model` 目录
- 定义了与数据库表对应的结构体
- 包括 `User`、`Account` 等实体
- 职责：表示数据实体，与数据库结构对应

#### 3. 以太坊交互层 (Ethereum)
- 位于 `ethereum` 目录
- 提供与以太坊区块链的底层交互
- 采用懒汉式加载模式，确保资源高效利用
- 使用 `sync.Once` 保证线程安全的单例模式
- 职责：封装区块链API，处理交易构建和发送

#### 4. 服务层 (Service)
- 位于 `service` 目录
- 包含三种核心服务：
  - `AccountService`: 处理账户和交易记录的数据库操作
  - `EthService`: 调用以太坊层提供的功能，处理交易相关业务逻辑
  - `UserService`: 处理用户认证、注册、权限管理等业务逻辑
- 所有服务都采用懒汉式加载和 `sync.Once` 确保高效和线程安全
- 职责：实现业务逻辑，协调Model层和外部服务

#### 5. 处理器层 (Handlers)
- 位于 `handlers` 目录
- 处理HTTP请求，调用适当的服务
- 包括：
  - `account.go`: 处理与数据库相关的查询（如交易历史）
  - `ethereum.go`: 处理与区块链相关的操作（如余额查询、交易发送）
  - `user.go`: 处理用户相关操作（如登录、注册）
- 职责：解析请求，调用服务，格式化响应

#### 6. 路由层 (Routes)
- 位于 `routes` 目录
- 定义API路由，将请求映射到处理器
- 实现中间件集成，如认证检查、权限验证
- 职责：路由管理，请求分发

#### 7. 工具层 (Utils)
- 位于 `utils` 目录
- 提供各种辅助功能
- 职责：提供通用工具，如数据库连接、JWT验证、密码哈希等

## 设计优势

1. **关注点分离**：每一层都有明确的职责，改动某一层不影响其他层
2. **单一职责原则**：每个组件只负责一项功能，提高代码可维护性
3. **依赖注入**：通过服务单例实现依赖注入，降低耦合度
4. **资源懒加载**：采用懒汉式加载，优化资源使用
5. **线程安全**：使用 `sync.Once` 确保单例模式的线程安全
6. **可测试性**：清晰的层次结构使单元测试更加容易

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
- **GET /api/users/admin/users**：获取所有用户（仅管理员）
- **GET /api/users/admin/users/:id**：获取特定用户（仅管理员）
- **PUT /api/users/admin/users/:id/role**：更新用户角色（仅管理员）
- **DELETE /api/users/admin/users/:id**：删除用户（仅管理员）

### 账户与交易管理

- **GET /api/accounts/transactions/:id**：获取交易详情
- **GET /api/accounts/transactions/hash/:hash**：通过哈希获取交易
- **GET /api/accounts/transactions/user/:address**：获取用户交易历史

### 以太坊交易

- **GET /api/ethereum/balance/:address**：获取地址余额
- **POST /api/ethereum/tx/prepare**：准备交易（警员或管理员）
- **POST /api/ethereum/tx/sign-send**：签名并发送交易（警员或管理员）

## 交易流程

系统的交易过程分为两步：

1. **准备交易**：
   - 用户通过API提交发送方地址、接收方地址和金额
   - 系统生成交易数据并返回消息哈希，等待签名

2. **签名并发送交易**：
   - 用户提供消息哈希和签名
   - 系统验证签名并将交易提交到以太坊网络

## 懒汉式加载和Sync.Once

系统中的服务实例采用懒汉式加载和 `sync.Once` 实现线程安全的单例模式：

```go
var (
    serviceInstance     *Service
    serviceInstanceOnce sync.Once
)

func GetInstance() (*Service, error) {
    var initErr error
    
    serviceInstanceOnce.Do(func() {
        client, err := GetClientInstance()
        if err != nil {
            initErr = fmt.Errorf("初始化失败: %w", err)
            return
        }
        
        serviceInstance = &Service{
            client: client,
        }
    })
    
    if initErr != nil {
        return nil, initErr
    }
    
    return serviceInstance, nil
}
```

这种设计确保：
- 资源只在第一次被请求时初始化
- 即使在并发环境中也只初始化一次
- 避免了启动时的资源浪费

## 技术栈

- Go 1.16+
- Gin Web框架
- GORM ORM库
- SQLite数据库
- go-ethereum (以太坊Go客户端)
- JWT认证

## 部署指南

1. 克隆代码库
2. 安装依赖：`go mod tidy`
3. 运行应用：`go run main.go`
4. 默认服务运行在http://localhost:8080
## 改进小结

在项目重构过程中，我们进行了以下主要改进：

1. **添加DTO层**：创建了dto包，专门用于定义数据传输对象，使不同层之间的数据交换更加规范化。

2. **服务层优化**：
   - 设计了完整的服务层，包括EthService、AccountService和UserService
   - 所有服务都采用懒汉式加载和sync.Once确保线程安全和资源高效利用
   - 明确定义了每个服务的职责边界

3. **以太坊交互层重构**：
   - 优化client.go，使用懒汉式加载
   - 重构service.go，简化交易处理流程
   - 移除了冗余代码和全局变量

4. **处理器层职责划分**：
   - ethereum.go：处理与区块链直接交互的请求
   - account.go：处理与数据库交互的请求
   - user.go：处理用户管理相关请求
 
5. **路由优化**：路由结构更加清晰，映射到相应的处理函数

6. **资源利用改进**：
   - 避免在应用启动时初始化所有资源
   - 通过懒汉式加载实现按需初始化
   - 使用sync.Once确保并发安全

这些改进使系统架构更加清晰，代码组织更加规范，不仅提高了可维护性和可测试性，还优化了资源利用效率。

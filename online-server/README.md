# 以太坊交易管理系统 (Ethereum Transaction Management System)

![版本](https://img.shields.io/badge/版本-1.0.0-blue)![Go版本](https://img.shields.io/badge/Go-1.24.0-blue)![许可证](https://img.shields.io/badge/许可证-MIT-green)

## 项目概述

本项目是一个安全可靠的以太坊交易管理系统，专为需要安全管理加密资产的企业和组织设计。系统采用在线-离线分离的架构，确保私钥安全，同时提供完整的用户权限控制和交易生命周期管理。

### 背景

在加密货币托管服务中，安全性和可审计性至关重要。传统的热钱包方案存在私钥被盗风险，而纯冷钱包方案则操作不便。本系统结合两者优势，实现了高安全性与操作便捷性的平衡。

### 目标用户

- 加密货币托管服务提供商
- 数字资产管理机构
- 需要管理多个以太坊账户的企业
- 对交易安全性有高要求的组织

本系统提供基于RBAC（基于角色的访问控制）的用户管理和以太坊交易功能，支持用户注册、登录、权限管理以及区块链交易操作。系统采用Go语言开发，使用Gin框架提供Web API服务，能够安全高效地管理数字资产交易。

## 功能特性

- **用户管理**：注册、登录、登出、密码修改、角色管理
- **角色权限控制**：支持管理员(admin)、警员(officer)和游客(guest)三种角色
- **以太坊账户管理**：创建、查询账户、余额查询
- **交易管理**：预备交易、签名交易、发送交易、查询交易状态
- **区块链集成**：连接以太坊网络，支持Sepolia测试网
- **安全架构**：采用在线-离线分离的私钥管理方式，提高安全性

## 系统架构设计

系统采用了清晰的分层架构设计，遵循关注点分离原则，确保每一层都有明确的职责：

```
online-server/
├── dto/                    # 数据传输对象层
│   ├── account_dto.go      # 账户相关的数据传输对象
│   ├── transaction_dto.go  # 交易相关的数据传输对象
│   └── user_dto.go         # 用户相关的数据传输对象
├── model/                  # 数据模型层
│   ├── account_model.go    # 账户模型
│   ├── transaction_model.go # 交易模型
│   └── user_model.go       # 用户模型
├── ethereum/               # 以太坊交互层
│   ├── client.go           # 以太坊客户端(使用懒汉式加载)
│   ├── transaction_manager.go # 交易管理器
│   └── README.md           # 以太坊模块文档
├── service/                # 服务层
│   ├── account_service.go  # 账户服务(处理数据库操作)
│   ├── transaction_service.go # 交易服务
│   └── user_service.go     # 用户服务(处理用户相关逻辑)
├── handler/                # 处理器层
│   ├── account_handler.go  # 账户处理器(处理数据库查询)
│   ├── transaction_handler.go # 交易处理器(处理区块链交互)
│   └── user_handler.go     # 用户处理器(处理用户管理)
├── route/                  # 路由层
│   ├── account_router.go   # 账户相关路由
│   ├── transaction_router.go # 交易相关路由
│   ├── user_router.go      # 用户相关路由
│   └── router.go           # 主路由配置
├── middleware/             # 中间件
│   └── middleware.go       # 中间件定义
├── utils/                  # 工具层
│   ├── database.go         # 数据库操作
│   ├── hash.go             # 密码哈希
│   ├── jwt.go              # JWT令牌认证
│   ├── logger.go           # 日志管理
│   ├── logger_helpers.go   # 日志辅助函数
│   └── response.go         # HTTP响应工具
├── docs/                   # 文档
│   ├── user_management_api.md      # 用户管理API文档
│   ├── user_management_guide.md    # 用户管理开发指南
│   ├── transaction_management_api.md     # 交易管理API文档
│   └── transaction_management_guide.md   # 交易管理开发指南
├── tests/                  # 测试
│   ├── user_test.go        # 用户模块测试
│   └── logs/               # 测试日志
├── database/               # 数据库文件
│   └── crypto-custody.db   # SQLite数据库
├── logs/                   # 日志文件
├── main.go                 # 应用入口
└── go.mod                  # Go模块定义
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
- 提供与以太坊��块链的底层交互
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
- **PUT /api/users/admin/users/:id/username**：更新用户名（仅管理员）
- **DELETE /api/users/admin/users/:id**：删除用户（仅管理员）

### 账户与交易管理

- **GET /api/accounts/transactions/:id**：获取交易详情
- **GET /api/accounts/transactions/hash/:hash**：通过哈希获取交易
- **GET /api/accounts/transactions/user/:address**：获取用户交易历史
- **GET /api/accounts/transactions/status/:status**：获取指定状态的交易

### 以太坊交易

- **GET /api/transaction/balance/:address**：获取地址余额
- **POST /api/transaction/tx/prepare**：准备交易（警员或管理员）
- **POST /api/transaction/tx/sign-send**：签名并发送交易（警员或管理员）

详细API文档请参考:
- [用户管理API文档](docs/user_management_api.md)
- [交易管理API文档](docs/transaction_management_api.md)

## 交易流程

系统实现了安全的在线-离线分离交易流程：

1. **准备交易（在线系统）**：
   - 用户通过API提交发送方地址、接收方地址和金额
   - 系统获取nonce和估算gas费用
   - 系统生成交易数据和消息哈希
   - 将交易记录保存到数据库，状态设为"Created"
   - 返回消息哈希，等待签名

2. **离线签名（安全环境）**：
   - 消息哈希传递到安全的离线环境
   - 使用私钥对消息哈希进行签名
   - 生成签名数据

3. **签名验证并发送（在线系统）**：
   - 用户提供消息哈希和签名数据
   - 系统验证签名与发送方地址是否匹配
   - 系统将交易提交到以太坊网络
   - 更新交易状态为"Submitted"

4. **交易监控与确认**：
   - 系统自动监控交易状态
   - 交易被确认后，更新状态为"Confirmed"
   - 如果交易失败，更新状态为"Failed"

整个流程确保了私钥永远不会暴露在在线环境中，显著提高了安全性。

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

- **语言**: Go 1.16+ (项目已升级至 Go 1.24.0)
- **Web框架**: Gin 1.10.0
- **ORM库**: GORM 1.26.1
- **数据库**: SQLite 3
- **区块链客户端**: go-ethereum 1.15.11
- **认证**: JWT (JSON Web Token)
- **日志**: 自定义日志系统，基于ceyewan/clog 0.2.0

## 部署指南

### 环境要求
- Go 1.16+
- SQLite 3
- 以太坊节点访问（可使用Infura或其他服务）

### 安装步骤
1. 克隆代码库
   ```bash
   git clone https://github.com/your-username/crypto-custody.git
   cd crypto-custody/online-server
   ```

2. 安装依赖
   ```bash
   go mod tidy
   ```

3. 配置环境变量（可选）
   ```bash
   export ETH_NODE_URL="https://sepolia.infura.io/v3/YOUR_API_KEY"  # 以太坊节点URL
   export JWT_SECRET="your-secret-key"                              # JWT密钥
   ```

4. 构建应用
   ```bash
   go build -o online-server.bin
   ```

5. 运行应用
   ```bash
   ./online-server.bin
   ```

默认服务运行在 http://localhost:8080

### Docker 部署

本项目提供了通过 Docker 进行容器化部署的方案，以实现跨平台和标准化的部署。

#### 1. 构建并推送镜像

使用提供的 `docker-build-push.sh` 脚本可以方便地构建 `linux/amd64` 架构的镜像并推送到 Docker Hub。

```bash
# 赋予脚本执行权限
chmod +x docker-build-push.sh

# 运行脚本（使用你的 Docker Hub 用户名）
./docker-build-push.sh your-dockerhub-username
```

#### 2. 使用 Docker Compose 运行

`docker-compose.yml` 文件定义了如何运行服务。在运行前，请确保你有一个 `.env` 文件，其中包含所有必需的环境变量。

```bash
# 在后台启动服务
docker-compose up -d
```

服务将在 `http://localhost:8080` 上可用。数据库和日志文件将分别持久化到 `./database` 和 `./logs` 目录中。

## 改进小结

在项目重构过程中，我们进行了以下主要改进：

1. **添加DTO层**：创建了dto包，专门用于定义数据传输对象，使不同层之间的数据交换更加规范化。

2. **服务层优化**：
   - 设计了完整的服务层，包括TransactionService、AccountService和UserService
   - 所有服务都采用懒汉式加载和sync.Once确保线程安全和资源高效利用
   - 明确定义了每个服务的职责边界

3. **以太坊交互层重构**：
   - 优化client.go，使用懒汉式加载
   - 实现transaction_manager.go，完整管理交易生命周期
   - 移除了冗余代码和全局变量
   - 添加了详细的错误处理和状态管理

4. **处理器层职责划分**：
   - transaction_handler.go：处理与区块链直接交互的请求
   - account_handler.go：处理与账户和交易记录相关的请求
   - user_handler.go：处理用户管理相关请求
  
5. **路由与中间件优化**：
   - 路由结构更加清晰，映射到相应的处理函数
   - 增强中间件功能，包括认证、权限控制和日志记录

6. **资源利用改进**：
   - 避免在应用启动时初始化所有资源
   - 通过懒汉式加载实现按需初始化
   - 使用sync.Once确保并发安全

7. **文档完善**：
   - 添加了详细的API文档
   - 提供了开发指南
   - 包含了每个模块的README文件

这些改进使系统架构更加清晰，代码组织更加规范，不仅提高了可维护性和可测试性，还优化了资源利用效率和系统安全性。

## 贡献指南

欢迎贡献代码、报告问题或提出新功能建议。请遵循以下步骤：

1. Fork本项目
2. 创建你的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交你的更改 (`git commit -m '添加一些很棒的功能'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 提交Pull Request

### 代码风格

- 遵循Go官方的代码风格指南
- 所有公开函数和结构应有完整的注释
- 单元测试覆盖率应达到70%以上

## 许可证

本项目采用MIT许可证 - 详情请参阅 LICENSE 文件

## 联系方式

项目维护者: [Your Name] - your.email@example.com

项目链接: [https://github.com/your-username/crypto-custody](https://github.com/your-username/crypto-custody)

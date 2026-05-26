# 在线服务说明

本服务是加密资产托管系统的在线端，负责用户认证、权限控制、账户管理、交易准备、签名提交、交易广播和交易状态查询。系统采用在线与离线分离的交易流程，在线端只处理交易构建、签名验证和广播，不保存私钥。

## 核心能力

- 用户管理：注册、登录、登出、密码修改、管理员用户管理。
- 权限控制：支持 `admin`、`officer`、`guest` 三类角色。
- 账户管理：查询账户、创建账户、批量导入账户、管理员查看和删除账户。
- 交易管理：准备交易、提交签名并发送交易、查询交易列表和统计信息。
- 区块链交互：连接 Sepolia 测试网，查询余额、广播交易、检查交易收据。
- 数据持久化：使用 SQLite 保存用户、账户和交易记录。

## 目录结构

```text
online-server/
├── cmd/                   # 本地调试和辅助工具
├── docs/                  # API 文档和开发指南
├── dto/                   # 请求与响应数据结构
├── ethereum/              # 以太坊客户端和交易管理器
├── handler/               # HTTP 请求处理入口
├── middleware/            # Gin 中间件
├── model/                 # 数据库模型
├── route/                 # API 路由注册
├── service/               # 业务逻辑和数据访问
├── tests/                 # 接口测试和集成测试
├── utils/                 # 数据库、JWT、日志、响应等工具
├── API_DOCUMENTATION.md   # 汇总版接口文档
├── Dockerfile             # 容器镜像构建配置
├── docker-compose.yml     # 本地容器运行配置
├── go.mod                 # Go 依赖定义
└── main.go                # 服务启动入口
```

## 分层说明

### 辅助工具

`cmd/local-eth-tool` 提供本地以太坊测试链调试工具，可用于生成测试账户、给测试账户转账、生成待签名交易哈希，以及用普通私钥模拟离线签名结果。

### 路由层

`route` 目录负责注册 API 路径，并为不同接口挂载认证和角色权限中间件。

### 处理器层

`handler` 目录负责解析 HTTP 请求、调用服务层、组织统一响应。处理器不直接实现复杂业务规则。

### 服务层

`service` 目录负责业务逻辑、数据库读写和跨模块协调，例如用户认证、账户维护、交易记录更新等。

### 数据模型层

`model` 目录定义 GORM 模型，与 SQLite 表结构对应。

### DTO 层

`dto` 目录定义请求体、响应体和跨层传递的数据结构，避免直接暴露数据库模型。

### 以太坊交互层

`ethereum` 目录封装节点连接、余额查询、交易构建、签名验证、交易广播和确认检查。

### 工具层

`utils` 目录提供数据库初始化、JWT 生成与验证、密码哈希、日志和响应封装等通用能力。

## 角色权限

| 角色 | 主要权限 |
| --- | --- |
| `admin` | 用户管理、账户管理、交易查询和交易删除等管理能力 |
| `officer` | 账户创建/导入、交易准备、签名提交、交易列表和统计查询 |
| `guest` | 基础登录态能力和允许公开访问的查询 |

## 主要接口

### 用户

- `POST /api/login`：登录。
- `POST /api/register`：注册。
- `POST /api/check-auth`：检查令牌有效性。
- `GET /api/users/profile`：获取当前用户信息。
- `POST /api/users/logout`：登出。
- `POST /api/users/change-password`：修改密码。
- `GET /api/users/admin/users`：管理员获取用户列表。
- `PUT /api/users/admin/users/:id/role`：管理员更新用户角色。
- `PUT /api/users/admin/users/:id/password`：管理员修改用户密码。
- `DELETE /api/users/admin/users/:id`：管理员删除用户。

### 账户

- `GET /api/accounts/address/:address`：通过地址查询账户。
- `GET /api/accounts/officer/`：获取当前用户导入或创建的账户。
- `POST /api/accounts/officer/create`：创建账户。
- `POST /api/accounts/officer/import`：批量导入账户。
- `GET /api/accounts/admin/all`：管理员获取所有账户。
- `DELETE /api/accounts/admin/:id`：管理员删除账户。

### 交易

- `GET /api/transaction/balance/:address`：查询地址余额。
- `GET /api/transaction/:id`：查询交易详情。
- `GET /api/transaction/list`：查询交易列表。
- `GET /api/transaction/stats`：查询交易统计。
- `POST /api/transaction/tx/prepare`：准备交易。
- `POST /api/transaction/tx/sign-send`：提交签名并发送交易。
- `GET /api/transaction/admin/all`：管理员获取所有交易。
- `DELETE /api/transaction/admin/:id`：管理员删除交易。

更完整的字段说明见 `API_DOCUMENTATION.md` 和 `docs/` 目录。

## 交易流程

1. 在线端接收发送方、接收方和金额，查询 nonce 与 gas 信息。
2. 在线端生成待签名消息哈希，并保存交易记录。
3. 离线环境使用私钥对消息哈希签名。
4. 在线端接收签名，验证签名对应的地址。
5. 验证通过后广播交易，并更新交易状态。
6. 后续通过交易收据检查确认或失败状态。

## 环境变量

| 变量 | 用途 |
| --- | --- |
| `ETH_RPC` | Sepolia Infura 项目标识，服务会拼接为 `https://sepolia.infura.io/v3/<ETH_RPC>` |
| `JWT_SECRET_KEY` | JWT 签名密钥 |
| `DEFAULT_ADMIN_PASSWORD` | 首次初始化默认管理员账号时使用的密码 |
| `ENV` | 日志环境标识，设置为 `production` 时使用生产日志配置 |

服务启动时会读取 `.env` 文件；如果该文件不存在或变量缺失，启动或相关功能可能失败。

## 本地运行

```bash
go mod tidy
go run .
```

服务默认监听：

```text
http://localhost:8080
```

## 构建运行

```bash
go build -o online-server.bin
./online-server.bin
```

## Docker 运行

```bash
docker compose up -d
```

容器运行时需要准备好必要环境变量。数据库和日志文件会分别写入 `database/` 和 `logs/`。

## 测试

运行单元测试：

```bash
go test ./...
```

运行接口测试前需要先启动本地服务，并按测试目录 README 准备管理员密码、警员密码、测试账户和测试签名等参数。

```bash
cd tests/user_test
go test -v

cd ../account_test
go test -v

cd ../transaction_test
go test -v
```

交易测试会连接测试网络并可能真实广播交易，请只使用测试网络账户和测试资产。

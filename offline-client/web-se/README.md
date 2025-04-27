# Web-SE 安全芯片离线交易客户端

一个基于Go语言的离线交易客户端，结合安全芯片技术实现多方计算（MPC）密钥管理和交易签名功能。该项目旨在提供高安全性的加密货币资产管理解决方案。

## 📋 功能特点

- **多方计算（MPC）密钥生成**：支持门限签名，可配置阈值和参与方数量
- **交易签名验证**：安全进行交易数据签名
- **安全芯片集成**：使用安全芯片存储敏感密钥材料
- **加密保护**：采用AES-GCM加密算法保护密钥数据
- **日志记录**：详细的操作日志和错误追踪
- **RESTful API**：提供统一的HTTP接口

## 🛠️ 技术栈

- Go语言 (v1.24.0)
- Gin框架：Web服务器
- PCSC：智能卡通信
- Ethereum加密库：地址生成和签名验证
- Zap：高性能日志系统
- Viper：配置管理

## 🔧 安装与部署

### 前置要求

- Go 1.24.0 或更高版本
- PCSC兼容的智能卡读卡器
- 支持的安全芯片

### 获取代码

```bash
git clone [repository-url]
cd web-se
```

### 安装依赖

```bash
go mod download
```

### 编译项目

```bash
go build -o bin/web-se main.go
```

### 配置文件

项目根目录或`config`目录下创建`config.yaml`文件：

```yaml
# 基本配置
debug: true
port: "8080"
temp_dir: "./temp"
bin_dir: "./bin"
keygen_bin: "gg20_keygen"
signing_bin: "gg20_signing"
card_reader_name: ""  # 留空将自动选择第一个可用读卡器
manager_addr: "http://127.0.0.1:8000"

# 日志配置
log_dir: "./logs"
log_file: "web-se.log"
log_max_size: 10       # 单个日志文件最大大小(MB)
log_max_backups: 10    # 保留的旧日志文件数量
log_max_age: 30        # 日志文件保留天数
log_compress: true     # 是否压缩旧日志
```

### 运行服务

```bash
# 直接运行
./bin/web-se

# 或使用Air进行开发（热重载）
air
```

## 📚 API接口

### 1. 密钥生成

生成新的MPC密钥并存储到安全芯片中。

- **URL**: `/api/v1/mpc/keygen`
- **方法**: `POST`
- **请求体**:

```json
{
  "threshold": 2,
  "parties": 3,
  "index": 1,
  "filename": "key1.json",
  "username": "user1"
}
```

- **响应**:

```json
{
  "success": true,
  "userName": "user1",
  "address": "0x123abc...",
  "encryptedKey": "base64编码的加密密钥数据"
}
```

### 2. 消息签名

使用MPC密钥对消息进行签名。

- **URL**: `/api/v1/mpc/sign`
- **方法**: `POST`
- **请求体**:

```json
{
  "parties": "1,2,3",
  "data": "0x1234abcd...",
  "filename": "signature1.json",
  "encryptedKey": "base64编码的加密密钥",
  "userName": "user1",
  "address": "0x123abc...",
  "signature": "base64编码的安全芯片签名"
}
```

- **响应**:

```json
{
  "success": true,
  "signature": "0x9876fedc..."
}
```

### 3. 获取CPLC信息

获取安全芯片的CPLC（Card Production Life Cycle）信息。

- **URL**: `/api/v1/mpc/cplc`
- **方法**: `GET`
- **响应**:

```json
{
  "success": true,
  "cpic": "base64编码的CPLC数据"
}
```

### 4. 删除密钥

从安全芯片中删除指定密钥。

- **URL**: `/api/v1/mpc/delete`
- **方法**: `POST`
- **请求体**:

```json
{
  "username": "user1",
  "address": "0x123abc...",
  "signature": "base64编码的安全芯片签名"
}
```

- **响应**:

```json
{
  "success": true,
  "address": "0x123abc..."
}
```

## 🗂️ 项目结构

```
web-se/
├── main.go          # 程序入口
├── config/          # 配置管理
│   ├── config.go
│   └── config.yaml
├── controllers/     # API控制器
│   └── mpc.go
├── models/          # 数据模型
│   └── mpc.go
├── services/        # 业务逻辑
│   ├── mpc.go
│   └── security.go
├── utils/           # 工具函数
│   ├── crypto.go
│   ├── file.go
│   └── command.go
├── seclient/        # 安全芯片客户端
│   ├── cardreader.go
│   ├── commands.go
│   ├── constants.go
│   └── utils.go
├── clog/            # 日志系统
├── bin/             # 可执行文件
├── logs/            # 日志文件
└── temp/            # 临时文件
```

## 📝 日志系统

项目使用zap日志库，结合lumberjack进行日志分割和管理。日志分为DEBUG、INFO、WARN、ERROR、FATAL五个级别。

日志输出示例：
```
{"level":"INFO","ts":"2023-07-01T12:34:56.789Z","caller":"main.go:42","msg":"系统启动"}
{"level":"INFO","ts":"2023-07-01T12:34:56.790Z","caller":"main.go:43","msg":"配置加载成功","port":"8080","debug":true,"log_file":"web-se.log","log_dir":"./logs"}
```

## ⚙️ 开发与调试

### 使用Air进行热重载

项目包含`.air.toml`配置文件，支持使用Air工具进行开发时的热重载。

```bash
# 安装Air (如果尚未安装)
go install github.com/cosmtrek/air@latest

# 启动开发模式
air
```

### 调试模式

在`config.yaml`中设置`debug: true`启用调试模式，会输出更详细的日志信息，包括安全芯片通信数据。

## 🔐 安全注意事项

1. 确保配置文件中的敏感信息得到妥善保护
2. 生产环境建议关闭调试模式
3. 定期备份配置和日志
4. 使用安全的网络环境部署服务

## 📄 许可证

[许可证信息]

## 📧 联系方式

[联系方式信息]

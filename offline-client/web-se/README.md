# MPC Web服务

该项目是一个基于Go+GIN框架的Web服务，用于封装MPC密钥生成和签名操作，支持与安全芯片集成。

## 功能特点

- 封装MPC密钥生成命令 `gg20_keygen`
- 封装MPC签名命令 `gg20_signing`
- 使用安全芯片存储密钥
- 提供RESTful API接口
- 支持密钥的安全存储和传输

## 环境要求

- Go 1.21+
- GIN框架
- 安全芯片接口库
- MPC相关执行文件

## 配置文件

配置文件位于`config/config.yaml`，包含以下配置项：

```yaml
debug: true                # 调试模式
port: "8080"               # 服务端口
temp_dir: "./temp"         # 临时文件目录
bin_dir: "./bin"           # 可执行文件目录
keygen_bin: "gg20_keygen"  # 密钥生成可执行文件
signing_bin: "gg20_signing" # 签名可执行文件
security_chip: true        # 是否启用安全芯片
```

## 安装与使用

1. 克隆项目
```bash
git clone https://github.com/yourusername/mpc-web.git
cd mpc-web
```

2. 安装依赖
```bash
go mod tidy
```

3. 构建项目
```bash
go build -o web-se
```

4. 运行服务
```bash
./web-se
```

## API接口说明

### 1. 密钥生成

**请求**

```
POST /api/v1/mpc/keygen
```

**请求参数**

```json
{
  "threshold": 1,         // 门限值t
  "parties": 3,           // 参与方总数n
  "index": 1,             // 当前参与方序号i
  "filename": "share.json", // 输出文件名
  "userName": "alice"     // 用户名
}
```

**响应**

```json
{
  "success": true,
  "address": "0x123...",   // 生成的以太坊地址
  "encryptedKey": "base64..." // 加密后的密钥数据
}
```

### 2. 消息签名

**请求**

```
POST /api/v1/mpc/sign
```

**请求参数**

```json
{
  "parties": "1,2",       // 参与方
  "data": "hello",        // 要签名的数据
  "filename": "share.json", // 密钥文件名
  "encryptedKey": "base64...", // 加密后的密钥数据
  "userName": "alice",    // 用户名
  "address": "0x123...",  // 地址
  "signature": "base64..." // 签名
}
```

**响应**

```json
{
  "success": true,
  "signature": "0x456..."  // 签名结果
}
```

## 目录结构

```
├── config/               # 配置相关
│   ├── config.go         # 配置加载
│   └── config.yaml       # 配置文件
├── controllers/          # 控制器
│   └── mpc.go            # MPC控制器
├── middleware/           # 中间件
│   └── error.go          # 错误处理中间件
├── models/               # 数据模型
│   └── mpc.go            # MPC相关模型
├── seclient/             # 安全芯片客户端
│   ├── cardreader.go     # 读卡器实现
│   ├── commands.go       # 命令实现
│   ├── constants.go      # 常量定义
│   └── utils.go          # 工具函数
├── services/             # 服务层
│   ├── mpc.go            # MPC服务
│   └── security.go       # 安全服务
├── utils/                # 工具函数
│   ├── command.go        # 命令执行
│   ├── crypto.go         # 加密相关
│   └── file.go           # 文件操作
├── bin/                  # 可执行文件目录
│   ├── gg20_keygen       # 密钥生成程序
│   └── gg20_signing      # 签名程序
├── temp/                 # 临时文件目录
├── go.mod                # Go模块文件
├── go.sum                # Go依赖校验文件
├── main.go               # 主程序入口
└── README.md             # 项目说明
```

## 安全说明

- 临时文件使用后会被删除
- 密钥使用安全芯片存储，不直接保存在磁盘上
- 传输的密钥数据使用AES-GCM加密
- 用户需提供签名才能读取安全芯片中的数据

## 扩展和定制

该项目设计为模块化结构，便于扩展和定制：

1. 可以通过配置文件更改可执行文件路径
2. 安全芯片集成可以通过`security_chip`配置项禁用
3. 临时文件路径可配置
4. 所有服务都是接口化设计，便于替换实现

## 许可证

MIT 

# 密钥生成服务测试脚本

这个脚本用于测试 2-3 门限签名的密钥生成服务。

## 环境要求

- Python 3.6+
- requests 库

## 安装依赖

```bash
pip install -r requirements.txt
```

## 使用方法

1. 确保密钥生成服务已经启动并运行在 `http://localhost:8080`（如果不是，请修改 `test_keygen.py` 中的 `SERVER_URL`）

2. 运行测试脚本：

```bash
python test_keygen.py
```

## 测试说明

- 脚本会模拟 3 个参与方（index 从 1 到 3）的密钥生成过程
- 每个参与方会生成自己的密钥份额
- 生成的密钥信息会保存在 `keygen_result.json` 文件中
- 每个请求之间会有 1 秒的延迟，以确保服务有足够时间处理请求

## 输出说明

- 成功响应会显示 "密钥生成成功！"
- 失败响应会显示具体的错误信息
- 所有响应都会打印完整的请求和响应信息 

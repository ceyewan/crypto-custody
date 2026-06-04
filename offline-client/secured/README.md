# 安全芯片存储 Applet

一个运行在 JavaCard 智能卡平台上的数据存储应用，提供安全的数据存储和检索功能。

本文档为**用户**提供快速入门指南。设计边界请参阅 [SECURITY_CHIP_DESIGN.md](SECURITY_CHIP_DESIGN.md)，技术细节请参阅 [DEVELOPMENT.md](DEVELOPMENT.md)。

## 功能特性

- **数据存储**: 存储 `record_id` (32字节)、地址 (20字节) 和消息 (32字节)。
- **安全检索**: 通过 `record_id` 和地址查询数据。
- **ECDSA 签名验证**: 读取和删除操作需要签名验证，防止未经授权的访问。
- **存储管理**: 支持数据覆盖和删除，最多可存储 100 条记录。

## 快速开始

请按照以下步骤编译、部署和测试 Applet。

### 1. 环境准备

确保您的系统已安装以下工具：

- **Java JDK 11**: 用于编译 JavaCard 应用。当前 JavaCard 3.0.5 工具链不建议使用 JDK 17+。
- **Apache Ant**: 一个用于自动化构建流程的工具。
- **Python 3.11**: 用于运行 Goodix `pygse` 下载工具链。
- **pygse**: 一个由汇顶 (Goodix) 提供的 Python 工具，用于将编译后的 `.cap` 文件安装到安全芯片中。

> 注意：不要直接使用 Homebrew 默认的 `python3`，如果它是 Python 3.14 或更新版本，`pygse 2.1.5` 和 `pyscard` 容易出现兼容性问题。当前已验证组合是 `python@3.11 + pygse==2.1.5 + pyscard==2.2.2`。

#### **安装 Apache Ant**

- **macOS (使用 Homebrew):**
  ```bash
  brew install ant openjdk@11 swig python@3.11
  ```

- **Debian/Ubuntu (使用 apt):**
  ```bash
  sudo apt-get update
  sudo apt-get install ant
  ```

- **Windows:**
  1.  从 [Apache Ant 官网](https://ant.apache.org/bindownload.cgi) 下载二进制 `.zip` 文件。
  2.  解压到一个固定位置，例如 `C:\apache-ant`。
  3.  将 `C:\apache-ant\bin` 添加到系统的 `Path` 环境变量中。

安装完成后，运行 `ant -version` 验证是否成功。

#### **安装 pygse**

Goodix 工具包位于 `tools/` 目录。macOS 推荐使用独立 Python 3.11 虚拟环境：

```bash
cd offline-client/secured/tools
/opt/homebrew/opt/python@3.11/bin/python3.11 -m venv .venv311
source .venv311/bin/activate
python -m pip install -U pip setuptools wheel
python -m pip install -U ./pygse-2.1.5-py3-none-any.whl ./gpqc-1.0.1-py3-none-any.whl
python -m pip install --force-reinstall "pyscard==2.2.2"
pygse ls-dev
```

之后也可以不激活虚拟环境，直接使用：

```bash
offline-client/secured/tools/.venv311/bin/pygse ls-dev
```

**重要提示**: `lib` 目录包含了构建所需的 JavaCard SDK (`jc305u4_kit`) 和 Ant 扩展 (`ant-javacard.jar`)。请确保此目录与 `secured` 目录位于同一父目录下。

### 2. 构建与部署流程

以下命令默认从仓库根目录执行。

#### 步骤 1: 生成密钥对

```bash
cd offline-client/secured/genkey
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out ec_private_key.pem
openssl ec -in ec_private_key.pem -pubout -conv_form uncompressed -outform DER | tail -c 65 > ec_public_key.bin
```
以上命令将生成 `ec_private_key.pem` (私钥) 和 `ec_public_key.bin` (公钥)。如果本机已经安装 Python `cryptography` 包，也可以运行 `python generate_keys.py` 生成同等格式的密钥对。

**注意**: 每次生成新的公钥后，您都需要将其内容更新到 `src/securitychip/SecurityChipApplet.java` 文件中的 `EC_PUBLIC_KEY_BYTES` 变量中，然后重新编译 Applet。私钥需要部署到离线服务端 `offline-server/private_keys/ec_private_key.pem`，不要提交到 Git。

#### 步骤 2: 编译 Applet

返回项目根目录 (`secured`) 并使用 Ant 构建项目。
```bash
cd offline-client/secured
JAVA_HOME=/opt/homebrew/opt/openjdk@11 ant clean all
```
生成的 `.cap` 文件位于 `build/cap/securitychip.cap`。

#### 步骤 3: 部署到安全芯片

**现场固定读卡器**: 下载 CAP 时必须使用 `GOODIX GSE SmartCard Reader`。如果 `pygse ls-dev` 同时列出 `GOODIX GSE SmartCard Reader` 和 `GOODIX GSE SmartCard Reader 01`，不要使用带 `01` 的设备名，否则 CAP 会安装到另一颗 SE 上。

```bash
cd offline-client/secured
tools/.venv311/bin/pygse ls-dev
tools/.venv311/bin/pygse install \
  --dev "GOODIX GSE SmartCard Reader" \
  --app-aid=. \
  build/cap/securitychip.cap \
  --log-level info
```

如果同一 AID 已经安装过，`pygse install` 会删除旧包并重新安装，芯片内旧记录也会被清空。部署生产 SE 前需要确认该 SE 不再需要保留旧数据。

### 3. 运行 SE smoke 测试

SE smoke 测试已经迁移到桌面端 MPC core，直接复用生产路径的 `mpc_core/seclient` 和 `SecurityService`，避免测试代码和真实调用链不一致。

```bash
cd offline-client/offline-client-wails
go run ./mpc_core/cmd/se-smoke
```

指定读卡器或私钥路径：

```bash
go run ./mpc_core/cmd/se-smoke -reader "GOODIX GSE SmartCard Reader"
go run ./mpc_core/cmd/se-smoke -private-key ../secured/genkey/ec_private_key.pem
```

该测试会覆盖：

- 直接 `mpc_core/seclient`：连接读卡器、读取 CPLC、选择 Applet、存储、读取、删除、更新、无效签名、错误输入、清理测试记录。
- `SecurityService`：按桌面端真实调用方式执行 `StoreData`、`ReadData`、`DeleteData`，验证 `record_id`/地址解析和授权签名链路。

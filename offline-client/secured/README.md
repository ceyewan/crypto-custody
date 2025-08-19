# 安全芯片存储 Applet

一个运行在 JavaCard 智能卡平台上的数据存储应用，提供安全的数据存储和检索功能。

本文档为**用户**提供快速入门指南。如需了解技术细节，请参阅 [DEVELOPMENT.md](DEVELOPMENT.md)。

## 功能特性

- **数据存储**: 存储用户名 (32字节)、地址 (20字节) 和消息 (32字节)。
- **安全检索**: 通过用户名和地址查询数据。
- **ECDSA 签名验证**: 读取和删除操作需要签名验证，防止未经授权的访问。
- **存储管理**: 支持数据覆盖和删除，最多可存储 100 条记录。

## 快速开始

请按照以下步骤编译、部署和测试 Applet。

### 1. 环境准备

确保您的系统已安装以下工具：

- **Java JDK 8+**: 用于编译 JavaCard 应用。
- **Apache Ant**: 一个用于自动化构建流程的工具。
- **Python 3.x**: 用于运行密钥生成脚本。
- **pygse**: 一个由汇顶 (Goodix) 提供的 Python 工具，用于将编译后的 `.cap` 文件安装到安全芯片中。

#### **安装 Apache Ant**

- **macOS (使用 Homebrew):**
  ```bash
  brew install ant
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

```bash
pip install pygse (实际需要汇顶提供的 pygse 包)
```

**重要提示**: `lib` 目录包含了构建所需的 JavaCard SDK (`jc305u4_kit`) 和 Ant 扩展 (`ant-javacard.jar`)。请确保此目录与 `secured` 目录位于同一父目录下。

### 2. 构建与部署流程

#### 步骤 1: 生成密钥对

```bash
cd genkey
python generate_keys.py
```
此脚本将生成 `ec_private_key.pem` (私钥) 和 `ec_public_key.bin` (公钥)。

**注意**: 每次生成新的公钥后，您都需要将其内容手动更新到 `src/securitychip/SecurityChipApplet.java` 文件中的 `EC_PUBLIC_KEY_BYTES` 变量中，然后重新编译 Applet。

#### 步骤 2: 编译 Applet

返回项目根目录 (`secured`) 并使用 Ant 构建项目。
```bash
cd ..
ant
```
生成的 `.cap` 文件位于 `build/cap/securitychip.cap`。

#### 步骤 3: 部署到安全芯片

```bash
pygse install build/cap/securitychip.cap
```

### 3. 运行测试客户端

```bash
cd test/go
go run main.go
```
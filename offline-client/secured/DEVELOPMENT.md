# 安全芯片 Applet 开发者文档

本文档为希望深入了解、修改或集成此 JavaCard Applet 的开发者提供详细的技术信息。

## 目录

1.  [项目结构](#1-项目结构)
2.  [构建流程详解](#2-构建流程详解)
3.  [APDU 通信协议](#3-apdu-通信协议)
4.  [签名工作流程](#4-签名工作流程)
5.  [安全机制：ECDSA 签名](#5-安全机制ecdsa-签名)
6.  [内部实现细节](#6-内部实现细节)
7.  [测试与验证](#7-测试与验证)

---

## 1. 项目结构

```
crypto-custody/offline-client/
├── lib/
│   ├── ant-javacard.jar         # Ant 构建扩展
│   └── jc305u4_kit/             # JavaCard 3.0.5 u4 SDK
└── secured/
    ├── build.xml                # Ant 构建脚本
    ├── README.md                # 用户快速入门指南
    ├── DEVELOPMENT.md           # 开发者文档 (本文)
    ├── build/                   # 编译输出目录
    │   ├── cap/
    │   │   └── securitychip.cap # 最终的 Applet 文件
    │   └── classes/
    │       └── securitychip/
    ├── genkey/                  # 密钥生成工具
    │   ├── generate_keys.py     # Python 密钥生成脚本
    │   ├── ec_private_key.pem   # [输出] ECDSA 私钥
    │   └── ec_public_key.bin    # [输出] ECDSA 公钥
    ├── src/                     # Java 源代码
    │   └── securitychip/
    │       └── SecurityChipApplet.java
    └── test/                    # 测试入口说明
        └── go/                  # 指向 mpc_core/cmd/se-smoke 的 README
```

---

## 2. 构建流程详解

项目使用 **Apache Ant** 和 **ant-javacard** 插件进行构建。核心配置文件是 `build.xml`。

### 已验证的本地工具链

当前本地已验证的组合如下：

| 工具 | 版本要求 | 说明 |
| :--- | :--- | :--- |
| JDK | `openjdk@11` | JavaCard 3.0.5 工具链在 JDK 17+ 下可能失败 |
| Ant | Homebrew 或系统包管理器安装 | 调用 `build.xml` 生成 CAP |
| Python | `python@3.11` | 用于运行 Goodix `pygse` |
| pygse | `2.1.5` | 使用 `tools/pygse-2.1.5-py3-none-any.whl` |
| pyscard | `2.2.2` | `pygse 2.1.5` 与 `pyscard 2.3.x` 存在 API 不兼容 |

不建议用 Homebrew 默认 `python3` 直接安装 `pygse`。如果默认版本是 Python 3.14，可能出现 `pyscard` 构建或运行问题；如果自动装到 `pyscard 2.3.x`，可能出现 `Session.__init__()` 参数不兼容。

推荐初始化命令：

```bash
cd offline-client/secured/tools
/opt/homebrew/opt/python@3.11/bin/python3.11 -m venv .venv311
source .venv311/bin/activate
python -m pip install -U pip setuptools wheel
python -m pip install -U ./pygse-2.1.5-py3-none-any.whl ./gpqc-1.0.1-py3-none-any.whl
python -m pip install --force-reinstall "pyscard==2.2.2"
pygse ls-dev
```

编译 Applet 时固定 JDK 11：

```bash
cd offline-client/secured
JAVA_HOME=/opt/homebrew/opt/openjdk@11 ant clean all
```

部署 Applet 示例：

```bash
tools/.venv311/bin/pygse ls-dev
tools/.venv311/bin/pygse info --dev "GOODIX GSE SmartCard Reader 01"
tools/.venv311/bin/pygse install \
  --dev "GOODIX GSE SmartCard Reader 01" \
  --app-aid=. \
  build/cap/securitychip.cap \
  --log-level info
```

安装同一个 Package/Applet AID 时会覆盖旧 Applet，并清除该 Applet 下已有记录。对真实 SE 操作前，先确认目标读卡器名称和 CPLC。

### `build.xml` 关键配置

- **`jcdk.home`**: 指向 `../lib/jc305u4_kit`，即 JavaCard SDK 的位置。
- **`ant-javacard.jar`**: 指向 `../lib/ant-javacard.jar`，这是 Ant 的一个任务扩展，提供了 `<javacard>` 任务。
- **`package.aid` / `applet.aid`**: 定义了 Applet 的唯一标识符 (AID)。
- **Targets**:
    - `compile`: 使用 `javac` 编译 `src/` 目录下的 `.java` 文件。
    - `convert`: 使用 `<javacard>` 任务将 `.class` 文件转换为 `.cap` 文件。这是 JavaCard 开发的核心步骤。
    - `all`: 默认任务，执行完整的编译和转换流程。

### 手动更新公钥

在 `src/securitychip/SecurityChipApplet.java` 中，`EC_PUBLIC_KEY_BYTES` 常量硬编码了用于签名验证的公钥。当 `genkey/generate_keys.py` 生成新的密钥对时，必须将新的公钥 (`ec_public_key.bin` 的内容) 复制到此常量中，然后重新运行 `ant` 来构建包含新公钥的 Applet。

---

## 3. APDU 通信协议

APDU (Application Protocol Data Unit) 是与 JavaCard Applet 通信的标准格式。

### APDU 基础结构

| 字段 | 长度 (字节) | 描述 |
| :--- | :--- | :--- |
| CLA | 1 | 类别字节，本项目固定为 `0x80` |
| INS | 1 | 指令字节，标识具体操作 |
| P1, P2 | 1, 1 | 参数字节，本项目中保留为 `0x00` |
| Lc | 1 | 命令数据长度 |
| Data | Lc | 命令数据 |
| Le | 1 | 期望响应数据长度 |

### 支持的指令

| 指令 | INS | 功能 |
| :--- | :--- | :--- |
| `STORE_DATA` | `0x10` | 存储或覆盖一条记录 |
| `READ_DATA` | `0x20` | 读取一条记录 (需签名) |
| `DELETE_DATA` | `0x30` | 删除一条记录 (需签名) |

---

### 指令详解

#### **A. `STORE_DATA` (INS: 0x10)**

存储一条记录。如果记录已存在 (基于 `record_id` 和 `addr`)，则覆盖。

- **请求 (Data)**:
  `[record_id(32 bytes)][addr(20 bytes)][message(32 bytes)]`
  - `Lc` = 84 (0x54)
- **响应 (Data)**:
  `[recordIndex(1 byte)][recordCount(1 byte)]`
- **状态码 (SW)**:
  - `0x9000`: 成功
  - `0x6700`: 数据长度错误
  - `0x6A84`: 存储空间已满

#### **B. `READ_DATA` (INS: 0x20)**

根据 `record_id` 和 `addr` 读取一条记录。需要提供有效签名。

- **请求 (Data)**:
  `[record_id(32 bytes)][addr(20 bytes)][signature(variable length)]`
  - `Lc` > 52
- **响应 (Data)**:
  `[message(32 bytes)]`
- **状态码 (SW)**:
  - `0x9000`: 成功
  - `0x6A83`: 记录未找到
  - `0x6982`: 签名验证失败

#### **C. `DELETE_DATA` (INS: 0x30)**

根据 `record_id` 和 `addr` 删除一条记录。需要提供有效签名。

- **请求 (Data)**:
  `[record_id(32 bytes)][addr(20 bytes)][signature(variable length)]`
  - `Lc` > 52
- **响应 (Data)**:
  `[deletedIndex(1 byte)][recordCount(1 byte)]`
- **状态码 (SW)**:
  - `0x9000`: 成功
  - `0x6A83`: 记录未找到
  - `0x6982`: 签名验证失败

---

## 4. 签名工作流程

本 Applet 的安全模型涉及三方：**客户端**、一个**离线签名服务器**和**安全芯片**。私钥被严格保管在离线服务器上，确保其安全性。

- **公钥**: 硬编码在 `SecurityChipApplet.java` 中，部署到安全芯片上，用于**验证**签名。
- **私钥**: 存储在安全的离线服务器上，用于对敏感操作（读取、删除）的请求数据进行**签名**。

### 交互流程图

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant OfflineServer as 离线签名服务器 <br> (持有私钥)
    participant SecurityChip as 安全芯片 <br> (持有公钥)

    Client->>+OfflineServer: 请求读取/删除数据 (含 record_id, addr)
    Note right of OfflineServer: 1. 准备签名数据 <br> data = record_id (32B) + addr (20B)
    OfflineServer->>OfflineServer: 2. 使用私钥对数据进行签名 <br> signature = sign(data)
    Note right of OfflineServer: 3. 构造完整的 APDU 命令 <br> apdu = [HEADER][data][signature]
    OfflineServer->>+SecurityChip: 4. 发送 APDU 命令
    Note left of SecurityChip: 5. 使用内置公钥验证签名 <br> isValid = verify(data, signature)
    alt 签名有效
        SecurityChip->>SecurityChip: 6. 执行读取/删除操作
        SecurityChip-->>-OfflineServer: 7. 返回操作结果
    else 签名无效
        SecurityChip-->>-OfflineServer: 7. 返回错误码 (0x6982)
    end
    OfflineServer-->>-Client: 8. 返回最终结果
```

---

## 5. 安全机制：ECDSA 签名

为防止未经授权的数据读取和删除，这些操作需要通过 ECDSA 签名进行验证。本章节将详细说明签名的技术细节。

### 5.1 签名参数

- **曲线**: `secp256r1` (NIST P-256)
- **哈希算法**: SHA-256
- **签名格式**: **DER 编码**。这是一个符合 X.509 标准的变长格式。对于 P-256 曲线，签名长度通常在 70 到 72 字节之间。

### 5.2 签名作用的数据 (Message to be Signed)

签名的核心是确保请求的完整性和来源。签名操作**不是**对整个 APDU 命令进行的，而是针对请求中最关键的识别信息。

- **待签名数据**: `record_id` (32字节) 和 `addr` (20字节) 的**直接二进制拼接**。
- **总长度**: 32 + 20 = 52 字节。

**示例:**
假设 `record_id` 的字节表示为 `R_BYTES`，`addr` 的字节表示为 `A_BYTES`，那么离线服务器需要计算的签名是：
`signature = sign(SHA256(R_BYTES || A_BYTES))`
其中 `||` 代表字节数组的拼接。

### 5.3 APDU 命令中的数据布局

在构造 `READ_DATA` 或 `DELETE_DATA` 的 APDU 命令时，数据字段 (`Data`) 由三部分组成：

`[record_id (32 bytes)][addr (20 bytes)][signature (variable length)]`

安全芯片在收到此命令后，会：
1.  提取前 52 字节作为待验证的原始数据。
2.  提取剩余字节作为签名。
3.  使用芯片内存储的公钥执行 ECDSA 验证。

桌面端生产 `seclient` 中的以下代码片段清晰地展示了这一数据布局：

```go
// offline-client-wails/mpc_core/seclient/commands.go

// ... 构造待发送的完整数据
fullData := make([]byte, 0, RECORD_ID_LENGTH+ADDR_LENGTH+len(signature))
fullData = append(fullData, recordID...)
fullData = append(fullData, addr...)
fullData = append(fullData, signature...)

// ... 构建APDU命令
command := []byte{CLA, INS_READ_DATA, 0x00, 0x00, byte(len(fullData))}
command = append(command, fullData...)
```

### 5.4 客户端签名实现 (Python 示例)

```python
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.serialization import load_pem_private_key

# 1. 加载 PEM 格式的私钥
with open('ec_private_key.pem', 'rb') as f:
    private_key = load_pem_private_key(f.read(), password=None)

# 2. 准备要签名的数据 (record_id + addr)
data_to_sign = b'\xDE\xAD\xBE\xEF...' # 52 bytes of data

# 3. 使用 SHA-256 和 ECDSA 生成 DER 格式的签名
signature = private_key.sign(
    data_to_sign,
    ec.ECDSA(hashes.SHA256())
)

# signature 变量现在包含了可发送给 Applet 的 DER 编码签名
```

---

## 6. 内部实现细节

- **存储模型**: Applet 内部维护一个包含 100 个槽位的数组来存储记录。
- **记录结构**: 每条记录包含 `record_id`, `addr`, `message` 和一个 `isUsed` 标志。
- **空间重用**: 当一条记录被删除时，其 `isUsed` 标志被设为 `false`。该槽位可以被后续的 `STORE_DATA` 操作重用，从而避免了碎片化。
- **数据填充**: 所有传入的数据字段必须符合固定的长度。如果数据不足，客户端有责任进行填充（例如，使用 `0x00`）。

---

## 7. 测试与验证

- **SE smoke 测试**: 使用 `offline-client/offline-client-wails/mpc_core/cmd/se-smoke`。该命令直接复用桌面端生产 `mpc_core/seclient`，并额外覆盖 `SecurityService`，避免测试客户端和真实调用链漂移。
- **预期状态码**:
  - 成功: `0x9000`
  - 未找到: `0x6A83`
  - 签名失败: `0x6982`
  - 空间已满: `0x6A84`

运行方式：

```bash
cd offline-client/offline-client-wails
go run ./mpc_core/cmd/se-smoke
```

常用参数：

```bash
go run ./mpc_core/cmd/se-smoke -reader "GOODIX GSE SmartCard Reader 01"
go run ./mpc_core/cmd/se-smoke -private-key ../secured/genkey/ec_private_key.pem
go run ./mpc_core/cmd/se-smoke -skip-direct
go run ./mpc_core/cmd/se-smoke -skip-service
```

# 安全芯片存储 Applet 使用指南

## 概述

该项目包含两个主要部分：
1. **安全芯片存储 Applet**：运行在 JavaCard 智能卡平台上的程序，提供数据存储和检索功能。
2. **seclient 客户端**：基于 Go 语言开发的客户端程序，用于与安全芯片进行交互，发送 APDU 指令并处理响应。

主要功能包括：
- 存储固定长度的用户名(32字节)、地址(20字节)和消息数据(32字节)
- 通过用户名和地址检索存储的消息数据
- 支持覆盖已存在的(userName, Addr)对的数据
- 支持删除已存在的(userName, Addr)对的数据

## 项目结构

```
crypto-custody/offline-client/secured
├── build.xml                # Ant 构建脚本
├── README.md                # 项目说明文档
├── build/                   # 编译生成的文件
│   ├── cap/                 # CAP 文件目录
│   │   └── securitychip.cap # 安全芯片 Applet 的 CAP 文件
│   └── classes/             # 编译生成的类文件
│       └── securitychip/
│           └── SecurityChipApplet.class
├── genkey/                  # 密钥生成工具
│   ├── ec_private_key.pem   # ECDSA 私钥
│   ├── ec_public_key.bin    # ECDSA 公钥
│   └── generate_keys.py     # 密钥生成脚本
├── src/                     # 源代码目录
│   └── securitychip/
│       └── SecurityChipApplet.java # 安全芯片 Applet 源代码
└── test/                    # 测试目录
    ├── go/                  # Go 客户端代码
    │   ├── go.mod           # Go 模块配置文件
    │   ├── go.sum           # Go 依赖锁定文件
    │   ├── main.go          # 客户端主程序
    │   └── seclient/        # 客户端核心逻辑
    │       ├── cardreader.go
    │       ├── commands.go
    │       ├── constants.go
    │       └── utils.go
```

## 编译与运行

### 1. 编译安全芯片 Applet

使用 Ant 工具编译和安装 Applet：

```bash
ant
```

- 任务会编译 Java 源代码并生成 CAP 文件。

### 2. 运行密钥生成脚本

在客户端与安全芯片交互前，需要生成 ECDSA 密钥对：

```bash
cd genkey
python generate_keys.py
```

- 生成的 `ec_private_key.pem` 用于客户端签名。
- 生成的 `ec_public_key.bin` 需要嵌入到 Applet 中。

### 3. 运行 Go 客户端

进入 `test/go` 目录，运行客户端程序：

```bash
cd test/go
go run main.go
```

客户端会通过 APDU 指令与安全芯片交互，支持存储、读取和删除数据操作。

## APDU 通信基础

APDU (Application Protocol Data Unit) 是智能卡与主机之间的通信单元。每个 APDU 命令包含以下部分:

| 字段 | 长度 | 描述 |
|------|------|------|
| CLA | 1字节 | 类别字节，标识命令类别 |
| INS | 1字节 | 指令字节，标识具体操作 |
| P1, P2 | 各1字节 | 参数字节，提供额外信息 |
| Lc | 1字节 | 命令数据长度 |
| 命令数据 | Lc字节 | 实际数据内容 |
| Le | 0-3字节 | 期望响应长度 |

## 支持的指令

本 Applet 支持三种基本指令:

| 指令名称 | INS值 | 描述 |
|---------|------|------|
| STORE_DATA | 0x10 | 存储新记录或更新现有记录 |
| READ_DATA | 0x20 | 读取现有记录 |
| DELETE_DATA | 0x30 | 删除现有记录 |

## 1. 存储数据指令 (STORE_DATA)

### 请求格式

```
CLA: 0x80 (默认JavaCard应用类)
INS: 0x10 (存储数据指令)
P1: 0x00 (保留)
P2: 0x00 (保留)
Lc: 0x54 (总数据长度: 32+20+32=84字节)
数据: [userName(32字节)][addr(20字节)][message(32字节)]
```

### 数据字段说明

- `userName`: 固定32字节，用户名标识
- `addr`: 固定20字节，以太坊地址数据
- `message`: 固定32字节，要存储的消息数据

### 响应格式

成功时返回2字节数据:
- 字节0: 记录索引 (0-99)
- 字节1: 当前存储的记录总数

### 错误码

| 状态码 | 描述 |
|-------|------|
| 0x9000 | 成功 |
| 0x6700 | 错误的数据长度 |
| 0x6A84 | 存储空间已满 |

### 示例 (以十六进制表示)

```
>> 80 10 00 00 80 
   [32字节userName] [20字节addr] [32字节message]

<< [记录索引] [记录总数] 90 00
```

## 2. 读取数据指令 (READ_DATA)

### 请求格式

```
CLA: 0x80 (默认JavaCard应用类)
INS: 0x20 (读取数据指令)
P1: 0x00 (保留)
P2: 0x00 (保留)
Lc: 变长 (至少96字节: 32+20=96)
数据: [userName(32字节)][addr(20字节)][sign(可变长度)]
```

### 数据字段说明

- `userName`: 固定32字节，要查找的用户名
- `addr`: 固定20字节，要查找的地址
- `sign`: 可变长度，签名数据（读取时可选，不参与查找过程）

### 响应格式

成功时返回32字节的消息数据。

### 错误码

| 状态码 | 描述 |
|-------|------|
| 0x9000 | 成功 |
| 0x6700 | 错误的数据长度 |
| 0x6A83 | 记录未找到 |

### 示例 (以十六进制表示)

```
>> 80 20 00 00 60 
   [32字节userName] [20字节addr]

<< [32字节message] 90 00
```

## 3. 删除数据指令 (DELETE_DATA)

### 请求格式

```
CLA: 0x80 (默认JavaCard应用类)
INS: 0x30 (删除数据指令)
P1: 0x00 (保留)
P2: 0x00 (保留)
Lc: 0x60 (总数据长度: 32+20=96字节)
数据: [userName(32字节)][addr(20字节)]
```

### 数据字段说明

- `userName`: 固定32字节，要删除的记录的用户名
- `addr`: 固定20字节，要删除的记录的地址

### 响应格式

成功时返回2字节数据:
- 字节0: 被删除的记录索引 (0-99)
- 字节1: 删除后剩余的记录总数

### 错误码

| 状态码 | 描述 |
|-------|------|
| 0x9000 | 成功 |
| 0x6700 | 错误的数据长度 |
| 0x6A83 | 记录未找到 |

### 示例 (以十六进制表示)

```
>> 80 30 00 00 60 
   [32字节userName] [20字节addr]

<< [记录索引] [剩余记录数] 90 00
```

## 性能与容量限制

- 最大记录数: 100条
- 用户名长度: 固定32字节
- 地址长度: 固定20字节 
- 消息长度: 固定32字节

## 内部实现说明

- 记录存储采用槽位管理机制，支持删除后重用空间
- 使用标记数组记录每个槽位的使用状态，提高存储效率
- 删除记录时不会物理删除数据，仅标记为未使用状态
- 新增记录时优先使用已删除的空闲槽位

## 最佳实践

1. **错误处理**: 始终检查响应状态码，确保操作成功完成
   
2. **重复键处理**: 当存储具有相同(userName, addr)的记录时，新数据将覆盖旧数据

3. **数据填充**: 如果实际数据未达到固定长度，需要进行填充:
   - 字符串数据建议使用空字节(0x00)右填充
   - 二进制数据可考虑使用0xFF或0x00填充

4. **数据格式化**: 确保所有字段严格按照指定的字节长度发送

5. **删除后重新存储**: 删除记录后，可立即在相同位置存储新数据

## 开发示例

### Go客户端代码示例

```go
// DeleteData 删除数据
func (r *CardReader) DeleteData(username string, addr []byte) error {
    // 确保输入数据符合长度要求
    usernameBytes := usernameToBytes(username)
    addrBytes := ensureAddrLength(addr)

    // 构造完整数据
    fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH)
    fullData = append(fullData, usernameBytes...)
    fullData = append(fullData, addrBytes...)

    // 构建APDU命令
    command := []byte{0x80, 0x30, 0x00, 0x00, byte(len(fullData))}
    command = append(command, fullData...)

    // 发送命令
    resp, err := r.card.Transmit(command)
    if err != nil {
        return fmt.Errorf("发送删除数据命令失败: %v", err)
    }

    // 检查响应状态码
    if len(resp) < 2 {
        return fmt.Errorf("响应数据不完整")
    }
    
    sw := uint16(resp[len(resp)-2])<<8 | uint16(resp[len(resp)-1])
    if sw != 0x9000 {
        return fmt.Errorf("删除数据失败，状态码: 0x%04X", sw)
    }
    
    return nil
}
```

## 安全机制: ECDSA 签名验证

为了增强安全性，本应用支持以下安全机制：

### ECDSA 签名验证

从版本 2.3 开始，读取和删除数据操作必须提供有效的 ECDSA 签名才能执行。这可以防止未授权的访问和修改操作。

#### 签名机制

- **签名算法**: ECDSA 使用 NIST P-256 (secp256r1) 曲线
- **哈希算法**: SHA-256
- **签名格式**: DER 编码格式，变长（通常为 70-72 字节）
- **签名数据**: 用户名(32字节) + 地址(20字节)

#### 更新后的 APDU 格式

| 指令 | APDU 格式 |
|------|-----------|
| READ_DATA | `[CLA][INS=0x20][P1][P2][Lc][userName(32)][addr(20)][signature(DER格式,变长)]` |
| DELETE_DATA | `[CLA][INS=0x30][P1][P2][Lc][userName(32)][addr(20)][signature(DER格式,变长)]` |

#### 状态码

| 状态码 | 描述 |
|-------|------|
| 0x6982 | 签名无效 |

#### 使用 generate_keys.py 生成密钥

为了使用签名验证功能，请按以下步骤操作：

1. 运行 `generate_keys.py` 脚本生成 ECDSA 密钥对
2. 将生成的公钥复制到 `SecurityChipApplet.java` 中的 `EC_PUBLIC_KEY_BYTES` 常量
3. 将生成的私钥 (`ec_private_key.pem`) 用于客户端签名操作

#### 安全注意事项

- 私钥必须妥善保管，不应与智能卡一起存储
- 公钥内置在 JavaCard Applet 中，用于验证签名
- 确保在执行任何读取或删除操作前创建有效签名

## 版本更新说明

### 版本 2.3 更新

在2.3版本中，我们对ECDSA签名验证机制进行了以下优化：

1. **DER格式签名**：现在使用标准的DER编码格式签名代替原始的R和S值。这样做的好处：
   - 符合行业标准实践
   - 更好的兼容性，无需在客户端进行额外处理
   - 更灵活的签名长度处理

2. **使用方法变化**：
   - 读取和删除操作现在使用DER格式的签名
   - 测试脚本已更新为从ec_private_key.pem文件读取私钥
   - 签名大小不再是固定的64字节，而是变长(通常为70-72字节)

### 测试和使用指南

1. **生成密钥对**：
   ```
   python generate_keys.py
   ```
   这将生成`ec_private_key.pem`和`ec_public_key.bin`文件。记得将公钥字节数组复制到JavaCard Applet中。

2. **编译和安装Applet**：
   ```
   ant compile
   ant install
   ```

3. **运行测试脚本**：
   ```
   python test.py
   ```

### 签名说明

在客户端应用中，可以使用以下方式生成DER格式签名：

```python
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.serialization import load_pem_private_key

# 加载私钥
with open('ec_private_key.pem', 'rb') as key_file:
    private_key = load_pem_private_key(key_file.read(), password=None)

# 对数据进行签名 - 直接使用DER格式
data_to_sign = b'your data here'
signature = private_key.sign(
    data_to_sign,
    ec.ECDSA(hashes.SHA256())
)

# 现在signature变量包含DER格式的签名
```

其他语言请参考相应的密码学库文档。

## 测试情况

### 测试环境

- **操作系统**: macOS
- **JavaCard 平台**: JCOP 3.0.4
- **Go 版本**: 1.20+

### 测试步骤

1. **安装 Applet**：确保 CAP 文件已成功安装到智能卡。
2. **生成密钥**：运行 `generate_keys.py` 生成密钥对。
3. **运行客户端**：通过 `main.go` 测试存储、读取和删除数据功能。
4. **验证签名**：确保读取和删除操作提供有效的 ECDSA 签名。

### 测试结果

- **存储数据**: 成功存储 100 条记录，超出限制时返回 `0x6A84`。
- **读取数据**: 成功读取已存储的记录，未找到记录时返回 `0x6A83`。
- **删除数据**: 成功删除指定记录，未找到记录时返回 `0x6A83`。
- **签名验证**: 提供无效签名时返回 `0x6982`。

## 注意事项

1. **错误处理**: 始终检查响应状态码，确保操作成功完成。
2. **数据填充**: 确保所有字段严格按照指定的字节长度发送。
3. **密钥管理**: 私钥必须妥善保管，不应与智能卡一起存储。
4. **签名格式**: 确保签名为 DER 编码格式，长度通常为 70-72 字节。

## 联系方式

如有任何问题或建议，请联系安全芯片团队。
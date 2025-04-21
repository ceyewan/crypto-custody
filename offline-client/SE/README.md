# 安全芯片存储Applet

## 项目简介

这是一个使用JavaCard技术开发的安全芯片存储Applet，可以安装在支持JavaCard的安全芯片中。该Applet提供安全的数据存储与检索功能，适用于需要高安全性的密钥和敏感数据的离线存储场景。

## 功能特性

- **安全数据存储**：将数据安全地存储在芯片内存中
- **签名验证**：使用ECDSA进行身份验证，确保只有授权用户才能读取数据
- **多记录管理**：支持存储多条记录，每条记录包含用户名、地址和消息数据
- **分段传输**：支持大数据分段传输，适用于大容量数据的传输
- **状态机管理**：采用状态机设计模式，确保操作序列的正确性
- **内存安全**：使用JavaCard提供的安全内存管理机制，防止内存溢出和信息泄露

## 技术架构

- **开发平台**：JavaCard 3.0.5+
- **加密算法**：ECDSA with SHA-1
- **密钥类型**：ECC公钥 (192位)
- **内存管理**：混合使用永久存储(EEPROM)和临时存储(RAM)
- **通信协议**：标准APDU命令集

## 代码运行逻辑详解

### 核心数据结构

1. **记录存储结构**：
   - `userNames`：存储所有用户名，总大小为`MAX_RECORDS * MAX_USERNAME_LENGTH`
   - `userNameLengths`：记录每个用户名的实际长度
   - `addresses`：存储所有地址，总大小为`MAX_RECORDS * MAX_ADDR_LENGTH`
   - `addressLengths`：记录每个地址的实际长度
   - `messages`：存储所有消息数据，总大小为`MAX_RECORDS * MAX_MESSAGE_LENGTH`
   - `messageLengths`：记录每条消息的实际长度

2. **内存管理**：
   - 永久存储(EEPROM)：用于存储用户记录和关键状态信息
   - 临时存储(RAM)：用于处理过程中的临时缓冲区，减少对永久内存的写入次数
   - 最大支持存储20条记录
   - 单个消息最大支持6KB数据
   - 单次APDU通信最大240字节

3. **状态机**：
   - `OP_NONE`：空闲状态，可以接受新操作请求
   - `OP_STORE`：存储数据状态，正在处理数据存储操作
   - `OP_READ`：读取数据状态，正在处理数据读取操作

### 执行流程

1. **存储数据流程**：
   - 初始化阶段(`processStoreDataInit`)：验证当前状态为空闲，检查记录空间，存储用户名和地址，转换状态为`OP_STORE`
   - 数据传输阶段(`processStoreDataContinue`)：分段接收消息数据，存入永久存储
   - 完成阶段(`processStoreDataFinalize`)：更新记录计数，重置状态为空闲

2. **读取数据流程**：
   - 初始化阶段(`processReadDataInit`)：验证当前状态为空闲，验证ECDSA签名，查找匹配记录，转换状态为`OP_READ`，返回消息长度
   - 数据传输阶段(`processReadDataContinue`)：分段返回消息数据，如果数据发送完毕自动重置状态
   - 可选完成阶段(`processReadDataFinalize`)：显式重置状态为空闲

3. **安全验证流程**：
   - 将用户名和地址拼接作为签名数据
   - 使用ECDSA with SHA-1算法验证签名
   - 只有签名验证通过才允许读取数据

## APDU命令接口

### 命令格式概述

所有APDU命令的CLA固定为`0x80`，不同操作使用不同的INS值。

| 命令类型 | INS | 说明 |
|---------|-----|------|
| 存储初始化 | 0x10 | 启动存储过程，提供用户名和地址 |
| 存储继续 | 0x11 | 传输消息数据块 |
| 存储完成 | 0x12 | 结束存储过程 |
| 读取初始化 | 0x20 | 启动读取过程，提供用户名、地址和签名 |
| 读取继续 | 0x21 | 获取消息数据块 |
| 读取完成 | 0x22 | 结束读取过程 |

### 详细命令规范

#### 1. 存储数据

##### 存储初始化 (INS: 0x10)
```
CLA: 0x80
INS: 0x10
P1: 0x00
P2: 0x00
Lc: 数据长度
Data: [用户名长度(1)][用户名(变长)][地址长度(1)][地址(变长)]
```

##### 存储继续 (INS: 0x11)
```
CLA: 0x80
INS: 0x11
P1: 0x00
P2: 0x00
Lc: 数据块长度
Data: [消息数据块]
```

##### 存储完成 (INS: 0x12)
```
CLA: 0x80
INS: 0x12
P1: 0x00
P2: 0x00
Lc: 0
```

#### 2. 读取数据

##### 读取初始化 (INS: 0x20)
```
CLA: 0x80
INS: 0x20
P1: 0x00
P2: 0x00
Lc: 数据长度
Data: [用户名长度(1)][用户名(变长)][地址长度(1)][地址(变长)][签名长度(1)][签名(变长)]
Le: 0x02
```

响应: 两字节的消息长度（高字节在前）

##### 读取继续 (INS: 0x21)
```
CLA: 0x80
INS: 0x21
P1: 0x00
P2: 0x00
Le: 要读取的最大长度（通常为0xF0，即240字节）
```

响应: 消息数据块

##### 读取完成 (INS: 0x22)
```
CLA: 0x80
INS: 0x22
P1: 0x00
P2: 0x00
Lc: 0
```

### 响应状态码

| 状态码 | 含义 | 可能原因 |
|-------|------|---------|
| 0x9000 | 操作正常完成 | 命令执行成功 |
| 0x6300 | 验证失败 | 签名验证失败 |
| 0x6A83 | 记录不存在 | 查询的用户名和地址组合不存在 |
| 0x6A86 | 参数不正确 | P1P2参数错误 |
| 0x6700 | 长度错误 | 数据长度超出限制 |
| 0x6100 | 更多数据可用 | 读取操作中还有更多数据可获取 |
| 0x6982 | 安全状态不满足 | 当前状态下不允许操作 |
| 0x6A84 | 文件已满 | 记录数量已达上限 |

## 客户端开发指南

### Java客户端示例代码

#### 存储数据示例

```java
// 初始化存储过程
byte[] userName = "Alice".getBytes();
byte[] addr = "0x1234567890abcdef".getBytes();

// 构建初始化命令
int initDataLength = 2 + userName.length + addr.length; // 2个长度字节 + 用户名 + 地址
byte[] initData = new byte[initDataLength];
int offset = 0;

initData[offset++] = (byte)userName.length;
System.arraycopy(userName, 0, initData, offset, userName.length);
offset += userName.length;

initData[offset++] = (byte)addr.length;
System.arraycopy(addr, 0, initData, offset, addr.length);

CommandAPDU initCommand = new CommandAPDU(0x80, 0x10, 0x00, 0x00, initData);
ResponseAPDU response = channel.transmit(initCommand);
if (response.getSW() != 0x9000) {
    throw new Exception("存储初始化失败: " + Integer.toHexString(response.getSW()));
}

// 准备要存储的消息数据
byte[] message = "这是一段需要安全存储的消息数据".getBytes();

// 分段发送数据
int maxChunkSize = 240; // 单次最大传输240字节
for (int i = 0; i < message.length; i += maxChunkSize) {
    int chunkSize = Math.min(maxChunkSize, message.length - i);
    byte[] chunk = new byte[chunkSize];
    System.arraycopy(message, i, chunk, 0, chunkSize);
    
    CommandAPDU continueCommand = new CommandAPDU(0x80, 0x11, 0x00, 0x00, chunk);
    response = channel.transmit(continueCommand);
    if (response.getSW() != 0x9000) {
        throw new Exception("数据传输失败: " + Integer.toHexString(response.getSW()));
    }
}

// 完成存储操作
CommandAPDU finalizeCommand = new CommandAPDU(0x80, 0x12, 0x00, 0x00);
response = channel.transmit(finalizeCommand);
if (response.getSW() != 0x9000) {
    throw new Exception("存储完成失败: " + Integer.toHexString(response.getSW()));
}
```

#### 读取数据示例

```java
// 准备读取所需参数
byte[] userName = "Alice".getBytes();
byte[] addr = "0x1234567890abcdef".getBytes();

// 生成签名
byte[] dataToSign = new byte[1 + userName.length + 1 + addr.length];
int offset = 0;
dataToSign[offset++] = (byte)userName.length;
System.arraycopy(userName, 0, dataToSign, offset, userName.length);
offset += userName.length;
dataToSign[offset++] = (byte)addr.length;
System.arraycopy(addr, 0, dataToSign, offset, addr.length);

Signature signature = Signature.getInstance("SHA1withECDSA");
signature.initSign(privateKey); // 使用您的私钥
signature.update(dataToSign);
byte[] signatureValue = signature.sign();

// 构建初始化命令
int initDataLength = 3 + userName.length + addr.length + signatureValue.length;
byte[] initData = new byte[initDataLength];
offset = 0;

initData[offset++] = (byte)userName.length;
System.arraycopy(userName, 0, initData, offset, userName.length);
offset += userName.length;

initData[offset++] = (byte)addr.length;
System.arraycopy(addr, 0, initData, offset, addr.length);
offset += addr.length;

initData[offset++] = (byte)signatureValue.length;
System.arraycopy(signatureValue, 0, initData, offset, signatureValue.length);

CommandAPDU initCommand = new CommandAPDU(0x80, 0x20, 0x00, 0x00, initData, 2);
ResponseAPDU response = channel.transmit(initCommand);
if (response.getSW() != 0x9000) {
    throw new Exception("读取初始化失败: " + Integer.toHexString(response.getSW()));
}

// 解析消息长度
byte[] responseData = response.getData();
int messageLength = ((responseData[0] & 0xFF) << 8) | (responseData[1] & 0xFF);

// 分段读取消息数据
ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
while (outputStream.size() < messageLength) {
    CommandAPDU continueCommand = new CommandAPDU(0x80, 0x21, 0x00, 0x00, 240);
    response = channel.transmit(continueCommand);
    
    if (response.getSW() != 0x9000 && response.getSW() != 0x6100) {
        throw new Exception("数据读取失败: " + Integer.toHexString(response.getSW()));
    }
    
    byte[] chunk = response.getData();
    outputStream.write(chunk);
    
    // 如果没有更多数据则退出循环
    if (response.getSW() == 0x9000) {
        break;
    }
}

// 可选：显式完成读取操作
CommandAPDU finalizeCommand = new CommandAPDU(0x80, 0x22, 0x00, 0x00);
response = channel.transmit(finalizeCommand);
// 读取过程在数据全部传输后会自动完成，此步骤可选

// 获取完整的消息数据
byte[] completeMessage = outputStream.toByteArray();
```

## 内存与性能考量

### 内存限制

该Applet设计考虑到JavaCard平台的内存限制：
- 最大支持20条记录
- 每个用户名最大32字节
- 每个地址最大64字节
- 每条消息最大6KB
- 总共最大内存占用约：
  - 用户名：20 * 32 = 640字节
  - 地址：20 * 64 = 1280字节
  - 消息：20 * 6KB = 120KB
  - 元数据：约500字节

对于资源受限的JavaCard，可能需要调整记录数量或每条记录的最大尺寸。

### 性能优化

1. **减少EEPROM写入**
   - 使用瞬态内存(RAM)存储临时数据
   - 在操作完成时才一次性写入EEPROM

2. **分段传输**
   - 单次最大传输240字节，避免超出APDU缓冲区限制
   - 支持大数据分段存储和读取

3. **快速查找**
   - 使用线性查找匹配记录
   - 对于大量记录，可以考虑实现更高效的索引结构

## 安全设计

### 身份验证

1. **ECDSA签名验证**
   - 使用192位ECC密钥提供强大的安全性
   - 签名数据为用户名和地址的组合，确保针对特定记录的访问控制

2. **状态机设计**
   - 确保操作按正确顺序执行
   - 防止中间状态的非法访问

### 防御措施

1. **输入验证**
   - 对所有输入进行长度和边界检查
   - 防止缓冲区溢出攻击

2. **内存隔离**
   - 使用JavaCard提供的安全内存管理
   - 瞬态缓冲区在电源重置后自动清除

3. **错误处理**
   - 提供详细的错误状态码
   - 在异常情况下不泄露敏感信息

## 扩展开发建议

1. **删除记录功能**
   - 添加删除指令以回收存储空间
   - 实现安全擦除，确保敏感数据不可恢复

2. **密钥管理**
   - 支持多个验证密钥
   - 实现密钥轮换机制

3. **数据加密**
   - 将存储的消息数据进行加密
   - 使用不同的加密密钥分别保护不同记录

4. **事务支持**
   - 添加事务机制，防止写入操作中断导致数据不一致
   - 利用JavaCard的事务API

## 测试建议

1. **功能测试**
   - 验证所有指令的正常功能
   - 测试边界条件(最大记录数、最大数据大小等)

2. **安全测试**
   - 尝试使用错误签名进行操作
   - 测试在各种状态下的非法操作

3. **性能测试**
   - 测量大数据传输性能
   - 评估内存使用情况

4. **兼容性测试**
   - 在不同JavaCard平台上测试
   - 测试与不同读卡器和中间件的兼容性

## 许可证

版权所有 © 2025 Security Chip Team

---

## 更新日志

- **2025-04-21**: 代码注释全面更新，改进文档
- **2025-03-15**: 初始版本
# 安全芯片存储 Applet 使用指南

## 概述

该 Applet 实现了一个简单的数据存储和检索系统，运行在 JavaCard 智能卡平台上。主要功能包括:
- 存储固定长度的用户名(32字节)、地址(64字节)和消息数据(32字节)
- 通过用户名和地址检索存储的消息数据
- 支持覆盖已存在的(userName, Addr)对的数据

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

本 Applet 支持两种基本指令:

| 指令名称 | INS值 | 描述 |
|---------|------|------|
| STORE_DATA | 0x10 | 存储新记录或更新现有记录 |
| READ_DATA | 0x20 | 读取现有记录 |

## 1. 存储数据指令 (STORE_DATA)

### 请求格式

```
CLA: 0x80 (默认JavaCard应用类)
INS: 0x10 (存储数据指令)
P1: 0x00 (保留)
P2: 0x00 (保留)
Lc: 0x80 (总数据长度: 32+64+32=128字节)
数据: [userName(32字节)][addr(64字节)][message(32字节)]
```

### 数据字段说明

- `userName`: 固定32字节，用户名标识
- `addr`: 固定64字节，地址数据
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
   [32字节userName] [64字节addr] [32字节message]

<< [记录索引] [记录总数] 90 00
```

## 2. 读取数据指令 (READ_DATA)

### 请求格式

```
CLA: 0x80 (默认JavaCard应用类)
INS: 0x20 (读取数据指令)
P1: 0x00 (保留)
P2: 0x00 (保留)
Lc: 变长 (至少96字节: 32+64=96)
数据: [userName(32字节)][addr(64字节)][sign(可变长度)]
```

### 数据字段说明

- `userName`: 固定32字节，要查找的用户名
- `addr`: 固定64字节，要查找的地址
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
   [32字节userName] [64字节addr]

<< [32字节message] 90 00
```

## 性能与容量限制

- 最大记录数: 100条
- 用户名长度: 固定32字节
- 地址长度: 固定64字节 
- 消息长度: 固定32字节

## 最佳实践

1. **错误处理**: 始终检查响应状态码，确保操作成功完成
   
2. **重复键处理**: 当存储具有相同(userName, addr)的记录时，新数据将覆盖旧数据

3. **数据填充**: 如果实际数据未达到固定长度，需要进行填充:
   - 字符串数据建议使用空字节(0x00)右填充
   - 二进制数据可考虑使用0xFF或0x00填充

4. **数据格式化**: 确保所有字段严格按照指定的字节长度发送

## 开发示例

### Java客户端代码示例

```java
public class SecurityChipClient {
    private CardChannel channel; // 假设已获取到卡通道
    
    // 存储数据
    public void storeData(byte[] userName, byte[] address, byte[] message) throws Exception {
        if (userName.length != 32 || address.length != 64 || message.length != 32) {
            throw new IllegalArgumentException("数据长度不符合要求");
        }
        
        CommandAPDU command = new CommandAPDU(
            0x80,           // CLA
            0x10,           // INS (STORE_DATA)
            0x00,           // P1
            0x00,           // P2
            concatenate(userName, address, message) // 数据
        );
        
        ResponseAPDU response = channel.transmit(command);
        
        if (response.getSW() != 0x9000) {
            throw new Exception("存储失败，错误码: " + Integer.toHexString(response.getSW()));
        }
        
        byte[] responseData = response.getData();
        System.out.println("存储成功！记录索引: " + responseData[0] + 
                          ", 记录总数: " + responseData[1]);
    }
    
    // 读取数据
    public byte[] readData(byte[] userName, byte[] address) throws Exception {
        if (userName.length != 32 || address.length != 64) {
            throw new IllegalArgumentException("数据长度不符合要求");
        }
        
        CommandAPDU command = new CommandAPDU(
            0x80,           // CLA
            0x20,           // INS (READ_DATA)
            0x00,           // P1
            0x00,           // P2
            concatenate(userName, address) // 数据
        );
        
        ResponseAPDU response = channel.transmit(command);
        
        if (response.getSW() != 0x9000) {
            throw new Exception("读取失败，错误码: " + Integer.toHexString(response.getSW()));
        }
        
        return response.getData(); // 返回32字节消息
    }
    
    // 辅助方法：连接多个字节数组
    private byte[] concatenate(byte[]... arrays) {
        int totalLength = 0;
        for (byte[] array : arrays) {
            totalLength += array.length;
        }
        
        byte[] result = new byte[totalLength];
        int currentIndex = 0;
        
        for (byte[] array : arrays) {
            System.arraycopy(array, 0, result, currentIndex, array.length);
            currentIndex += array.length;
        }
        
        return result;
    }
}
```

---
如有任何问题或建议，请联系安全芯片团队。
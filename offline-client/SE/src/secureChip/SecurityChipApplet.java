/**
 * 安全芯片存储Applet - 用于在JavaCard智能卡上安全存储和检索数据
 * 
 * 该Applet实现了一个简单但安全的数据存储和检索系统，使用ECDSA进行身份验证。
 * 主要功能包括:
 * 1. 存储用户名、地址和消息数据
 * 2. 通过ECDSA签名验证身份后检索数据
 * 3. 支持分段传输大型数据
 * 
 * 工作原理:
 * - 存储操作分为初始化、继续和完成三个阶段
 * - 读取操作同样分为初始化、继续和完成三个阶段
 * - 所有操作都包含了状态检查和边界验证，确保数据安全性
 * 
 * @author Security Chip Team
 * @version 1.0
 */
package securitychip;

import javacard.framework.*;
import javacard.security.*;
import javacardx.crypto.*;

public class SecurityChipApplet extends Applet {
    // APDU指令常量 - 定义了与客户端通信的命令码
    private static final byte INS_STORE_DATA_INIT = (byte) 0x10;      // 存储数据初始化命令
    private static final byte INS_STORE_DATA_CONTINUE = (byte) 0x11;  // 存储数据继续命令
    private static final byte INS_STORE_DATA_FINALIZE = (byte) 0x12;  // 存储数据完成命令
    private static final byte INS_READ_DATA_INIT = (byte) 0x20;       // 读取数据初始化命令
    private static final byte INS_READ_DATA_CONTINUE = (byte) 0x21;   // 读取数据继续命令
    private static final byte INS_READ_DATA_FINALIZE = (byte) 0x22;   // 读取数据完成命令
    
    // 状态常量 - APDU响应状态码
    private static final short SW_VERIFICATION_FAILED = (short) 0x6300;  // 验证失败
    private static final short SW_RECORD_NOT_FOUND = (short) 0x6A83;     // 记录未找到
    private static final short SW_INCORRECT_P1P2 = (short) 0x6A86;       // P1P2参数不正确
    private static final short SW_WRONG_LENGTH = (short) 0x6700;         // 长度错误
    private static final short SW_MORE_DATA_AVAILABLE = (short) 0x6100;  // 有更多数据可用
    private static final short SW_OPERATION_COMPLETE = (short) 0x9000;   // 操作正常完成
    
    // 存储限制常量
    private static final byte MAX_RECORDS = 20;                // 最大记录数量
    private static final byte MAX_USERNAME_LENGTH = 32;        // 用户名最大长度
    private static final byte MAX_ADDR_LENGTH = 64;            // 地址最大长度
    private static final short MAX_MESSAGE_LENGTH = 6144;      // 消息最大长度(6KB)
    private static final short BUFFER_SIZE = 240;              // APDU通信缓冲区大小
    
    // 数据存储结构 - 用于在永久内存中保存用户记录
    private byte[] userNames;         // 所有用户名存储数组
    private short[] userNameLengths;  // 每个用户名的实际长度
    private byte[] addresses;         // 所有地址存储数组
    private short[] addressLengths;   // 每个地址的实际长度  
    private byte[] messages;          // 所有消息存储数组
    private short[] messageLengths;   // 每条消息的实际长度
    private byte recordCount;         // 当前记录数量
    
    // 临时缓冲区 - 用于数据传输过程中的临时存储
    private byte[] buffer;             // 传输缓冲区
    private short bufferOffset;        // 缓冲区当前偏移量
    private short currentMessageOffset; // 当前处理消息的偏移量
    private byte currentOperation;     // 当前操作类型
    private byte currentRecordIndex;   // 当前操作的记录索引
    
    // 操作状态常量
    private static final byte OP_NONE = 0;   // 空闲状态
    private static final byte OP_STORE = 1;  // 存储操作状态
    private static final byte OP_READ = 2;   // 读取操作状态
    
    // 签名验证相关组件
    private ECPublicKey verificationKey;  // ECDSA公钥
    private Signature ecSignature;        // 签名验证引擎
    
    /**
     * 私有构造方法 - 初始化Applet
     * 
     * 此方法初始化所有数据结构和密码组件，包括:
     * - 分配存储数组
     * - 初始化状态变量
     * - 设置ECDSA验证组件
     */
    private SecurityChipApplet() {
        // 初始化存储结构 - 在卡片永久内存中分配空间
        userNames = new byte[MAX_RECORDS * MAX_USERNAME_LENGTH];
        userNameLengths = new short[MAX_RECORDS];
        addresses = new byte[MAX_RECORDS * MAX_ADDR_LENGTH];
        addressLengths = new short[MAX_RECORDS];
        messages = new byte[MAX_RECORDS * MAX_MESSAGE_LENGTH]; // 注意：这对于某些智能卡可能太大
        messageLengths = new short[MAX_RECORDS];
        recordCount = 0;
        
        // 使用瞬态内存(RAM)初始化数据传输缓冲区
        // 使用瞬态内存创建缓冲区，这样在断电后会自动清除，同时可以减少EEPROM写入
        buffer = JCSystem.makeTransientByteArray(BUFFER_SIZE, JCSystem.CLEAR_ON_RESET);
        bufferOffset = 0;
        currentMessageOffset = 0;
        currentOperation = OP_NONE;
        
        // 初始化EC签名验证组件
        try {
            // 创建并初始化EC公钥
            verificationKey = (ECPublicKey) KeyBuilder.buildKey(KeyBuilder.TYPE_EC_FP_PUBLIC, KeyBuilder.LENGTH_EC_FP_192, false);
            
            // 这里应设置曲线参数和公钥值
            // 这是一个占位符 - 实际值应在个性化阶段设置
            
            // 初始化签名引擎
            ecSignature = Signature.getInstance(Signature.ALG_ECDSA_SHA, false);
            ecSignature.init(verificationKey, Signature.MODE_VERIFY);
        } catch (CryptoException e) {
            // 处理初始化错误 - 在实际部署中应添加适当的错误处理
        }
        
        // 注册Applet以便可以被选择
        register();
    }
    
    /**
     * 安装方法 - JavaCard框架调用此静态方法安装Applet
     * 
     * @param bArray 安装参数数组
     * @param bOffset 参数数组中的偏移量
     * @param bLength 参数数组的长度
     */
    public static void install(byte[] bArray, short bOffset, byte bLength) {
        new SecurityChipApplet();
    }
    
    /**
     * 处理APDU命令 - JavaCard框架的入口点，处理所有传入的APDU命令
     * 
     * @param apdu APDU对象，包含命令和数据
     */
    public void process(APDU apdu) {
        if (selectingApplet()) {
            return;
        }
        
        byte[] buffer = apdu.getBuffer();
        byte ins = buffer[ISO7816.OFFSET_INS];
        
        // 根据指令字节分发到不同的处理方法
        switch (ins) {
            case INS_STORE_DATA_INIT:
                processStoreDataInit(apdu);
                break;
            case INS_STORE_DATA_CONTINUE:
                processStoreDataContinue(apdu);
                break;
            case INS_STORE_DATA_FINALIZE:
                processStoreDataFinalize(apdu);
                break;
            case INS_READ_DATA_INIT:
                processReadDataInit(apdu);
                break;
            case INS_READ_DATA_CONTINUE:
                processReadDataContinue(apdu);
                break;
            case INS_READ_DATA_FINALIZE:
                processReadDataFinalize(apdu);
                break;
            default:
                ISOException.throwIt(ISO7816.SW_INS_NOT_SUPPORTED);
        }
    }
    
    /**
     * 处理存储数据初始化 - 开始新记录的存储过程
     * 
     * 此方法处理新记录的存储初始化:
     * 1. 验证Applet当前是否处于空闲状态
     * 2. 验证是否有空间存储新记录
     * 3. 解析并存储用户名和地址
     * 4. 设置Applet进入存储状态
     * 
     * APDU格式: [CLA][INS][P1=0][P2=0][Lc][userNameLength(1)][userName(var)][addrLength(1)][addr(var)]
     * 
     * @param apdu APDU命令对象
     */
    private void processStoreDataInit(APDU apdu) {
        // 验证当前操作状态必须为空闲
        if (currentOperation != OP_NONE) {
            ISOException.throwIt(ISO7816.SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] buffer = apdu.getBuffer();
        short lc = apdu.setIncomingAndReceive();
        
        // 检查是否有空间存储新记录
        if (recordCount >= MAX_RECORDS) {
            ISOException.throwIt(ISO7816.SW_FILE_FULL);
        }
        
        // 解析APDU数据
        short offset = ISO7816.OFFSET_CDATA;
        
        // 获取并验证用户名长度
        byte userNameLength = buffer[offset++];
        if (userNameLength > MAX_USERNAME_LENGTH || userNameLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 保存用户名到永久存储
        short userNameOffset = (short)(recordCount * MAX_USERNAME_LENGTH);
        Util.arrayCopy(buffer, offset, userNames, userNameOffset, userNameLength);
        userNameLengths[recordCount] = userNameLength;
        offset += userNameLength;
        
        // 获取并验证地址长度
        byte addrLength = buffer[offset++];
        if (addrLength > MAX_ADDR_LENGTH || addrLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 保存地址到永久存储
        short addrOffset = (short)(recordCount * MAX_ADDR_LENGTH);
        Util.arrayCopy(buffer, offset, addresses, addrOffset, addrLength);
        addressLengths[recordCount] = addrLength;
        
        // 准备消息存储 - 设置状态为存储操作
        currentRecordIndex = recordCount;
        currentOperation = OP_STORE;
        currentMessageOffset = 0;
        bufferOffset = 0;
        
        // 将消息长度初始化为0
        messageLengths[currentRecordIndex] = 0;
    }
    
    /**
     * 处理存储数据继续 - 接收并存储消息数据块
     * 
     * 此方法处理分段接收的消息数据:
     * 1. 验证当前是否处于存储状态
     * 2. 接收数据块并保存到消息存储区
     * 3. 更新已处理的消息偏移量
     * 
     * APDU格式: [CLA][INS][P1=0][P2=0][Lc][数据块]
     * 
     * @param apdu APDU命令对象
     */
    private void processStoreDataContinue(APDU apdu) {
        // 验证当前必须处于存储操作状态
        if (currentOperation != OP_STORE) {
            ISOException.throwIt(ISO7816.SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] buffer = apdu.getBuffer();
        short lc = apdu.setIncomingAndReceive();
        
        // 验证缓冲区是否能容纳更多数据
        if ((short)(currentMessageOffset + lc) > MAX_MESSAGE_LENGTH) {
            ISOException.throwIt(ISO7816.SW_FILE_FULL);
        }
        
        // 将接收的消息块保存到永久存储
        short messageOffset = (short)(currentRecordIndex * MAX_MESSAGE_LENGTH + currentMessageOffset);
        Util.arrayCopy(buffer, ISO7816.OFFSET_CDATA, messages, messageOffset, lc);
        currentMessageOffset += lc; // 更新已处理的消息长度
    }
    
    /**
     * 处理存储数据完成 - 结束当前存储操作
     * 
     * 此方法完成整个消息的存储过程:
     * 1. 验证当前是否处于存储状态
     * 2. 设置消息的最终长度
     * 3. 增加记录计数
     * 4. 重置Applet状态为空闲
     * 
     * APDU格式: [CLA][INS][P1=0][P2=0][Lc=0]
     * 
     * @param apdu APDU命令对象
     */
    private void processStoreDataFinalize(APDU apdu) {
        // 验证当前必须处于存储操作状态
        if (currentOperation != OP_STORE) {
            ISOException.throwIt(ISO7816.SW_CONDITIONS_NOT_SATISFIED);
        }
        
        // 设置消息的最终长度
        messageLengths[currentRecordIndex] = currentMessageOffset;
        
        // 仅在成功完成存储后增加记录计数
        recordCount++;
        
        // 重置Applet状态为空闲
        currentOperation = OP_NONE;
        currentMessageOffset = 0;
    }
    
    /**
     * 处理读取数据初始化 - 开始数据检索过程
     * 
     * 此方法处理读取初始化请求:
     * 1. 验证当前是否处于空闲状态
     * 2. 解析用户名、地址和签名
     * 3. 验证签名以确认身份
     * 4. 查找匹配的记录
     * 5. 设置Applet进入读取状态
     * 6. 返回消息总长度
     * 
     * APDU格式: [CLA][INS][P1=0][P2=0][Lc][userNameLength(1)][userName(var)][addrLength(1)][addr(var)][signatureLength(1)][signature(var)]
     * 
     * @param apdu APDU命令对象
     */
    private void processReadDataInit(APDU apdu) {
        // 验证当前操作状态必须为空闲
        if (currentOperation != OP_NONE) {
            ISOException.throwIt(ISO7816.SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] buffer = apdu.getBuffer();
        short lc = apdu.setIncomingAndReceive();
        
        // 解析APDU数据
        short offset = ISO7816.OFFSET_CDATA;
        
        // 获取并验证用户名长度
        byte userNameLength = buffer[offset++];
        if (userNameLength > MAX_USERNAME_LENGTH || userNameLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 创建临时缓冲区保存用户名 (使用RAM而非EEPROM)
        byte[] tempUserName = JCSystem.makeTransientByteArray(userNameLength, JCSystem.CLEAR_ON_RESET);
        Util.arrayCopy(buffer, offset, tempUserName, (short)0, userNameLength);
        offset += userNameLength;
        
        // 获取并验证地址长度
        byte addrLength = buffer[offset++];
        if (addrLength > MAX_ADDR_LENGTH || addrLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 创建临时缓冲区保存地址
        byte[] tempAddr = JCSystem.makeTransientByteArray(addrLength, JCSystem.CLEAR_ON_RESET);
        Util.arrayCopy(buffer, offset, tempAddr, (short)0, addrLength);
        offset += addrLength;
        
        // 获取并验证签名长度
        byte signatureLength = buffer[offset++];
        if (signatureLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 准备签名验证 - 连接用户名和地址作为验证数据
        byte[] dataToVerify = JCSystem.makeTransientByteArray((short)(userNameLength + addrLength), JCSystem.CLEAR_ON_RESET);
        Util.arrayCopy(tempUserName, (short)0, dataToVerify, (short)0, userNameLength);
        Util.arrayCopy(tempAddr, (short)0, dataToVerify, userNameLength, addrLength);
        
        // 执行签名验证
        boolean verified = false;
        try {
            ecSignature.update(dataToVerify, (short)0, (short)(userNameLength + addrLength));
            verified = ecSignature.verify(buffer, offset, signatureLength, this.buffer, (short)0, (short)0);
        } catch (CryptoException e) {
            ISOException.throwIt(SW_VERIFICATION_FAILED);
        }
        
        // 如果签名验证失败，返回验证失败状态
        if (!verified) {
            ISOException.throwIt(SW_VERIFICATION_FAILED);
        }
        
        // 查找匹配的记录
        byte foundIndex = -1;
        for (byte i = 0; i < recordCount; i++) {
            boolean userNameMatch = true;
            boolean addrMatch = true;
            
            // 检查用户名是否匹配
            if (userNameLengths[i] == userNameLength) {
                short userNameOffset = (short)(i * MAX_USERNAME_LENGTH);
                for (short j = 0; j < userNameLength; j++) {
                    if (userNames[(short)(userNameOffset + j)] != tempUserName[j]) {
                        userNameMatch = false;
                        break;
                    }
                }
            } else {
                userNameMatch = false;
            }
            
            // 如果用户名匹配，继续检查地址是否匹配
            if (userNameMatch && addressLengths[i] == addrLength) {
                short addrOffset = (short)(i * MAX_ADDR_LENGTH);
                for (short j = 0; j < addrLength; j++) {
                    if (addresses[(short)(addrOffset + j)] != tempAddr[j]) {
                        addrMatch = false;
                        break;
                    }
                }
            } else {
                addrMatch = false;
            }
            
            // 如果用户名和地址都匹配，找到了对应记录
            if (userNameMatch && addrMatch) {
                foundIndex = i;
                break;
            }
        }
        
        // 如果没有找到匹配记录，返回记录未找到状态
        if (foundIndex == -1) {
            ISOException.throwIt(SW_RECORD_NOT_FOUND);
        }
        
        // 设置状态为读取操作并准备数据传输
        currentRecordIndex = foundIndex;
        currentOperation = OP_READ;
        currentMessageOffset = 0;
        
        // 返回消息总长度作为响应
        short messageLength = messageLengths[currentRecordIndex];
        buffer[0] = (byte)((messageLength >> 8) & 0xFF); // 高字节
        buffer[1] = (byte)(messageLength & 0xFF);        // 低字节
        apdu.setOutgoingAndSend((short)0, (short)2);
    }
    
    /**
     * 处理读取数据继续 - 返回消息数据块
     * 
     * 此方法处理分段返回消息数据:
     * 1. 验证当前是否处于读取状态
     * 2. 检查是否有剩余数据要发送
     * 3. 计算本次返回的数据块大小
     * 4. 从消息存储区复制数据并返回
     * 5. 如果数据全部发送完成，自动重置状态
     * 
     * APDU格式: [CLA][INS][P1=0][P2=0][Le]
     * 
     * @param apdu APDU命令对象
     */
    private void processReadDataContinue(APDU apdu) {
        // 验证当前必须处于读取操作状态
        if (currentOperation != OP_READ) {
            ISOException.throwIt(ISO7816.SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] buffer = apdu.getBuffer();
        
        // 检查是否已读取完整个消息
        short messageLength = messageLengths[currentRecordIndex];
        if (currentMessageOffset >= messageLength) {
            // 所有数据都已读取，重置状态并返回操作完成状态
            currentOperation = OP_NONE;
            ISOException.throwIt(SW_OPERATION_COMPLETE);
            return;
        }
        
        // 计算本次返回的数据块大小
        short chunkSize = (short)Math.min(BUFFER_SIZE, (short)(messageLength - currentMessageOffset));
        
        // 从永久存储复制消息块到缓冲区
        short messageOffset = (short)(currentRecordIndex * MAX_MESSAGE_LENGTH + currentMessageOffset);
        Util.arrayCopy(messages, messageOffset, buffer, (short)0, chunkSize);
        
        // 更新已处理的消息偏移量
        currentMessageOffset += chunkSize;
        
        // 发送数据块
        apdu.setOutgoingAndSend((short)0, chunkSize);
        
        // 如果还有更多数据待发送，返回特殊状态码
        if (currentMessageOffset < messageLength) {
            ISOException.throwIt(SW_MORE_DATA_AVAILABLE);
        } else {
            // 如果所有数据都已发送，自动重置操作状态
            currentOperation = OP_NONE;
            // 默认返回正常完成状态
        }
    }
    
    /**
     * 处理读取数据完成 - 显式结束读取操作
     * 
     * 此方法显式完成读取过程，与存储操作保持对称:
     * 1. 验证当前是否处于读取状态
     * 2. 重置Applet状态为空闲
     * 
     * APDU格式: [CLA][INS][P1=0][P2=0][Lc=0]
     * 
     * @param apdu APDU命令对象
     * 注意: 即使读取操作在最后一块数据发送后会自动完成，此方法仍提供显式完成的选项
     */
    private void processReadDataFinalize(APDU apdu) {
        // 验证当前必须处于读取操作状态
        if (currentOperation != OP_READ) {
            ISOException.throwIt(ISO7816.SW_CONDITIONS_NOT_SATISFIED);
        }
        
        // 重置读取操作状态为空闲
        currentOperation = OP_NONE;
        currentMessageOffset = 0;
        
        // 返回成功状态 - 操作完成
    }
}
/**
 * 安全芯片存储Applet - 用于在JavaCard智能卡上安全存储和检索数据
 * 
 * 该Applet实现了一个简单的数据存储和检索系统。
 * 主要功能包括:
 * 1. 存储用户名、地址和消息数据
 * 2. 通过用户名和地址检索数据
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

public class SecurityChipApplet extends Applet {
    // APDU指令常量 - 定义了与客户端通信的命令码
    private static final byte INS_STORE_DATA_INIT = (byte) 0x10;      // 存储数据初始化命令
    private static final byte INS_STORE_DATA_CONTINUE = (byte) 0x11;  // 存储数据继续命令
    private static final byte INS_STORE_DATA_FINALIZE = (byte) 0x12;  // 存储数据完成命令
    private static final byte INS_READ_DATA_INIT = (byte) 0x20;       // 读取数据初始化命令
    private static final byte INS_READ_DATA_CONTINUE = (byte) 0x21;   // 读取数据继续命令
    private static final byte INS_READ_DATA_FINALIZE = (byte) 0x22;   // 读取数据完成命令
    
    // 状态常量 - APDU响应状态码
    private static final short SW_RECORD_NOT_FOUND = (short) 0x6A83;     // 记录未找到
    private static final short SW_INCORRECT_P1P2 = (short) 0x6A86;       // P1P2参数不正确
    private static final short SW_WRONG_LENGTH = (short) 0x6700;         // 长度错误
    private static final short SW_MORE_DATA_AVAILABLE = (short) 0x6100;  // 有更多数据可用
    private static final short SW_OPERATION_COMPLETE = (short) 0x9000;   // 操作正常完成
    private static final short SW_CONDITIONS_NOT_SATISFIED = ISO7816.SW_CONDITIONS_NOT_SATISFIED; // 操作条件不满足
    private static final short SW_FILE_FULL = ISO7816.SW_FILE_FULL; // 存储空间已满
    
    // 存储限制常量
    private static final byte MAX_RECORDS = 20;                // 最大记录数量
    private static final byte MAX_USERNAME_LENGTH = 32;        // 用户名最大长度
    private static final byte MAX_ADDR_LENGTH = 64;            // 地址最大长度
    private static final short MAX_MESSAGE_LENGTH = 1024;      // 消息最大长度(调整为1KB)
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
    private byte[] transferBuffer;     // 传输缓冲区 (重命名以更清晰)
    private short bufferOffset;        // 缓冲区当前偏移量 (可能不再需要，检查用途)
    private short currentMessageOffset; // 当前处理消息的偏移量
    private byte currentOperation;     // 当前操作类型
    private byte currentRecordIndex;   // 当前操作的记录索引
    
    // 操作状态常量
    private static final byte OP_NONE = 0;   // 空闲状态
    private static final byte OP_STORE = 1;  // 存储操作状态
    private static final byte OP_READ = 2;   // 读取操作状态
    
    /**
     * 私有构造方法 - 初始化Applet
     * 
     * 此方法初始化所有数据结构，包括:
     * - 分配存储数组
     * - 初始化状态变量
     */
    private SecurityChipApplet() {
        // 初始化存储结构 - 在卡片永久内存中分配空间
        userNames = new byte[MAX_RECORDS * MAX_USERNAME_LENGTH];
        userNameLengths = new short[MAX_RECORDS];
        addresses = new byte[MAX_RECORDS * MAX_ADDR_LENGTH];
        addressLengths = new short[MAX_RECORDS];
        
        // 修复大数组分配问题 - 拆分计算以避免整数溢出
        short messageArraySize = (short)(MAX_RECORDS * MAX_MESSAGE_LENGTH);
        messages = new byte[messageArraySize]; // 调整后的大小应该符合JavaCard限制
        
        messageLengths = new short[MAX_RECORDS];
        recordCount = 0;
        
        // 使用瞬态内存(RAM)初始化数据传输缓冲区
        transferBuffer = JCSystem.makeTransientByteArray(BUFFER_SIZE, JCSystem.CLEAR_ON_RESET);
        bufferOffset = 0; // 检查是否仍需要此变量
        currentMessageOffset = 0;
        currentOperation = OP_NONE;
        
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
        
        byte[] apduBuffer = apdu.getBuffer(); // 使用更明确的变量名
        byte ins = apduBuffer[ISO7816.OFFSET_INS];
        
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
     * @param apdu APDU命令对象
     */
    private void processStoreDataInit(APDU apdu) {
        // 验证当前操作状态必须为空闲
        if (currentOperation != OP_NONE) {
            ISOException.throwIt(SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] apduBuffer = apdu.getBuffer();
        short lc = apdu.setIncomingAndReceive();
        
        // 检查是否有空间存储新记录
        if (recordCount >= MAX_RECORDS) {
            ISOException.throwIt(SW_FILE_FULL);
        }
        
        // 解析APDU数据
        short offset = ISO7816.OFFSET_CDATA;
        
        // 获取并验证用户名长度
        byte userNameLength = apduBuffer[offset++];
        if (userNameLength > MAX_USERNAME_LENGTH || userNameLength <= 0 || offset > lc) { // 增加边界检查
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 检查用户名数据是否完整
        if ((short)(offset + userNameLength) > (short)(ISO7816.OFFSET_CDATA + lc)) {
             ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 保存用户名到永久存储
        short userNameOffset = (short)(recordCount * MAX_USERNAME_LENGTH);
        Util.arrayCopy(apduBuffer, offset, userNames, userNameOffset, userNameLength);
        userNameLengths[recordCount] = userNameLength;
        offset += userNameLength;
        
        // 获取并验证地址长度
        if (offset >= (short)(ISO7816.OFFSET_CDATA + lc)) { // 检查是否还有数据读取地址长度
             ISOException.throwIt(SW_WRONG_LENGTH);
        }
        byte addrLength = apduBuffer[offset++];
        if (addrLength > MAX_ADDR_LENGTH || addrLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 检查地址数据是否完整
        if ((short)(offset + addrLength) > (short)(ISO7816.OFFSET_CDATA + lc)) {
             ISOException.throwIt(SW_WRONG_LENGTH);
        }
        
        // 保存地址到永久存储
        short addrOffset = (short)(recordCount * MAX_ADDR_LENGTH);
        Util.arrayCopy(apduBuffer, offset, addresses, addrOffset, addrLength);
        addressLengths[recordCount] = addrLength;
        
        // 准备消息存储 - 设置状态为存储操作
        currentRecordIndex = recordCount;
        currentOperation = OP_STORE;
        currentMessageOffset = 0;
        // bufferOffset = 0; // 确认是否需要重置
        
        // 将消息长度初始化为0
        messageLengths[currentRecordIndex] = 0;
    }
    
    /**
     * 处理存储数据继续 - 接收并存储消息数据块
     * @param apdu APDU命令对象
     */
    private void processStoreDataContinue(APDU apdu) {
        // 验证当前必须处于存储操作状态
        if (currentOperation != OP_STORE) {
            ISOException.throwIt(SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] apduBuffer = apdu.getBuffer();
        short lc = apdu.setIncomingAndReceive();
        
        // 验证缓冲区是否能容纳更多数据
        if ((short)(currentMessageOffset + lc) > MAX_MESSAGE_LENGTH) {
            ISOException.throwIt(SW_FILE_FULL); // 使用更具体的错误码
        }
        
        // 将接收的消息块保存到永久存储
        short messageOffset = (short)(currentRecordIndex * MAX_MESSAGE_LENGTH + currentMessageOffset);
        Util.arrayCopy(apduBuffer, ISO7816.OFFSET_CDATA, messages, messageOffset, lc);
        currentMessageOffset += lc; // 更新已处理的消息长度
    }
    
    /**
     * 处理存储数据完成 - 结束当前存储操作
     * @param apdu APDU命令对象
     */
    private void processStoreDataFinalize(APDU apdu) {
        // 验证当前必须处于存储操作状态
        if (currentOperation != OP_STORE) {
            ISOException.throwIt(SW_CONDITIONS_NOT_SATISFIED);
        }
        
        // 设置消息的最终长度
        messageLengths[currentRecordIndex] = currentMessageOffset;
        
        // 仅在成功完成存储后增加记录计数
        recordCount++;
        
        // 重置Applet状态为空闲
        resetOperationState();
    }
    
    /**
     * 处理读取数据初始化 - 开始数据检索过程
     * 
     * 此方法处理读取初始化请求:
     * 1. 验证当前是否处于空闲状态
     * 2. 解析用户名、地址和签名占位符长度 (占位符数据被忽略)
     * 3. 查找匹配的记录
     * 4. 设置Applet进入读取状态
     * 5. 返回消息总长度
     * 
     * APDU格式: [CLA][INS][P1=0][P2=0][Lc][userNameLength(1)][userName(var)][addrLength(1)][addr(var)][signatureLength(1)][signaturePlaceholder(var)]
     * 
     * @param apdu APDU命令对象
     */
    private void processReadDataInit(APDU apdu) {
        // 验证当前操作状态必须为空闲
        if (currentOperation != OP_NONE) {
            ISOException.throwIt(SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] apduBuffer = apdu.getBuffer();
        short lc = apdu.setIncomingAndReceive();
        
        // 解析APDU数据
        short offset = ISO7816.OFFSET_CDATA;
        
        // 获取并验证用户名长度
        if (offset >= (short)(ISO7816.OFFSET_CDATA + lc)) { ISOException.throwIt(SW_WRONG_LENGTH); }
        byte userNameLength = apduBuffer[offset++];
        if (userNameLength > MAX_USERNAME_LENGTH || userNameLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        if ((short)(offset + userNameLength) > (short)(ISO7816.OFFSET_CDATA + lc)) { ISOException.throwIt(SW_WRONG_LENGTH); }
        
        // 创建临时缓冲区保存用户名 (使用RAM而非EEPROM)
        byte[] tempUserName = JCSystem.makeTransientByteArray(userNameLength, JCSystem.CLEAR_ON_RESET);
        Util.arrayCopy(apduBuffer, offset, tempUserName, (short)0, userNameLength);
        offset += userNameLength;
        
        // 获取并验证地址长度
        if (offset >= (short)(ISO7816.OFFSET_CDATA + lc)) { ISOException.throwIt(SW_WRONG_LENGTH); }
        byte addrLength = apduBuffer[offset++];
        if (addrLength > MAX_ADDR_LENGTH || addrLength <= 0) {
            ISOException.throwIt(SW_WRONG_LENGTH);
        }
        if ((short)(offset + addrLength) > (short)(ISO7816.OFFSET_CDATA + lc)) { ISOException.throwIt(SW_WRONG_LENGTH); }
        
        // 创建临时缓冲区保存地址
        byte[] tempAddr = JCSystem.makeTransientByteArray(addrLength, JCSystem.CLEAR_ON_RESET);
        Util.arrayCopy(apduBuffer, offset, tempAddr, (short)0, addrLength);
        offset += addrLength;
        
        // 获取签名占位符长度并跳过其数据
        if (offset >= (short)(ISO7816.OFFSET_CDATA + lc)) { ISOException.throwIt(SW_WRONG_LENGTH); }
        byte signaturePlaceholderLength = apduBuffer[offset++];
        // 检查数据长度是否足够包含声明的占位符长度
        if ((short)(offset + signaturePlaceholderLength) > (short)(ISO7816.OFFSET_CDATA + lc)) {
             ISOException.throwIt(SW_WRONG_LENGTH); // 提供的总数据长度不足
        }
        // offset += signaturePlaceholderLength; // 跳过占位符字段 (实际上不需要移动offset，因为后面没有数据了)
        
        // 查找匹配的记录
        byte foundIndex = findRecord(tempUserName, userNameLength, tempAddr, addrLength);
        
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
        apduBuffer[0] = (byte)((messageLength >> 8) & 0xFF); // 高字节
        apduBuffer[1] = (byte)(messageLength & 0xFF);        // 低字节
        apdu.setOutgoingAndSend((short)0, (short)2);
    }
    
    /**
     * 处理读取数据继续 - 返回消息数据块
     * @param apdu APDU命令对象
     */
    private void processReadDataContinue(APDU apdu) {
        // 验证当前必须处于读取操作状态
        if (currentOperation != OP_READ) {
            ISOException.throwIt(SW_CONDITIONS_NOT_SATISFIED);
        }
        
        byte[] apduBuffer = apdu.getBuffer();
        
        // 检查是否已读取完整个消息
        short messageLength = messageLengths[currentRecordIndex];
        if (currentMessageOffset >= messageLength) {
            // 所有数据都已读取，重置状态并返回操作完成状态
            resetOperationState();
            // 默认返回 9000
            return; 
        }
        
        // 计算本次返回的数据块大小
        short le = apdu.setOutgoing(); // 获取客户端期望的长度
        short chunkSize;
        short remaining = (short)(messageLength - currentMessageOffset);
        
        // 确定实际发送的块大小
        chunkSize = (le < remaining) ? le : remaining;
        
        // 从永久存储复制消息块到APDU缓冲区
        short messageOffset = (short)(currentRecordIndex * MAX_MESSAGE_LENGTH + currentMessageOffset);
        Util.arrayCopy(messages, messageOffset, apduBuffer, (short)0, chunkSize);
        
        // 更新已处理的消息偏移量
        currentMessageOffset += chunkSize;
        
        // 发送数据块
        apdu.setOutgoingLength(chunkSize);
        apdu.sendBytes((short)0, chunkSize);
        
        // 检查是否还有更多数据
        if (currentMessageOffset < messageLength) {
             // 计算剩余字节数，确保不超过 255 (因为状态码低位只有1字节)
             short remainingBytesForSW = (short)(messageLength - currentMessageOffset);
             byte sw2 = (remainingBytesForSW > 255) ? (byte)0xFF : (byte)remainingBytesForSW;
             ISOException.throwIt((short)(SW_MORE_DATA_AVAILABLE | sw2)); // 返回 61xx
        } else {
            // 如果所有数据都已发送，自动重置操作状态
            resetOperationState();
            // 默认返回正常完成状态 9000 (由框架处理)
        }
    }
    
    /**
     * 处理读取数据完成 - 显式结束读取操作
     * @param apdu APDU命令对象
     */
    private void processReadDataFinalize(APDU apdu) {
        // 允许在读取操作中或完成后调用
        if (currentOperation != OP_READ && currentOperation != OP_NONE) { 
            ISOException.throwIt(SW_CONDITIONS_NOT_SATISFIED);
        }
        
        // 重置读取操作状态为空闲
        resetOperationState();
        
        // 返回成功状态 - 操作完成 (默认 9000)
    }

    /**
     * 根据用户名和地址查找记录索引
     * @param userName 用户名字节数组
     * @param userNameLen 用户名长度
     * @param addr 地址字节数组
     * @param addrLen 地址长度
     * @return 找到的记录索引，未找到则返回 -1
     */
    private byte findRecord(byte[] userName, short userNameLen, byte[] addr, short addrLen) {
        for (byte i = 0; i < recordCount; i++) {
            // 检查长度是否匹配
            if (userNameLengths[i] == userNameLen && addressLengths[i] == addrLen) {
                // 比较用户名
                short currentUserNameOffset = (short)(i * MAX_USERNAME_LENGTH);
                if (Util.arrayCompare(userName, (short)0, userNames, currentUserNameOffset, userNameLen) == 0) {
                    // 比较地址
                    short currentAddrOffset = (short)(i * MAX_ADDR_LENGTH);
                    if (Util.arrayCompare(addr, (short)0, addresses, currentAddrOffset, addrLen) == 0) {
                        return i; // 找到匹配记录
                    }
                }
            }
        }
        return -1; // 未找到
    }

    /**
     * 重置当前操作状态为空闲
     */
    private void resetOperationState() {
        currentOperation = OP_NONE;
        currentMessageOffset = 0;
        currentRecordIndex = -1; // 或其他无效值
        // bufferOffset = 0; // 如果需要
    }
}
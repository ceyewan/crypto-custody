/**
 * 安全芯片存储Applet - 用于在JavaCard智能卡上安全存储和检索数据
 * 
 * 该Applet实现了一个简单的数据存储和检索系统。
 * 主要功能包括:
 * 1. 存储固定长度的用户名(32字节)、地址(64字节)和消息数据(32字节)
 * 2. 通过用户名和地址检索数据
 * 3. 支持覆盖已存在的(userName, Addr)对的数据
 * 
 * @author Security Chip Team
 * @version 2.0 
 */

 package securitychip;

 import javacard.framework.*;
 
 public class SecurityChipApplet extends Applet {
     // APDU指令常量
     private static final byte INS_STORE_DATA = (byte) 0x10;  // 存储数据命令
     private static final byte INS_READ_DATA = (byte) 0x20;   // 读取数据命令
     
     // 状态常量
     private static final short SW_RECORD_NOT_FOUND = (short) 0x6A83;  // 记录未找到
     private static final short SW_FILE_FULL = ISO7816.SW_FILE_FULL;   // 存储空间已满
     
     // 存储限制常量
     private static final byte MAX_RECORDS = 100;          // 最大记录数量
     private static final byte USERNAME_LENGTH = 32;      // 用户名固定长度
     private static final byte ADDR_LENGTH = 64;          // 地址固定长度
     private static final byte MESSAGE_LENGTH = 32;       // 消息固定长度
     
     // 数据存储结构
     private byte[] userNames;    // 所有用户名存储数组
     private byte[] addresses;    // 所有地址存储数组
     private byte[] messages;     // 所有消息存储数组
     private byte recordCount;    // 当前记录数量
     
     /**
      * 私有构造方法 - 初始化Applet
      */
     private SecurityChipApplet() {
         // 初始化存储结构 - 固定长度数组
         userNames = new byte[MAX_RECORDS * USERNAME_LENGTH];
         addresses = new byte[MAX_RECORDS * ADDR_LENGTH];
         messages = new byte[MAX_RECORDS * MESSAGE_LENGTH];
         recordCount = 0;
         
         register();
     }
     
     /**
      * 安装方法 - JavaCard框架调用此静态方法安装Applet
      */
     public static void install(byte[] bArray, short bOffset, byte bLength) {
         new SecurityChipApplet();
     }
     
     /**
      * 处理APDU命令 - JavaCard框架的入口点
      */
     public void process(APDU apdu) {
         if (selectingApplet()) {
             return;
         }
         
         byte[] apduBuffer = apdu.getBuffer();
         byte ins = apduBuffer[ISO7816.OFFSET_INS];
         
         switch (ins) {
             case INS_STORE_DATA:
                 processStoreData(apdu);
                 break;
             case INS_READ_DATA:
                 processReadData(apdu);
                 break;
             default:
                 ISOException.throwIt(ISO7816.SW_INS_NOT_SUPPORTED);
         }
     }
     
     /**
      * 处理存储数据 - 一次性存储完整记录
      * 
      * APDU格式: [CLA][INS][P1][P2][Lc][userName(32)][addr(64)][message(32)]
      */
     private void processStoreData(APDU apdu) {
         byte[] apduBuffer = apdu.getBuffer();
         short dataLength = apdu.setIncomingAndReceive();
         
         // 验证数据长度
         if (dataLength != (short)(USERNAME_LENGTH + ADDR_LENGTH + MESSAGE_LENGTH)) {
             ISOException.throwIt(ISO7816.SW_WRONG_LENGTH);
         }
         
         short offset = ISO7816.OFFSET_CDATA;
         
         // 检查是否已存在相同(userName, addr)的记录
         byte existingIndex = findRecord(
             apduBuffer, offset, 
             apduBuffer, (short)(offset + USERNAME_LENGTH)
         );
         
         byte recordIndex;
         if (existingIndex != -1) {
             // 找到匹配记录，覆盖现有数据
             recordIndex = existingIndex;
         } else {
             // 检查是否有空间存储新记录
             if (recordCount >= MAX_RECORDS) {
                 ISOException.throwIt(SW_FILE_FULL);
             }
             recordIndex = recordCount;
             recordCount++; // 增加记录数
         }
         
         // 保存用户名
         short userNameOffset = (short)(recordIndex * USERNAME_LENGTH);
         Util.arrayCopy(apduBuffer, offset, userNames, userNameOffset, USERNAME_LENGTH);
         offset += USERNAME_LENGTH;
         
         // 保存地址
         short addrOffset = (short)(recordIndex * ADDR_LENGTH);
         Util.arrayCopy(apduBuffer, offset, addresses, addrOffset, ADDR_LENGTH);
         offset += ADDR_LENGTH;
         
         // 保存消息
         short messageOffset = (short)(recordIndex * MESSAGE_LENGTH);
         Util.arrayCopy(apduBuffer, offset, messages, messageOffset, MESSAGE_LENGTH);
         
         // 构建响应：记录索引 + 记录总数
         apduBuffer[0] = recordIndex;
         apduBuffer[1] = recordCount;
         
         apdu.setOutgoingAndSend((short)0, (short)2);
     }
     
     /**
      * 处理读取数据 - 一次性读取完整记录
      * 
      * APDU格式: [CLA][INS][P1][P2][Lc][userName(32)][addr(64)][sign(64)]
      */
     private void processReadData(APDU apdu) {
         byte[] apduBuffer = apdu.getBuffer();
         short dataLength = apdu.setIncomingAndReceive();
         
         // 验证最小数据长度
         if (dataLength < (short)(USERNAME_LENGTH + ADDR_LENGTH)) {
             ISOException.throwIt(ISO7816.SW_WRONG_LENGTH);
         }
         
         short offset = ISO7816.OFFSET_CDATA;
         
         // 查找匹配的记录
         byte foundIndex = findRecord(
             apduBuffer, offset, 
             apduBuffer, (short)(offset + USERNAME_LENGTH)
         );
         
         // 如果没有找到匹配记录，返回记录未找到状态
         if (foundIndex == -1) {
             ISOException.throwIt(SW_RECORD_NOT_FOUND);
         }
         
         // 获取消息数据并返回
         short messageOffset = (short)(foundIndex * MESSAGE_LENGTH);
         Util.arrayCopyNonAtomic(messages, messageOffset, apduBuffer, (short)0, MESSAGE_LENGTH);
         
         // 发送响应
         apdu.setOutgoingAndSend((short)0, MESSAGE_LENGTH);
     }
 
     /**
      * 根据用户名和地址查找记录索引
      * 
      * @param userNameArray 用户名所在的数组
      * @param userNameOffset 用户名在数组中的起始位置
      * @param addrArray 地址所在的数组
      * @param addrOffset 地址在数组中的起始位置
      * @return 找到的记录索引，未找到则返回 -1
      */
     private byte findRecord(byte[] userNameArray, short userNameOffset, 
                            byte[] addrArray, short addrOffset) {
         for (byte i = 0; i < recordCount; i++) {
             // 比较用户名
             short currentUserNameOffset = (short)(i * USERNAME_LENGTH);
             if (Util.arrayCompare(userNameArray, userNameOffset, userNames, currentUserNameOffset, USERNAME_LENGTH) == 0) {
                 // 比较地址
                 short currentAddrOffset = (short)(i * ADDR_LENGTH);
                 if (Util.arrayCompare(addrArray, addrOffset, addresses, currentAddrOffset, ADDR_LENGTH) == 0) {
                     return i; // 找到匹配记录
                 }
             }
         }
         return -1; // 未找到
     }
 }
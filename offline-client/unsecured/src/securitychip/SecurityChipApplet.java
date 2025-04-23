/**
 * 安全芯片存储Applet - 用于在JavaCard智能卡上安全存储和检索数据
 * 
 * 该Applet实现了一个简单的数据存储和检索系统。
 * 主要功能包括:
 * 1. 存储固定长度的用户名(32字节)、地址(64字节)和消息数据(32字节)
 * 2. 通过用户名和地址检索数据
 * 3. 支持覆盖已存在的(userName, Addr)对的数据
 * 4. 支持删除已存在的数据
 * 
 * @author Security Chip Team
 * @version 2.1 
 */

 package securitychip;

 import javacard.framework.*;
 
 public class SecurityChipApplet extends Applet {
     // APDU指令常量
     private static final byte INS_STORE_DATA = (byte) 0x10;  // 存储数据命令
     private static final byte INS_READ_DATA = (byte) 0x20;   // 读取数据命令
     private static final byte INS_DELETE_DATA = (byte) 0x30; // 删除数据命令
     
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
     private byte[] existFlags;   // 记录存在标志数组
     private byte recordCount;    // 当前记录数量
     private byte nextFreeSlot;   // 下一个可用槽位
     
     /**
      * 私有构造方法 - 初始化Applet
      */
     private SecurityChipApplet() {
         // 初始化存储结构 - 固定长度数组
         userNames = new byte[MAX_RECORDS * USERNAME_LENGTH];
         addresses = new byte[MAX_RECORDS * ADDR_LENGTH];
         messages = new byte[MAX_RECORDS * MESSAGE_LENGTH];
         existFlags = new byte[MAX_RECORDS]; // 0表示空槽，1表示有效记录
         recordCount = 0;
         nextFreeSlot = 0;
         
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
             case INS_DELETE_DATA:
                 processDeleteData(apdu);
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
             
             // 查找下一个可用槽位
             recordIndex = findNextFreeSlot();
             if (recordIndex == -1) {
                 ISOException.throwIt(SW_FILE_FULL);
             }
             
             existFlags[recordIndex] = 1; // 标记为已使用
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
         
         // 更新下一个可用槽位
         updateNextFreeSlot();
         
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
      * 处理删除数据 - 根据用户名和地址删除记录
      * 
      * APDU格式: [CLA][INS][P1][P2][Lc][userName(32)][addr(64)]
      */
     private void processDeleteData(APDU apdu) {
         byte[] apduBuffer = apdu.getBuffer();
         short dataLength = apdu.setIncomingAndReceive();
         
         // 验证数据长度
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
         
         // 标记记录为已删除
         existFlags[foundIndex] = 0;
         recordCount--; // 减少记录数量
         
         // 如果删除的是最低的索引，更新nextFreeSlot
         if (foundIndex < nextFreeSlot) {
             nextFreeSlot = foundIndex;
         }
         
         // 构建响应：删除的记录索引 + 剩余记录总数
         apduBuffer[0] = foundIndex;
         apduBuffer[1] = recordCount;
         
         apdu.setOutgoingAndSend((short)0, (short)2);
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
         for (byte i = 0; i < MAX_RECORDS; i++) {
             // 检查记录是否存在
             if (existFlags[i] == 0) {
                 continue; // 跳过已删除的记录
             }
             
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
     
     /**
      * 查找下一个可用槽位
      * 
      * @return 下一个可用槽位索引，如果没有可用槽位则返回 -1
      */
     private byte findNextFreeSlot() {
         // 从nextFreeSlot开始查找
         for (byte i = nextFreeSlot; i < MAX_RECORDS; i++) {
             if (existFlags[i] == 0) {
                 return i;
             }
         }
         
         // 如果没有找到，从头开始查找
         for (byte i = 0; i < nextFreeSlot; i++) {
             if (existFlags[i] == 0) {
                 return i;
             }
         }
         
         return -1; // 没有可用槽位
     }
     
     /**
      * 更新下一个可用槽位变量
      */
     private void updateNextFreeSlot() {
         nextFreeSlot = findNextFreeSlot();
     }
 }
/**
 * 安全芯片存储Applet - 用于在JavaCard智能卡上安全存储和检索数据
 * 
 * 该Applet实现了一个简单的数据存储和检索系统。
 * 主要功能包括:
 * 1. 存储固定长度的用户名(32字节)、以太坊地址(20字节)和消息数据(32字节)
 * 2. 通过用户名和地址检索数据
 * 3. 支持覆盖已存在的(userName, Addr)对的数据
 * 4. 支持删除已存在的数据
 * 5. 使用ECDSA签名验证操作的安全性
 * 
 * @author Security Chip Team
 * @version 2.3 
 */

package securitychip;

import javacard.framework.*;
import javacard.security.*;
import javacardx.crypto.*;

public class SecurityChipApplet extends Applet {
    // APDU指令常量
    private static final byte INS_STORE_DATA = (byte) 0x10; // 存储数据命令
    private static final byte INS_READ_DATA = (byte) 0x20; // 读取数据命令
    private static final byte INS_DELETE_DATA = (byte) 0x30; // 删除数据命令

    // 状态常量
    private static final short SW_RECORD_NOT_FOUND = (short) 0x6A83; // 记录未找到
    private static final short SW_FILE_FULL = ISO7816.SW_FILE_FULL; // 存储空间已满
    private static final short SW_SIGNATURE_INVALID = (short) 0x6982; // 签名无效

    // 存储限制常量
    private static final byte MAX_RECORDS = 100; // 最大记录数量
    private static final byte USERNAME_LENGTH = 32; // 用户名固定长度
    private static final byte ADDR_LENGTH = 20; // 地址固定长度
    private static final byte MESSAGE_LENGTH = 32; // 消息固定长度
    private static final byte MAX_SIGNATURE_LENGTH = 72; // ECDSA DER格式签名最大长度

    // ECDSA公钥 (NIST P-256 / secp256r1曲线)
    private static final byte[] EC_PUBLIC_KEY_BYTES = {
            (byte) 0x04, (byte) 0x79, (byte) 0x7C, (byte) 0xEF, (byte) 0x50, (byte) 0x1E, (byte) 0x84, (byte) 0xF2,
            (byte) 0xD3, (byte) 0x15, (byte) 0xBE, (byte) 0xDB, (byte) 0xDE, (byte) 0xF0, (byte) 0xD3, (byte) 0x0B,
            (byte) 0xCF, (byte) 0x3A, (byte) 0x16, (byte) 0x30, (byte) 0xA3, (byte) 0x79, (byte) 0x81, (byte) 0x51,
            (byte) 0xD2, (byte) 0xBC, (byte) 0xF7, (byte) 0xA3, (byte) 0x21, (byte) 0x3A, (byte) 0xD4, (byte) 0x22,
            (byte) 0x17, (byte) 0x64, (byte) 0x45, (byte) 0x01, (byte) 0x90, (byte) 0x5F, (byte) 0x0C, (byte) 0x58,
            (byte) 0xC9, (byte) 0x53, (byte) 0x4E, (byte) 0x3E, (byte) 0xAE, (byte) 0x69, (byte) 0x63, (byte) 0x43,
            (byte) 0x3A, (byte) 0xBE, (byte) 0xEE, (byte) 0x3D, (byte) 0x25, (byte) 0xB5, (byte) 0x87, (byte) 0xCD,
            (byte) 0xC1, (byte) 0x39, (byte) 0x9D, (byte) 0xD0, (byte) 0x19, (byte) 0x86, (byte) 0xBB, (byte) 0x1D,
            (byte) 0x12
    };

    private static final byte[] P = {
            (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x01,
            (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00,
            (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF,
            (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF
    };

    private static final byte[] A = {
            (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x01,
            (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00,
            (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF,
            (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFC
    };
    private static final byte[] B = {
            (byte) 0x5A, (byte) 0xC6, (byte) 0x35, (byte) 0xD8, (byte) 0xAA, (byte) 0x3A, (byte) 0x93, (byte) 0xE7,
            (byte) 0xB3, (byte) 0xEB, (byte) 0xBD, (byte) 0x55, (byte) 0x76, (byte) 0x98, (byte) 0x86, (byte) 0xBC,
            (byte) 0x65, (byte) 0x1D, (byte) 0x06, (byte) 0xB0, (byte) 0xCC, (byte) 0x53, (byte) 0xB0, (byte) 0xF6,
            (byte) 0x3B, (byte) 0xCE, (byte) 0x3C, (byte) 0x3E, (byte) 0x27, (byte) 0xD2, (byte) 0x60, (byte) 0x4B
    };
    private static final byte[] G = {
            (byte) 0x04,
            (byte) 0x6B, (byte) 0x17, (byte) 0xD1, (byte) 0xF2, (byte) 0xE1, (byte) 0x2C, (byte) 0x42, (byte) 0x47,
            (byte) 0xF8, (byte) 0xBC, (byte) 0xE6, (byte) 0xE5, (byte) 0x63, (byte) 0xA4, (byte) 0x40, (byte) 0xF2,
            (byte) 0x77, (byte) 0x03, (byte) 0x7D, (byte) 0x81, (byte) 0x2D, (byte) 0xEB, (byte) 0x33, (byte) 0xA0,
            (byte) 0xF4, (byte) 0xA1, (byte) 0x39, (byte) 0x45, (byte) 0xD8, (byte) 0x98, (byte) 0xC2, (byte) 0x96,
            (byte) 0x4F, (byte) 0xE3, (byte) 0x42, (byte) 0xE2, (byte) 0xFE, (byte) 0x1A, (byte) 0x7F, (byte) 0x9B,
            (byte) 0x8E, (byte) 0xE7, (byte) 0xEB, (byte) 0x4A, (byte) 0x7C, (byte) 0x0F, (byte) 0x9E, (byte) 0x16,
            (byte) 0x2B, (byte) 0xCE, (byte) 0x33, (byte) 0x57, (byte) 0x6B, (byte) 0x31, (byte) 0x5E, (byte) 0xCE,
            (byte) 0xCB, (byte) 0xB6, (byte) 0x40, (byte) 0x68, (byte) 0x37, (byte) 0xBF, (byte) 0x51, (byte) 0xF5
    };
    private static final byte[] N = {
            (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0x00, (byte) 0x00, (byte) 0x00, (byte) 0x00,
            (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF, (byte) 0xFF,
            (byte) 0xBC, (byte) 0xE6, (byte) 0xFA, (byte) 0xAD, (byte) 0xA7, (byte) 0x17, (byte) 0x9E, (byte) 0x84,
            (byte) 0xF3, (byte) 0xB9, (byte) 0xCA, (byte) 0xC2, (byte) 0xFC, (byte) 0x63, (byte) 0x25, (byte) 0x51
    };
    private static final short K = 0x01;

    // ECDSA验证对象
    private ECPublicKey ecPublicKey;
    private Signature ecSignature;

    // 数据存储结构
    private byte[] userNames; // 所有用户名存储数组
    private byte[] addresses; // 所有地址存储数组
    private byte[] messages; // 所有消息存储数组
    private byte[] existFlags; // 记录存在标志数组
    private byte recordCount; // 当前记录数量
    private byte nextFreeSlot; // 下一个可用槽位

    // 临时缓冲区，用于构建签名数据
    private byte[] tempBuffer;

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

        // 初始化临时缓冲区 - 只需要存储用户名和地址用于构建签名消息
        tempBuffer = JCSystem.makeTransientByteArray((short) (USERNAME_LENGTH + ADDR_LENGTH),
                JCSystem.CLEAR_ON_DESELECT);

        // 初始化ECDSA验证
        initializeECDSA();

        register();
    }

    /**
     * 初始化ECDSA验证组件
     */
    private void initializeECDSA() {
        // 创建ECDSA验证对象
        ecSignature = Signature.getInstance(Signature.ALG_ECDSA_SHA_256, false);

        // 创建并初始化EC公钥
        ecPublicKey = (ECPublicKey) KeyBuilder.buildKey(KeyBuilder.TYPE_EC_FP_PUBLIC, KeyBuilder.LENGTH_EC_FP_256,
                false);

        // 设置椭圆曲线参数
        ecPublicKey.setFieldFP(P, (short) 0, (short) P.length);
        ecPublicKey.setA(A, (short) 0, (short) A.length);
        ecPublicKey.setB(B, (short) 0, (short) B.length);
        ecPublicKey.setG(G, (short) 0, (short) G.length);
        ecPublicKey.setR(N, (short) 0, (short) N.length);
        ecPublicKey.setK(K);

        // 设置公钥
        ecPublicKey.setW(EC_PUBLIC_KEY_BYTES, (short) 0, (short) EC_PUBLIC_KEY_BYTES.length);

        // 初始化签名验证对象
        ecSignature.init(ecPublicKey, Signature.MODE_VERIFY);
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
     * APDU格式: [CLA][INS][P1][P2][Lc][userName(32)][addr(20)][message(32)]
     */
    private void processStoreData(APDU apdu) {
        byte[] apduBuffer = apdu.getBuffer();
        short dataLength = apdu.setIncomingAndReceive();

        // 验证数据长度
        if (dataLength != (short) (USERNAME_LENGTH + ADDR_LENGTH + MESSAGE_LENGTH)) {
            ISOException.throwIt(ISO7816.SW_WRONG_LENGTH);
        }

        short offset = ISO7816.OFFSET_CDATA;

        // 检查是否已存在相同(userName, addr)的记录
        byte existingIndex = findRecord(
                apduBuffer, offset,
                apduBuffer, (short) (offset + USERNAME_LENGTH));

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
        short userNameOffset = (short) (recordIndex * USERNAME_LENGTH);
        Util.arrayCopy(apduBuffer, offset, userNames, userNameOffset, USERNAME_LENGTH);
        offset += USERNAME_LENGTH;

        // 保存地址
        short addrOffset = (short) (recordIndex * ADDR_LENGTH);
        Util.arrayCopy(apduBuffer, offset, addresses, addrOffset, ADDR_LENGTH);
        offset += ADDR_LENGTH;

        // 保存消息
        short messageOffset = (short) (recordIndex * MESSAGE_LENGTH);
        Util.arrayCopy(apduBuffer, offset, messages, messageOffset, MESSAGE_LENGTH);

        // 更新下一个可用槽位
        updateNextFreeSlot();

        // 构建响应：记录索引 + 记录总数
        apduBuffer[0] = recordIndex;
        apduBuffer[1] = recordCount;

        apdu.setOutgoingAndSend((short) 0, (short) 2);
    }

    /**
     * 处理读取数据 - 一次性读取完整记录
     * 
     * APDU格式: [CLA][INS][P1][P2][Lc][userName(32)][addr(20)][sign_DER(变长)]
     * 注意: sign_DER是DER编码格式的ECDSA签名，最大长度为72字节
     */
    private void processReadData(APDU apdu) {
        byte[] apduBuffer = apdu.getBuffer();
        short dataLength = apdu.setIncomingAndReceive();

        // 验证数据最小长度
        if (dataLength < (short) (USERNAME_LENGTH + ADDR_LENGTH + 8)) {
            // 8是DER签名的最小长度 (序列头2字节 + r至少3字节 + s至少3字节)
            ISOException.throwIt(ISO7816.SW_WRONG_LENGTH);
        }

        short offset = ISO7816.OFFSET_CDATA;

        // 准备验证签名
        // 1. 复制用户名和地址到临时缓冲区，用于构建待签名的消息
        Util.arrayCopy(apduBuffer, offset, tempBuffer, (short) 0, (short) (USERNAME_LENGTH + ADDR_LENGTH));

        // 计算签名数据长度 (总数据长度减去用户名和地址的长度)
        short signatureLength = (short) (dataLength - USERNAME_LENGTH - ADDR_LENGTH);

        // 2. 验证签名 - 现在签名已经是DER格式
        if (!verifySignature(tempBuffer, (short) 0, (short) (USERNAME_LENGTH + ADDR_LENGTH),
                apduBuffer, (short) (offset + USERNAME_LENGTH + ADDR_LENGTH), signatureLength)) {
            ISOException.throwIt(SW_SIGNATURE_INVALID);
        }

        // 查找匹配的记录
        byte foundIndex = findRecord(
                apduBuffer, offset,
                apduBuffer, (short) (offset + USERNAME_LENGTH));

        // 如果没有找到匹配记录，返回记录未找到状态
        if (foundIndex == -1) {
            ISOException.throwIt(SW_RECORD_NOT_FOUND);
        }

        // 获取消息数据并返回
        short messageOffset = (short) (foundIndex * MESSAGE_LENGTH);
        Util.arrayCopyNonAtomic(messages, messageOffset, apduBuffer, (short) 0, MESSAGE_LENGTH);

        // 发送响应
        apdu.setOutgoingAndSend((short) 0, MESSAGE_LENGTH);
    }

    /**
     * 处理删除数据 - 根据用户名和地址删除记录
     * 
     * APDU格式: [CLA][INS][P1][P2][Lc][userName(32)][addr(20)][sign_DER(变长)]
     * 注意: sign_DER是DER编码格式的ECDSA签名，最大长度为72字节
     */
    private void processDeleteData(APDU apdu) {
        byte[] apduBuffer = apdu.getBuffer();
        short dataLength = apdu.setIncomingAndReceive();

        // 验证数据最小长度
        if (dataLength < (short) (USERNAME_LENGTH + ADDR_LENGTH + 8)) {
            // 8是DER签名的最小长度
            ISOException.throwIt(ISO7816.SW_WRONG_LENGTH);
        }

        short offset = ISO7816.OFFSET_CDATA;

        // 准备验证签名
        // 1. 复制用户名和地址到临时缓冲区，用于构建待签名的消息
        Util.arrayCopy(apduBuffer, offset, tempBuffer, (short) 0, (short) (USERNAME_LENGTH + ADDR_LENGTH));

        // 计算签名数据长度
        short signatureLength = (short) (dataLength - USERNAME_LENGTH - ADDR_LENGTH);

        // 2. 验证签名 - 现在签名已经是DER格式
        if (!verifySignature(tempBuffer, (short) 0, (short) (USERNAME_LENGTH + ADDR_LENGTH),
                apduBuffer, (short) (offset + USERNAME_LENGTH + ADDR_LENGTH), signatureLength)) {
            ISOException.throwIt(SW_SIGNATURE_INVALID);
        }

        // 查找匹配的记录
        byte foundIndex = findRecord(
                apduBuffer, offset,
                apduBuffer, (short) (offset + USERNAME_LENGTH));

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

        apdu.setOutgoingAndSend((short) 0, (short) 2);
    }

    /**
     * 验证ECDSA签名 - 直接使用DER格式
     * 
     * @param messageBuffer   消息数据缓冲区
     * @param messageOffset   消息数据偏移量
     * @param messageLength   消息数据长度
     * @param signatureBuffer DER格式签名缓冲区
     * @param signatureOffset 签名数据偏移量
     * @param signatureLength 签名数据长度
     * @return 签名是否有效
     */
    private boolean verifySignature(byte[] messageBuffer, short messageOffset, short messageLength,
            byte[] signatureBuffer, short signatureOffset, short signatureLength) {

        // 重置ECDSA验证对象
        ecSignature.init(ecPublicKey, Signature.MODE_VERIFY);

        // 验证签名 - 直接使用DER格式
        return ecSignature.verify(messageBuffer, messageOffset, messageLength,
                signatureBuffer, signatureOffset, signatureLength);
    }

    /**
     * 根据用户名和地址查找记录索引
     * 
     * @param userNameArray  用户名所在的数组
     * @param userNameOffset 用户名在数组中的起始位置
     * @param addrArray      地址所在的数组
     * @param addrOffset     地址在数组中的起始位置
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
            short currentUserNameOffset = (short) (i * USERNAME_LENGTH);
            if (Util.arrayCompare(userNameArray, userNameOffset, userNames, currentUserNameOffset,
                    USERNAME_LENGTH) == 0) {
                // 比较地址
                short currentAddrOffset = (short) (i * ADDR_LENGTH);
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
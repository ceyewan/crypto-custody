#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
安全芯片客户端脚本 - 用于与JavaCard安全芯片进行通信，存储和读取数据

该脚本与SecurityChipApplet通信，演示如何存储和检索数据。
主要功能包括:
1. 连接智能卡读卡器并选择安全芯片应用
2. 向芯片存储两份示例数据
3. 从芯片读取之前存储的数据
4. 包含签名生成和验证逻辑

使用说明：
- 安装依赖库：pip install pyscard ecdsa
- 确保读卡器已连接且JavaCard卡已插入
- 确保SecurityChipApplet已安装到卡上
- 运行脚本：python secure_chip_client.py

@author Harrick
@version 1.0
"""

from smartcard.System import readers
from smartcard.util import toHexString, toBytes
import binascii
import struct
import time
import ecdsa
from ecdsa import SigningKey, NIST192p
import logging
import sys

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# APDU命令常量
CLA = 0x80  # 应用类标识符

# 指令代码
INS_STORE_DATA_INIT = 0x10      # 存储数据初始化
INS_STORE_DATA_CONTINUE = 0x11  # 存储数据继续
INS_STORE_DATA_FINALIZE = 0x12  # 存储数据完成
INS_READ_DATA_INIT = 0x20       # 读取数据初始化
INS_READ_DATA_CONTINUE = 0x21   # 读取数据继续
INS_READ_DATA_FINALIZE = 0x22   # 读取数据完成

# 状态码
SW_SUCCESS = [0x90, 0x00]  # 操作成功
SW_MORE_DATA = [0x61, 0x00]  # 有更多数据可用

# 示例数据
TEST_DATA = [
    {
        "username": "alice123",
        "address": "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
        "message": "这是Alice的私钥备份，请安全保管。" + "A" * 10000  # 添加一些额外数据使消息更长
    },
    {
        "username": "bob456",
        "address": "3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy",
        "message": "这是Bob的私钥备份，请勿泄露。" + "B" * 20000  # 添加一些额外数据使消息更长
    }
]

# 默认的ECDSA密钥对（在实际应用中应安全存储）
# 警告：这仅用于测试，实际应用中应妥善保管私钥
PRIVATE_KEY = None
PUBLIC_KEY = None

class SecureChipClient:
    """安全芯片客户端类 - 处理与芯片的所有通信"""
    
    def __init__(self):
        """初始化客户端，连接读卡器并选择安全芯片应用"""
        self.connection = None
        self.reader = None
        self.connect_to_card()
        self.select_applet()
        
    def connect_to_card(self):
        """连接到智能卡读卡器并建立连接"""
        try:
            # 获取可用读卡器列表
            reader_list = readers()
            if not reader_list:
                logger.error("未找到读卡器。请确保读卡器已连接。")
                sys.exit(1)
                
            # 使用第一个可用的读卡器
            self.reader = reader_list[0]
            logger.info(f"使用读卡器: {self.reader}")
            
            # 连接到卡片
            self.connection = self.reader.createConnection()
            self.connection.connect()
            logger.info(f"卡片连接成功，ATR: {toHexString(self.connection.getATR())}")
            
        except Exception as e:
            logger.error(f"连接读卡器或智能卡时出错: {e}")
            sys.exit(1)
    
    def select_applet(self):
        """选择安全芯片应用"""
        # SecurityChipApplet的AID (应用标识符) - 需要与安装时的AID匹配
        # 这里使用一个示例AID，请根据实际安装的AID进行修改
        aid = [0xA0, 0x00, 0x00, 0x00, 0x62, 0x03, 0x01, 0x0C, 0x01, 0x01]
        
        # 构建选择应用的APDU命令
        select_cmd = [0x00, 0xA4, 0x04, 0x00, len(aid)] + aid
        
        try:
            # 发送命令并接收响应
            response, sw1, sw2 = self.connection.transmit(select_cmd)
            
            # 检查响应状态
            if [sw1, sw2] == SW_SUCCESS:
                logger.info("安全芯片应用选择成功")
            else:
                logger.error(f"应用选择失败，状态码: {sw1:02X} {sw2:02X}")
                sys.exit(1)
                
        except Exception as e:
            logger.error(f"选择应用时出错: {e}")
            sys.exit(1)
    
    def send_apdu(self, cla, ins, p1, p2, data=None, le=None):
        """发送APDU命令并接收响应
        
        参数:
            cla: 类字节
            ins: 指令字节
            p1, p2: 参数字节
            data: 要发送的数据（可选）
            le: 期望接收的数据长度（可选）
            
        返回:
            (response_data, [sw1, sw2]): 响应数据和状态码
        """
        command = [cla, ins, p1, p2]
        
        # 添加数据字段（如果有）
        if data:
            command.append(len(data))
            command.extend(data)
        elif le is not None:
            # 如果没有数据但有Le，需要添加Lc=0
            command.append(0)
            
        # 添加Le字段（如果有）
        if le is not None:
            command.append(le)
            
        try:
            logger.debug(f"发送APDU: {toHexString(command)}")
            response, sw1, sw2 = self.connection.transmit(command)
            logger.debug(f"接收响应: {toHexString(response) if response else '(无数据)'}, 状态: {sw1:02X} {sw2:02X}")
            return response, [sw1, sw2]
        except Exception as e:
            logger.error(f"发送APDU时出错: {e}")
            return [], [0x6F, 0x00]  # 返回一个通用错误状态码
    
    def store_data(self, username, address, message):
        """向安全芯片存储数据
        
        参数:
            username: 用户名
            address: 地址
            message: 要存储的消息
            
        返回:
            成功返回True，失败返回False
        """
        logger.info(f"开始存储数据 - 用户: {username}, 地址: {address}")
        
        # 将字符串转换为字节数组
        username_bytes = username.encode('utf-8')
        address_bytes = address.encode('utf-8')
        message_bytes = message.encode('utf-8')
        
        # 步骤1: 初始化存储过程
        init_data = [len(username_bytes)] + list(username_bytes) + [len(address_bytes)] + list(address_bytes)
        response, status = self.send_apdu(CLA, INS_STORE_DATA_INIT, 0, 0, init_data)
        
        if status != SW_SUCCESS:
            logger.error(f"存储初始化失败，状态码: {status[0]:02X} {status[1]:02X}")
            return False
            
        logger.info("存储初始化成功，开始传输消息数据")
        
        # 步骤2: 分块传输消息数据
        chunk_size = 200  # 每次传输的数据块大小
        for i in range(0, len(message_bytes), chunk_size):
            # 获取当前数据块
            chunk = message_bytes[i:i+chunk_size]
            # 发送数据块
            response, status = self.send_apdu(CLA, INS_STORE_DATA_CONTINUE, 0, 0, list(chunk))
            
            if status != SW_SUCCESS:
                logger.error(f"数据块传输失败，状态码: {status[0]:02X} {status[1]:02X}")
                return False
                
            logger.info(f"已传输 {i + len(chunk)}/{len(message_bytes)} 字节的消息数据")
        
        # 步骤3: 完成存储过程
        response, status = self.send_apdu(CLA, INS_STORE_DATA_FINALIZE, 0, 0, [])
        
        if status != SW_SUCCESS:
            logger.error(f"存储完成失败，状态码: {status[0]:02X} {status[1]:02X}")
            return False
        
        logger.info("数据存储成功完成")
        return True
    
    def read_data(self, username, address):
        """从安全芯片读取数据
        
        参数:
            username: 用户名
            address: 地址
            
        返回:
            成功返回消息内容，失败返回None
        """
        logger.info(f"开始读取数据 - 用户: {username}, 地址: {address}")
        
        # 生成用于身份验证的签名
        username_bytes = username.encode('utf-8')
        address_bytes = address.encode('utf-8')
        
        # 1. 将用户名和地址连接起来作为验证数据
        data_to_sign = username_bytes + address_bytes
        
        # 2. 使用ECDSA私钥对数据进行签名
        signature = generate_signature(data_to_sign)
        
        # 3. 构建读取初始化命令的数据
        init_data = [len(username_bytes)] + list(username_bytes) + \
                    [len(address_bytes)] + list(address_bytes) + \
                    [len(signature)] + list(signature)
        
        # 步骤1: 初始化读取过程
        response, status = self.send_apdu(CLA, INS_READ_DATA_INIT, 0, 0, init_data)
        
        if status[0] != 0x90:
            logger.error(f"读取初始化失败，状态码: {status[0]:02X} {status[1]:02X}")
            return None
            
        # 从响应中获取消息总长度
        if len(response) < 2:
            logger.error("响应数据格式错误，无法获取消息长度")
            return None
            
        message_length = (response[0] << 8) | response[1]
        logger.info(f"消息总长度: {message_length} 字节")
        
        # 步骤2: 分块读取消息数据
        message_data = bytearray()
        more_data = True
        
        while more_data:
            response, status = self.send_apdu(CLA, INS_READ_DATA_CONTINUE, 0, 0, None, 0)
            
            if status[0] == 0x90:  # 成功并且没有更多数据
                message_data.extend(response)
                more_data = False
            elif status[0] == 0x61:  # 成功并且还有更多数据
                message_data.extend(response)
                logger.info(f"已接收 {len(message_data)}/{message_length} 字节的消息数据")
            else:
                logger.error(f"读取数据失败，状态码: {status[0]:02X} {status[1]:02X}")
                return None
        
        # 步骤3: 完成读取过程（可选，因为芯片会在最后一块数据发送后自动完成）
        response, status = self.send_apdu(CLA, INS_READ_DATA_FINALIZE, 0, 0, [])
        
        if status != SW_SUCCESS:
            logger.warning(f"读取完成命令返回非成功状态: {status[0]:02X} {status[1]:02X}")
        
        # 解码并返回消息内容
        message = message_data.decode('utf-8')
        logger.info("数据读取成功完成")
        return message

def generate_keypair():
    """生成ECDSA密钥对"""
    global PRIVATE_KEY, PUBLIC_KEY
    
    # 使用NIST P-192曲线生成密钥对
    PRIVATE_KEY = SigningKey.generate(curve=NIST192p)
    PUBLIC_KEY = PRIVATE_KEY.verifying_key
    
    logger.info("已生成ECDSA密钥对")

def generate_signature(data):
    """使用ECDSA私钥生成签名
    
    参数:
        data: 要签名的数据（字节类型）
        
    返回:
        signature: 签名数据（字节数组）
    """
    global PRIVATE_KEY
    
    if PRIVATE_KEY is None:
        generate_keypair()
    
    # 生成签名
    signature = PRIVATE_KEY.sign(data)
    
    logger.debug(f"已生成签名，长度: {len(signature)} 字节")
    return signature

def main():
    """主函数 - 演示存储和读取数据"""
    # 生成密钥对
    generate_keypair()
    
    try:
        # 创建安全芯片客户端
        client = SecureChipClient()
        
        # 存储两份测试数据
        for i, data in enumerate(TEST_DATA):
            logger.info(f"=== 存储数据 #{i+1} ===")
            success = client.store_data(
                data["username"],
                data["address"],
                data["message"]
            )
            if success:
                logger.info(f"数据 #{i+1} 存储成功")
            else:
                logger.error(f"数据 #{i+1} 存储失败")
            
            # 短暂暂停，让智能卡有时间处理
            time.sleep(1)
        
        # 读取两份测试数据
        for i, data in enumerate(TEST_DATA):
            logger.info(f"=== 读取数据 #{i+1} ===")
            message = client.read_data(
                data["username"],
                data["address"]
            )
            if message:
                logger.info(f"读取到的消息: {message[:50]}...（共{len(message)}字节）")
                # 验证消息是否匹配
                if message == data["message"]:
                    logger.info("✓ 消息内容验证成功")
                else:
                    logger.error("✗ 消息内容验证失败")
            else:
                logger.error("读取数据失败")
            
            # 短暂暂停，让智能卡有时间处理
            time.sleep(1)
            
    except KeyboardInterrupt:
        logger.info("操作被用户中断")
    except Exception as e:
        logger.error(f"发生错误: {e}")
    finally:
        logger.info("脚本执行完成")

if __name__ == "__main__":
    main()
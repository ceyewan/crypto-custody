#!/usr/bin/env python3
# filepath: test_security_chip.py
"""
安全芯片Applet测试脚本
测试写入、读取和删除操作是否正常工作
"""

import time
import binascii
import os
from smartcard.System import readers
from smartcard.util import toHexString, toBytes
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import ec, utils
from cryptography.hazmat.primitives.serialization import load_pem_private_key

# 常量定义
CLA = 0x80  # 类字节
INS_STORE_DATA = 0x10  # 存储数据指令
INS_READ_DATA = 0x20   # 读取数据指令
INS_DELETE_DATA = 0x30  # 删除数据指令

USERNAME_LENGTH = 32   # 用户名固定长度
ADDR_LENGTH = 64       # 地址固定长度
MESSAGE_LENGTH = 32    # 消息数据固定长度
MAX_SIGNATURE_LENGTH = 72  # DER格式签名的最大长度

# 私钥文件路径 - 使用项目根目录下的ec_private_key.pem
PRIVATE_KEY_FILE = 'ec_private_key.pem'


def load_key():
    """从PEM文件加载ECDSA私钥"""
    if not os.path.exists(PRIVATE_KEY_FILE):
        raise FileNotFoundError(
            f"找不到私钥文件: {PRIVATE_KEY_FILE}，请先运行generate_keys.py生成密钥")

    with open(PRIVATE_KEY_FILE, 'rb') as key_file:
        return load_pem_private_key(key_file.read(), password=None)


def sign_data(private_key, data):
    """使用ECDSA对数据进行签名，直接返回DER格式签名"""
    # 使用私钥对数据进行签名，直接返回DER格式
    signature = private_key.sign(
        data,
        ec.ECDSA(hashes.SHA256())
    )

    print(f"DER格式签名长度: {len(signature)} 字节")
    print(f"签名内容(hex): {binascii.hexlify(signature).decode()}")

    return signature


def pad_data(data, length):
    """将数据填充到指定长度"""
    if isinstance(data, str):
        data = data.encode('utf-8')
    return data.ljust(length, b'\x00')


def connect_card():
    """连接到智能卡并选择Applet"""
    try:
        # 获取读卡器
        available_readers = readers()
        if not available_readers:
            print("没有找到读卡器")
            return None

        print(f"找到读卡器: {available_readers}")
        reader = available_readers[0]  # 使用第一个读卡器

        # 连接读卡器
        connection = reader.createConnection()
        connection.connect()
        print(f"已连接到卡片: {toHexString(connection.getATR())}")

        # 选择已安装的Applet
        # AID：A000000062CF0101 (从安装日志中获取)
        aid = [0xA0, 0x00, 0x00, 0x00, 0x62, 0xCF, 0x01, 0x01]
        select_command = [0x00, 0xA4, 0x04, 0x00, len(aid)] + aid
        response, sw1, sw2 = connection.transmit(select_command)

        if (sw1, sw2) == (0x90, 0x00):
            print("成功选择Applet")
            return connection
        else:
            print(
                f"选择Applet失败: SW={hex(sw1)[2:].zfill(2)}{hex(sw2)[2:].zfill(2)}")
            return None

    except Exception as e:
        print(f"连接错误: {str(e)}")
        return None


def store_data(connection, username, address, message):
    """存储数据记录"""
    print("\n==== 测试存储数据 ====")
    print(f"用户名: {username}")
    print(f"地址: {address}")
    print(f"消息: {message}")

    # 准备数据
    username_bytes = pad_data(username, USERNAME_LENGTH)
    address_bytes = pad_data(address, ADDR_LENGTH)
    message_bytes = pad_data(message, MESSAGE_LENGTH)

    data = username_bytes + address_bytes + message_bytes

    # 构造APDU命令
    command = [CLA, INS_STORE_DATA, 0x00, 0x00, len(data)] + list(data)

    # 发送命令
    try:
        response, sw1, sw2 = connection.transmit(command)

        if (sw1, sw2) == (0x90, 0x00):
            print(f"存储成功: 记录索引={response[0]}, 记录总数={response[1]}")
            return True
        else:
            print(f"存储失败: SW={hex(sw1)[2:].zfill(2)}{hex(sw2)[2:].zfill(2)}")
            return False
    except Exception as e:
        print(f"存储错误: {str(e)}")
        return False


def read_data(connection, username, address):
    """读取数据记录"""
    print("\n==== 测试读取数据 ====")
    print(f"用户名: {username}")
    print(f"地址: {address}")

    # 准备数据
    username_bytes = pad_data(username, USERNAME_LENGTH)
    address_bytes = pad_data(address, ADDR_LENGTH)

    # 生成签名 - 使用DER格式
    private_key = load_key()
    data_to_sign = username_bytes + address_bytes
    signature = sign_data(private_key, data_to_sign)

    data = username_bytes + address_bytes + signature

    # 构造APDU命令
    command = [CLA, INS_READ_DATA, 0x00, 0x00, len(data)] + list(data)

    # 发送命令
    try:
        response, sw1, sw2 = connection.transmit(command)

        if (sw1, sw2) == (0x90, 0x00):
            # 去除尾部的填充
            message = bytes(response).rstrip(b'\x00')
            try:
                message_text = message.decode('utf-8')
            except:
                message_text = "二进制数据:" + binascii.hexlify(message).decode()

            print(f"读取成功: {message_text}")
            return message
        else:
            print(f"读取失败: SW={hex(sw1)[2:].zfill(2)}{hex(sw2)[2:].zfill(2)}")
            return None
    except Exception as e:
        print(f"读取错误: {str(e)}")
        return None


def delete_data(connection, username, address):
    """删除数据记录"""
    print("\n==== 测试删除数据 ====")
    print(f"用户名: {username}")
    print(f"地址: {address}")

    # 准备数据
    username_bytes = pad_data(username, USERNAME_LENGTH)
    address_bytes = pad_data(address, ADDR_LENGTH)

    # 生成签名 - 使用DER格式
    private_key = load_key()
    data_to_sign = username_bytes + address_bytes
    signature = sign_data(private_key, data_to_sign)

    data = username_bytes + address_bytes + signature

    # 构造APDU命令
    command = [CLA, INS_DELETE_DATA, 0x00, 0x00, len(data)] + list(data)

    # 发送命令
    try:
        response, sw1, sw2 = connection.transmit(command)

        if (sw1, sw2) == (0x90, 0x00):
            print(f"删除成功: 记录索引={response[0]}, 剩余记录总数={response[1]}")
            return True
        else:
            print(f"删除失败: SW={hex(sw1)[2:].zfill(2)}{hex(sw2)[2:].zfill(2)}")
            return False
    except Exception as e:
        print(f"删除错误: {str(e)}")
        return False


def run_tests():
    """运行完整的测试流程"""
    connection = connect_card()
    if not connection:
        return

    print("开始测试...")

    # 测试数据
    test_username1 = "user1@example.com"
    test_address1 = "0x1234567890abcdef1234567890abcdef1234567890abcdef"
    test_message1 = "这是一条测试消息"

    test_username2 = "user2@example.com"
    test_address2 = "0xabcdef1234567890abcdef1234567890abcdef1234567890"
    test_message2 = "这是另一条测试消息"

    # 测试1: 存储数据
    store_success1 = store_data(
        connection, test_username1, test_address1, test_message1)
    store_success2 = store_data(
        connection, test_username2, test_address2, test_message2)

    # 等待一下确保数据写入成功
    time.sleep(0.5)

    # 测试2: 读取数据
    read_message1 = read_data(connection, test_username1, test_address1)
    read_message2 = read_data(connection, test_username2, test_address2)

    # 测试3: 删除数据
    delete_success1 = delete_data(connection, test_username1, test_address1)

    # 测试4: 验证删除后无法读取
    after_delete = read_data(connection, test_username1, test_address1)

    # 测试5: 验证其他数据仍然可读
    remaining_data = read_data(connection, test_username2, test_address2)

    print("\n测试执行完毕!")


# 当作为脚本直接执行时运行测试
if __name__ == "__main__":
    run_tests()

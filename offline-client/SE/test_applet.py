#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
JavaCard安全芯片测试脚本
用于测试存储和检索消息功能
"""

import sys
import os
from smartcard.System import readers
from smartcard.util import toHexString, toBytes

# 应用AID (Applet Identifier)
AID = [0x65, 0x66, 0x67, 0x68, 0x69, 0x01]

# 指令常量
CLA = 0x80  # 命令类
INS_STORE_INIT = 0x10  # 存储数据初始化命令
INS_STORE_CONTINUE = 0x11  # 存储数据继续命令
INS_STORE_FINALIZE = 0x12  # 存储数据完成命令
INS_READ_INIT = 0x20  # 读取数据初始化命令
INS_READ_CONTINUE = 0x21  # 读取数据继续命令
INS_READ_FINALIZE = 0x22  # 读取数据完成命令

# 状态码
SW_SUCCESS = [0x90, 0x00]  # 操作成功
SW_MORE_DATA_PREFIX = 0x61 # 还有更多数据的前缀

# 测试数据
USER_DATA = [
    {
        "username": "User1",
        "address": "Beijing Haidian",
        "message": "This is the first test message to verify the storage and retrieval functionality."
    },
    {
        "username": "User2",
        "address": "Shanghai Pudong",
        "message": "This is the second test message with a different length to ensure the system handles varying message sizes."
    }
]

def select_reader():
    """选择智能卡读卡器"""
    r = readers()
    if len(r) == 0:
        print("错误：没有找到读卡器!")
        sys.exit(1)
    
    print("可用读卡器:")
    for i, reader in enumerate(r):
        print(f"  {i}: {reader}")
    
    selected = 0
    if len(r) > 1:
        try:
            selected = int(input("请选择读卡器编号 [0]: ") or "0")
            if selected < 0 or selected >= len(r):
                print(f"无效选择，使用默认读卡器 {r[0]}")
                selected = 0
        except ValueError:
            print(f"无效输入，使用默认读卡器 {r[0]}")
            selected = 0
    
    return r[selected]


def connect_card(reader):
    """连接到读卡器中的智能卡"""
    try:
        connection = reader.createConnection()
        connection.connect()
        print(f"已连接到智能卡: {reader}")
        return connection
    except Exception as e:
        print(f"连接卡片失败: {e}")
        sys.exit(1)


def select_applet(connection):
    """选择JavaCard应用"""
    SELECT = [0x00, 0xA4, 0x04, 0x00, len(AID)] + AID
    response, sw1, sw2 = connection.transmit(SELECT)
    if [sw1, sw2] == SW_SUCCESS:
        print(f"已选择应用 AID: {toHexString(AID)}")
        return True
    else:
        print(f"选择应用失败，错误码: {sw1:02X} {sw2:02X}")
        return False


def string_to_bytes(s):
    """将字符串转换为字节列表"""
    return list(s.encode('utf-8'))


def bytes_to_string(byte_list):
    """将字节列表转换为字符串"""
    return bytes(byte_list).decode('utf-8')


def store_data(connection, username, address, message):
    """存储数据到安全芯片"""
    print(f"\n存储数据: 用户名='{username}', 地址='{address}'")
    
    # 1. 发送存储初始化命令
    username_bytes = string_to_bytes(username)
    address_bytes = string_to_bytes(address)
    
    # 数据格式: [userNameLength(1)][userName(var)][addrLength(1)][addr(var)]
    data = [len(username_bytes)] + username_bytes + [len(address_bytes)] + address_bytes
    command = [CLA, INS_STORE_INIT, 0x00, 0x00, len(data)] + data
    
    print("正在发送存储初始化命令...")
    response, sw1, sw2 = connection.transmit(command)
    if [sw1, sw2] != SW_SUCCESS:
        print(f"存储初始化失败，错误码: {sw1:02X} {sw2:02X}")
        return False
    
    # 2. 发送消息数据（如果消息较长，可能需要分段发送）
    message_bytes = string_to_bytes(message)
    CHUNK_SIZE = 200  # 每次发送的最大字节数
    offset = 0
    
    print(f"正在传输消息数据 ({len(message_bytes)} 字节)...")
    
    while offset < len(message_bytes):
        chunk = message_bytes[offset:offset + CHUNK_SIZE]
        command = [CLA, INS_STORE_CONTINUE, 0x00, 0x00, len(chunk)] + chunk
        response, sw1, sw2 = connection.transmit(command)
        
        if [sw1, sw2] != SW_SUCCESS:
            print(f"发送数据块失败，错误码: {sw1:02X} {sw2:02X}")
            return False
        
        offset += CHUNK_SIZE
        print(f"已发送 {min(offset, len(message_bytes))}/{len(message_bytes)} 字节")
    
    # 3. 完成存储操作
    command = [CLA, INS_STORE_FINALIZE, 0x00, 0x00, 0x00]
    response, sw1, sw2 = connection.transmit(command)
    
    if [sw1, sw2] == SW_SUCCESS:
        print("存储完成！")
        return True
    else:
        print(f"存储完成命令失败，错误码: {sw1:02X} {sw2:02X}")
        return False


def read_data(connection, username, address):
    """从安全芯片读取数据"""
    print(f"\n读取数据: 用户名='{username}', 地址='{address}'")
    
    # 1. 发送读取初始化命令
    username_bytes = string_to_bytes(username)
    address_bytes = string_to_bytes(address)
    
    # 数据格式: [userNameLength(1)][userName(var)][addrLength(1)][addr(var)][signatureLength(1)][signaturePlaceholder(var)]
    # 签名占位符长度设置为0，不发送实际数据
    signature_placeholder_length = 0
    data = [len(username_bytes)] + username_bytes + [len(address_bytes)] + address_bytes + [signature_placeholder_length]
    
    # 添加Le字段，预期返回2字节的消息长度
    command = [CLA, INS_READ_INIT, 0x00, 0x00, len(data)] + data + [0x02]
    
    print("正在发送读取初始化命令...")
    response, sw1, sw2 = connection.transmit(command)
    
    if [sw1, sw2] != SW_SUCCESS:
        print(f"读取初始化失败，错误码: {sw1:02X} {sw2:02X}")
        return None
    
    # 解析消息总长度（响应中的前2个字节）
    if len(response) >= 2:
        message_length = (response[0] << 8) | response[1]
        print(f"消息总长度: {message_length} 字节")
    else:
        print("无法获取消息长度")
        return None
    
    # 2. 读取消息数据
    message_bytes = []
    more_data = True
    
    while more_data:
        # 检查是否所有数据都已读取
        if len(message_bytes) >= message_length:
            more_data = False
            break

        # 设置 Le 为期望读取的最大长度 (例如 0xF0 或剩余长度)
        expected_len = min(0xF0, message_length - len(message_bytes))
        if expected_len <= 0: # 如果计算出的期望长度为0或负数，说明已读完
             more_data = False
             break
        command = [CLA, INS_READ_CONTINUE, 0x00, 0x00, expected_len]
        response, sw1, sw2 = connection.transmit(command)
        
        # 检查状态码
        if sw1 == SW_MORE_DATA_PREFIX:  # 还有更多数据 (0x61xx)
            message_bytes.extend(response)
            remaining_bytes = sw2 # 剩余字节数 (可能不准确，如果剩余超过255)
            print(f"已读取 {len(message_bytes)}/{message_length} 字节，状态码提示剩余 {remaining_bytes} 字节，继续读取...")
        elif [sw1, sw2] == SW_SUCCESS:
            message_bytes.extend(response)
            more_data = False
            print(f"已读取 {len(message_bytes)}/{message_length} 字节，数据接收完毕")
        else:
            print(f"读取数据失败，错误码: {sw1:02X} {sw2:02X}")
            return None
    
    # 3. 完成读取操作（可选，Applet在发送完最后一块数据后会自动重置状态）
    # command = [CLA, INS_READ_FINALIZE, 0x00, 0x00, 0x00]
    # connection.transmit(command) 
    
    # 将字节数据转换为字符串
    try:
        message = bytes_to_string(message_bytes)
        return message
    except Exception as e:
        print(f"解码消息失败: {e}")
        return None


def run_test():
    """运行测试流程"""
    
    # 1. 连接到读卡器
    reader = select_reader()
    connection = connect_card(reader)
    
    # 2. 选择JavaCard应用
    if not select_applet(connection):
        sys.exit(1)
    
    # 3. 存储两条测试消息
    for i, user_data in enumerate(USER_DATA):
        if not store_data(connection, user_data["username"], user_data["address"], user_data["message"]):
            print(f"存储消息 {i+1} 失败，终止测试")
            sys.exit(1)
    
    # 4. 读取两条测试消息 
    for i, user_data in enumerate(USER_DATA):
        message = read_data(connection, user_data["username"], user_data["address"])
        if message:
            print(f"\n成功读取消息 {i+1}:")
            print(f"原始消息: {user_data['message']}")
            print(f"读取消息: {message}")
            
            if message == user_data["message"]:
                print("验证结果: 消息内容匹配 ✓")
            else:
                print("验证结果: 消息内容不匹配 ✗")
        else:
            print(f"读取消息 {i+1} 失败")
    
    connection.disconnect()
    print("\n测试完成！")


if __name__ == "__main__":
    try:
        run_test()
    except KeyboardInterrupt:
        print("\n测试被用户中断")
    except Exception as e:
        print(f"测试出现异常: {e}")

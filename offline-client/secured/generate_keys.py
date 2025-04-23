#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
生成ECDSA密钥对，用于JavaCard安全芯片应用和测试脚本
"""

import os
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives import serialization
import base64
import binascii
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import utils

def generate_ec_key_pair():
    """生成ECDSA密钥对 (使用P-256曲线)"""
    # 使用NIST P-256曲线 (secp256r1) - 支持良好且安全性足够
    private_key = ec.generate_private_key(ec.SECP256R1())
    public_key = private_key.public_key()
    
    return private_key, public_key

def save_private_key(private_key, filename="ec_private_key.pem"):
    """将私钥保存为PEM格式文件"""
    pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption()
    )
    
    with open(filename, 'wb') as f:
        f.write(pem)
    
    print(f"私钥已保存至: {filename}")

def get_public_key_bytes(public_key):
    """获取公钥的未压缩字节表示 (04 || x || y)"""
    # 返回未压缩格式: 04 || x || y 
    # (04是前缀表示未压缩, 之后是x和y坐标)
    return public_key.public_bytes(
        encoding=serialization.Encoding.X962,
        format=serialization.PublicFormat.UncompressedPoint
    )

def format_for_javacard(key_bytes):
    """格式化字节序列为JavaCard代码中的数组初始化格式"""
    hex_array = []
    for b in key_bytes:
        hex_array.append(f"(byte)0x{b:02X}")
    
    # 按每行16个元素格式化
    formatted_lines = []
    for i in range(0, len(hex_array), 8):
        formatted_lines.append("        " + ", ".join(hex_array[i:i+8]))
    
    return "{\n" + ",\n".join(formatted_lines) + "\n    };"

def sign_and_verify(private_key, public_key):
    """使用私钥签名消息并使用公钥验证签名"""
    message = "这是一个测试消息".encode("utf-8")
    print("\n原始消息:", message.decode())

    # 使用私钥签名消息
    signature = private_key.sign(
        message,
        ec.ECDSA(hashes.SHA256())
    )
    print("签名:", binascii.hexlify(signature).decode())

    # 使用公钥验证签名
    try:
        public_key.verify(
            signature,
            message,
            ec.ECDSA(hashes.SHA256())
        )
        print("签名验证成功!")
    except Exception as e:
        print("签名验证失败:", str(e))

def load_private_key(filename="ec_private_key.pem"):
    """从PEM文件中加载私钥"""
    with open(filename, 'rb') as f:
        private_key = serialization.load_pem_private_key(
            f.read(),
            password=None
        )
    print(f"私钥已从文件 {filename} 加载")
    return private_key

def load_public_key(filename="ec_public_key.bin"):
    """从二进制文件中加载公钥"""
    with open(filename, 'rb') as f:
        public_key_bytes = f.read()
    
    # 从字节中加载公钥
    public_key = ec.EllipticCurvePublicKey.from_encoded_point(
        ec.SECP256R1(), public_key_bytes
    )
    print(f"公钥已从文件 {filename} 加载")
    return public_key

def main():
    # 生成密钥对
    print("正在生成ECDSA密钥对...")
    private_key, public_key = generate_ec_key_pair()
    
    # 保存私钥
    save_private_key(private_key)
    
    # 获取公钥字节
    public_key_bytes = get_public_key_bytes(public_key)
    
    # 获取JavaCard格式的公钥
    javacard_public_key = format_for_javacard(public_key_bytes)
    
    # 输出公钥信息
    print("\n公钥信息 (NIST P-256 / secp256r1):")
    print(f"长度: {len(public_key_bytes)} 字节")
    print(f"十六进制: {binascii.hexlify(public_key_bytes).decode()}")
    
    # 输出JavaCard格式的公钥
    print("\nJavaCard格式的公钥 (用于复制到SecurityChipApplet.java):")
    print("private static final byte[] EC_PUBLIC_KEY_BYTES = " + javacard_public_key)
    
    # 保存公钥字节到文件，供测试脚本使用
    with open("ec_public_key.bin", "wb") as f:
        f.write(public_key_bytes)
    print("\n公钥字节已保存至: ec_public_key.bin")
    
    # 从文件加载私钥和公钥
    loaded_private_key = load_private_key()
    loaded_public_key = load_public_key()
    
    # 签名和验证测试
    sign_and_verify(loaded_private_key, loaded_public_key)

    # 输出测试脚本使用说明
    print("\n使用说明:")
    print("1. 将JavaCard格式的公钥复制到SecurityChipApplet.java中")
    print("2. 使用ec_private_key.pem文件进行测试脚本签名操作")
    print("3. 重新编译并安装JavaCard应用")
    print("4. 运行修改后的测试脚本")

if __name__ == "__main__":
    main()
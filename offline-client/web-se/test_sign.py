#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import requests
import json
import base64
import os
import random
import string
from typing import Dict, Any
from hashlib import sha256
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.asymmetric.utils import encode_dss_signature

# 服务器配置
SERVER_URL = "http://localhost:8080"
API_ENDPOINT = f"{SERVER_URL}/api/v1/mpc/sign"

# 参与方数量
PARTIES = 3
USERNAME = "test_user"
FILENAME_TEMPLATE = "keygen_test_{}.json"
RESULT_TEMPLATE = "keygen_result_{}.json"
PRIVATE_KEY_DIR = "private_keys"
PRIVATE_KEY_TEMPLATE = os.path.join(PRIVATE_KEY_DIR, "ec_private_key.pem")

# 构造一个假的待签名数据（32字节，hex字符串）


def fake_data() -> str:
    return ''.join(random.choices('0123456789abcdef', k=64))


# 加载PEM私钥
def load_private_key(index: int):
    key_path = PRIVATE_KEY_TEMPLATE.format(index)
    if not os.path.exists(key_path):
        raise FileNotFoundError(f"未找到私钥文件: {key_path}")
    with open(key_path, "rb") as f:
        key_data = f.read()
    private_key = serialization.load_pem_private_key(key_data, password=None)
    return private_key


# userName+Addr 进行签名，DER格式后base64编码
def sign_username_addr(username: str, addr: str, index: int) -> str:
    private_key = load_private_key(index)
    # username 用 SHA-256 哈希，得到 32 字节
    username_bytes = sha256(username.encode("utf-8")).digest()
    # 地址去掉0x前缀，转成20字节
    clean_addr = addr[2:] if addr.startswith("0x") else addr
    addr_bytes = bytes.fromhex(clean_addr)
    if len(addr_bytes) != 20:
        raise ValueError("地址长度不是20字节")
    data = username_bytes + addr_bytes
    digest = sha256(data).digest()
    signature = private_key.sign(
        digest,
        ec.ECDSA(hashes.SHA256())
    )
    # cryptography默认输出ASN.1 DER格式
    return base64.b64encode(signature).decode()


def test_sign(parties: str, data: str, index: int, username: str) -> Dict[str, Any]:
    """
    测试签名服务
    Args:
        parties: 参与方字符串，如 "1,2,3"
        data: 待签名数据（hex字符串）
        index: 当前参与方序号
        username: 用户名
    Returns:
        响应数据
    """
    # 读取密钥生成结果
    result_file = RESULT_TEMPLATE.format(index)
    if not os.path.exists(result_file):
        print(f"未找到密钥文件: {result_file}")
        return None
    with open(result_file, "r") as f:
        keygen_result = json.load(f)
    address = keygen_result["address"]
    encrypted_key = keygen_result["encryptedKey"]

    # 构造请求数据
    payload = {
        "parties": parties,
        "data": data,
        "filename": FILENAME_TEMPLATE.format(index),
        "encryptedKey": encrypted_key,
        "userName": username,
        "address": address,
        "signature": sign_username_addr(username, address, index)
    }

    print(f"\n发起签名请求: 参与方{index}")
    print(json.dumps(payload, indent=2, ensure_ascii=False))

    try:
        response = requests.post(API_ENDPOINT, json=payload)
        response.raise_for_status()
        result = response.json()
        if result.get("success"):
            print("签名成功! 签名结果:")
            print(result.get("signature"))
        else:
            print("签名失败:", result.get("message"))
        return result
    except requests.exceptions.RequestException as e:
        print(f"请求失败: {str(e)}")
        if hasattr(e, 'response') and e.response is not None:
            print(f"错误响应: {e.response.text}")
        return None


def main():
    print("开始测试签名服务...")
    parties_str = ','.join(str(i) for i in range(1, PARTIES + 1))
    data = fake_data()
    results = []
    for i in range(1, PARTIES + 1):
        result = test_sign(parties_str, data, i, USERNAME)
        results.append(result)
    print("\n=== 签名测试完成 ===")
    print(
        f"成功签名的参与方数量: {sum(1 for r in results if r and r.get('success'))}/{PARTIES}")


if __name__ == "__main__":
    main()

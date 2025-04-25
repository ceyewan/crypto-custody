#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from multiprocessing import Process, Manager
import requests
import json
import base64
import os
import time
from typing import Dict, Any

# 服务器配置
SERVER_URL = "http://localhost:8080"
API_ENDPOINT = f"{SERVER_URL}/api/v1/mpc/keygen"


def test_keygen(threshold: int, parties: int, index: int, username: str) -> Dict[str, Any]:
    """
    测试密钥生成服务

    Args:
        threshold: 门限值
        parties: 参与方总数
        index: 当前参与方序号
        username: 用户名

    Returns:
        响应数据
    """
    # 构建请求数据
    payload = {
        "threshold": threshold,
        "parties": parties,
        "index": index,
        "filename": f"keygen_test_{index}.json",
        "userName": username
    }

    print(f"\n发起密钥生成请求:")
    print(f"门限值: {threshold}")
    print(f"参与方总数: {parties}")
    print(f"当前参与方序号: {index}")
    print(f"用户名: {username}")

    try:
        # 发送请求
        response = requests.post(API_ENDPOINT, json=payload)
        response.raise_for_status()

        # 解析响应
        result = response.json()

        if result["success"]:
            print("\n密钥生成成功!")
            print(f"地址: {result['address']}")
            print(f"加密密钥长度: {len(result['encryptedKey'])} 字节")

            # 保存结果到文件
            output_file = f"keygen_result_{index}.json"
            with open(output_file, "w") as f:
                json.dump(result, f, indent=2)
            print(f"结果已保存到: {output_file}")

        return result

    except requests.exceptions.RequestException as e:
        print(f"\n请求失败: {str(e)}")
        if hasattr(e, 'response') and e.response is not None:
            print(f"错误响应: {e.response.text}")
        return None


def run_keygen(threshold, parties, index, username, results):
    result = test_keygen(threshold, parties, index, username)
    if result:
        results.append(result)


def main():
    THRESHOLD = 2
    PARTIES = 3
    USERNAME = "test_user"

    print("开始测试密钥生成服务...")
    print(f"服务器地址: {SERVER_URL}")

    manager = Manager()
    results = manager.list()
    processes = []

    for i in range(1, PARTIES + 1):
        print(f"\n=== 启动参与方 {i} 进程 ===")
        p = Process(target=test_keygen, args=(THRESHOLD, PARTIES, i, USERNAME))
        p.start()
        processes.append(p)
        time.sleep(1)  # 等待1秒再发起下一个

    # 等待所有进程完成
    for p in processes:
        p.join()

    # 打印汇总信息
    print("\n=== 测试完成 ===")
    print(f"成功生成密钥的参与方数量: {len(results)}/{PARTIES}")
    if results:
        print("\n所有生成的地址:")
        for i, result in enumerate(results, 1):
            print(f"参与方 {i}: {result['address']}")


if __name__ == "__main__":
    main()

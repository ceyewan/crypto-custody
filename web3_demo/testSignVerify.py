from eth_utils import to_bytes
from eth_keys.exceptions import BadSignature
from eth_keys import keys
from ecdsa import SigningKey, VerifyingKey, SECP256k1, util
from ecdsa.ellipticcurve import Point
import binascii


def create_key_pair():
    """生成公私钥对"""
    # 生成 secp256k1 私钥
    private_key = SigningKey.generate(curve=SECP256k1)
    # 从私钥生成公钥
    public_key = private_key.get_verifying_key()
    # 将私钥和公钥以十六进制字符串追加写入 account.txt 文件
    with open("account.txt", "a") as f:
        f.write("私钥: " + private_key.to_string().hex() + "\n")
        f.write("公钥X: " + public_key.pubkey.point.x().to_bytes(32, "big").hex() + "\n")
        f.write("公钥Y: " + public_key.pubkey.point.y().to_bytes(32, "big").hex() + "\n")


# 对消息哈希进行签名
def sign_message(digest_hex, private_key_hex):
    """对消息哈希进行签名"""
    private_key = SigningKey.from_string(
        bytes.fromhex(private_key_hex), curve=SECP256k1)
    signature = private_key.sign_digest(to_bytes(hexstr=digest_hex))
    # 将签名拆分为 r 和 s 并写入 account.txt 文件
    r, s = util.sigdecode_string(signature, SECP256k1.order)
    with open("account.txt", "a") as f:
        f.write("签名R: " + r.to_bytes(32, "big").hex() + "\n")
        f.write("签名S: " + s.to_bytes(32, "big").hex() + "\n")


if __name__ == '__main__':
    # 用户选择执行什么操作
    print("1. 生成公私钥对")
    print("2. 对消息哈希进行签名")
    choice = input("请选择操作: ")
    if choice == "1":
        create_key_pair()
    elif choice == "2":
        # 用户输入私钥和消息哈希
        private_key_hex = input("请输入私钥: ")
        digest_hex = input("请输入消息哈希: ")
        sign_message(digest_hex, private_key_hex)
    else:
        print("输入有误")
        exit(1)

from smartcard.System import readers
from smartcard.util import toHexString

# APDU 命令：获取 CPLC 数据
GET_CPLC_APDU = [0x80, 0xCA, 0x9F, 0x7F, 0x00]

def get_cplc():
    r = readers()
    if not r:
        print("未找到任何智能卡读卡器。")
        return

    print(f"可用读卡器：{r}")
    reader = r[0]
    print(f"使用读卡器：{reader}")

    connection = reader.createConnection()
    connection.connect()

    print(f"发送 APDU: {toHexString(GET_CPLC_APDU)}")
    response, sw1, sw2 = connection.transmit(GET_CPLC_APDU)

    if sw1 == 0x90 and sw2 == 0x00:
        cplc = toHexString(response).replace(" ", "")
        print(f"CPLC: {cplc}")
        return cplc
    else:
        print(f"APDU 响应失败: SW1={hex(sw1)}, SW2={hex(sw2)}")
        return None

if __name__ == "__main__":
    get_cplc()
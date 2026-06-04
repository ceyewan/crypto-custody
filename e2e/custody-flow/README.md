# Custody Online/Offline E2E Flow

这个目录用于执行在线系统和离线系统协同的完整冒烟流程，并把每一步交换的 JSON 保存到同一个运行目录。

当前默认流程：

1. 在线系统创建源案件和目标案件。
2. 源案件导出 keygen 任务，离线系统用 `u1/u2/u3` 生成源托管地址。
3. 目标案件导出 keygen 任务，离线系统用 `u1/u2/u3` 生成目标托管地址。
4. Ganache 给源托管地址打 `50 ETH`。
5. 在线系统创建从源托管地址到目标托管地址的 `20 ETH` 交易。
6. 离线系统默认用 `u1/u2` 完成 2-of-3 签名；也可以用 `-signers u1,u3` 或 `-signers u2,u3` 切换组合。
7. 在线系统导入签名并广播。
8. 同步两个在线账户余额，并额外写入链上最终余额。

源地址最终余额会因为 gas 略小于 `30 ETH`；目标地址应接近或等于 `20 ETH`。

默认服务地址：

```text
在线系统 HTTP:  http://127.0.0.1:8088
离线系统 HTTP:  http://127.0.0.1:8080
离线系统 WS:    ws://127.0.0.1:8081/ws
Ganache RPC:    http://127.0.0.1:8545
```

默认用户：

```text
在线 admin: admin / admin123
离线 admin: admin / admin123
离线参与方: u1/u2/u3 / officer123
默认签名方: u1/u2 / officer123
```

运行：

```bash
e2e/custody-flow/run.sh
```

如果离线库里有旧任务导致 `任务编号已存在但payload_hash不同`，可以先清理离线 DB。脚本会先备份当前 DB，再停止离线容器、删除 DB/WAL/SHM 文件并重新启动容器：

```bash
e2e/custody-flow/reset-offline-db.sh
e2e/custody-flow/run.sh
```

常用参数：

```bash
e2e/custody-flow/run.sh \
  -online-url http://127.0.0.1:8088 \
  -offline-url http://127.0.0.1:8080 \
  -offline-ws ws://127.0.0.1:8081/ws \
  -ganache-rpc http://127.0.0.1:8545 \
  -offline-db offline-server-handoff/data/crypto-custody.db \
  -signers u1,u2 \
  -fund-amount 50 \
  -tx-value "20 ETH"
```

默认会开启 `-isolate-se`，自动把离线 SQLite 中其他 active 安全芯片记录标记为 disabled，使本机插入的芯片成为本次 E2E 的唯一 active SE。关闭时可传 `-isolate-se=false`。

如果需要指定读卡器名称：

```bash
e2e/custody-flow/run.sh -reader "GOODIX GSE SmartCard Reader 01"
```

输出目录默认是：

```text
e2e/custody-flow/runs/<timestamp>/
```

其中会包含：

```text
01_online_login.json
02_offline_login_admin.json
03_offline_login_u1.json
04_offline_login_u2.json
05_offline_login_u3.json
06_offline_se_list.json
06_offline_se_create.json
06_offline_se_isolate.json
07_source_case_create.json
07_source_keygen_task_online.json
07_source_keygen_task_offline_import.json
07_source_keygen_complete_ws.json
07_source_keygen_result_offline.json
07_source_wallet_import_online.json
13_target_case_create.json
13_target_keygen_task_online.json
13_target_keygen_task_offline_import.json
13_target_keygen_complete_ws.json
13_target_keygen_result_offline.json
13_target_wallet_import_online.json
19_ganache_fund_source.json
20_source_balance_after_fund.json
21_transaction_create.json
22_sign_task_online.json
23_sign_task_offline_import.json
24_sign_complete_ws.json
25_sign_result_offline.json
26_signature_verify_offline.json
27_signature_import_online.json
28_broadcast_online.json
29_source_balance_final.json
30_target_balance_final.json
31_chain_balances_final.json
summary.json
```

前置条件：

- 在线系统和离线系统已经在 Docker 中启动。
- Ganache 已启动，且在线系统容器能通过它自己的 `ETH_RPC_URL` 访问 Ganache。
- Ganache RPC 对测试账户开放 `eth_accounts` 和 unlocked `eth_sendTransaction`。
- 本机能访问安全芯片读卡器，并且已安装 Applet。

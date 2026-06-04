# Goodix pygse 工具链说明

本目录放置 Goodix 提供的离线安装包：

- `pygse-2.1.5-py3-none-any.whl`
- `gpqc-1.0.1-py3-none-any.whl`

## 已验证环境

不要直接使用系统默认 `python3`。如果默认 Python 是 3.14 或更新版本，`pygse` 依赖的 `pyscard` 容易出现兼容性问题。

当前已验证组合：

- Python `3.11`
- `pygse 2.1.5`
- `gpqc 1.0.1`
- `pyscard 2.2.2`

macOS 初始化命令：

```bash
# 从仓库根目录执行
cd offline-client/secured/tools
/opt/homebrew/opt/python@3.11/bin/python3.11 -m venv .venv311
source .venv311/bin/activate
python -m pip install -U pip setuptools wheel
python -m pip install -U ./pygse-2.1.5-py3-none-any.whl ./gpqc-1.0.1-py3-none-any.whl
python -m pip install --force-reinstall "pyscard==2.2.2"
pygse ls-dev
```

不激活虚拟环境时，可以直接调用：

```bash
offline-client/secured/tools/.venv311/bin/pygse ls-dev
```

## 常用命令

列出设备：

```bash
offline-client/secured/tools/.venv311/bin/pygse ls-dev
```

查看目标 SE 信息：

固定目标设备名为 `GOODIX GSE SmartCard Reader`。不要使用 `GOODIX GSE SmartCard Reader 01`。

```bash
offline-client/secured/tools/.venv311/bin/pygse info --dev "GOODIX GSE SmartCard Reader"
```

安装 Applet：

```bash
cd offline-client/secured
tools/.venv311/bin/pygse install \
  --dev "GOODIX GSE SmartCard Reader" \
  --app-aid=. \
  build/cap/securitychip.cap \
  --log-level info
```

注意：安装同一个 AID 会删除旧 Applet 并清空旧记录。对生产 SE 操作前，需要确认目标读卡器名称、CPLC 和是否允许清空数据。

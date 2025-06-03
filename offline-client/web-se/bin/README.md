# 跨平台二进制文件目录结构

本目录包含不同操作系统的可执行文件，程序会根据运行环境自动选择对应的二进制文件。

## 目录结构

```
bin/
├── MacOS/          # macOS 可执行文件
│   ├── gg20_keygen
│   └── gg20_signing
├── Linux/          # Linux 可执行文件
│   ├── gg20_keygen
│   └── gg20_signing
└── Windows/        # Windows 可执行文件
    ├── gg20_keygen.exe
    └── gg20_signing.exe
```

## 使用说明

1. **自动检测**: 程序启动时会自动检测当前操作系统（macOS/Linux/Windows）
2. **路径选择**: 根据操作系统自动选择对应子目录下的可执行文件
3. **Windows支持**: Windows 下会自动添加 `.exe` 后缀

## 添加新平台支持

1. 在 `bin/` 目录下创建对应的操作系统子目录
2. 将该平台的可执行文件放入对应目录
3. 如需要，可在 `utils/command.go` 中的 `getOSBinDir()` 函数添加新的操作系统支持

## 当前支持的操作系统

- **macOS** (darwin) → `bin/MacOS/`
- **Linux** → `bin/Linux/`
- **Windows** → `bin/Windows/`
- **其他** → 默认使用 `bin/Linux/`

## 日志信息

程序在执行命令时会在日志中记录：
- 使用的命令路径
- 当前操作系统
- 执行参数和结果

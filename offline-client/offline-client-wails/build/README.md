# build 目录

`build` 目录保存 Wails 桌面应用构建所需的图标、平台配置和安装器资源。

## 目录结构

- `bin/`：构建输出目录。
- `darwin/`：macOS 构建配置。
- `windows/`：Windows 构建配置和安装器资源。

## macOS

`darwin` 目录保存 macOS 构建使用的 plist 配置，可根据应用名称、权限和打包需求调整。

## 文件说明

- `Info.plist`：正式构建使用的 macOS plist。
- `Info.dev.plist`：开发模式使用的 macOS plist。

## Windows

`windows` 目录保存 Windows 构建、应用图标和安装器配置。

- `icon.ico`：Windows 应用图标。
- `installer/`：Windows 安装器资源。
- `info.json`：Windows 应用元信息。
- `wails.exe.manifest`：Windows 应用清单。

## 维护建议

- 修改应用名称、图标或平台权限后，同步检查对应平台配置。
- 构建产物不要手动混入配置目录，输出文件应放在 `bin/`。

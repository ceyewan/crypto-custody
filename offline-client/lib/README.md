# lib 目录

`lib` 目录保存 JavaCard Applet 构建所需的本地依赖，供 `secured` 和 `unsecured` 两个 Applet 使用。

## 文件说明

- `jc305u4_kit/`：JavaCard SDK、工具、导出文件和平台相关脚本。
- `ant-javacard.jar`：Ant 构建 JavaCard Applet 时使用的扩展包。

## 使用场景

`secured/build.xml` 和 `unsecured/build.xml` 会引用本目录中的依赖来编译并生成 `.cap` 文件。

## 维护建议

- 调整 SDK 版本后，需要同时验证 `secured` 和 `unsecured` 的构建流程。
- 构建脚本中的相对路径依赖本目录与 Applet 目录处于同一父目录下。
- 不要把运行时生成的临时文件放入本目录。

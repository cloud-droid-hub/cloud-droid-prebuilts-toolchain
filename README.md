# cloud-droid-prebuilts-toolchain

本私有仓库为`cloud-droid-hub`提供本地与Docker构建所需的固定宿主原生工具链。仓库只维护`main`分支，三个原平台分支的内容和历史分别归入以下顶层目录：

```text
darwin-arm64/
linux-arm64/
linux-amd64/
```

每个目录包含自己的安装程序、类型化归档目录、分片、许可证、测试与版本。平台目录固定发布Go 1.26.8、LLVM 19.1.7、LLVM 22、Temurin JDK 21.0.12.1+1、Android NDK r29 revision 29.0.14206865和Bazel 8.5.1。macOS LLVM 22为22.1.8，Linux LLVM 22为Ubuntu 26.04的22.1.2；NDK内Clang与LLD主版本为21。

归档以不超过90 MiB的普通Git对象保存，不使用Git LFS。`<host>/bin/cdh-toolchain`负责摘要校验、安全展开、宿主架构检查和原子发布。主项目把本仓放在`tools/toolchains/prebuilts/`，再把当前宿主工具展开到`output/toolchains/<host>/`；缺失、版本错误或架构不符都会直接失败，不会改用系统工具。

## 安装当前平台

从仓库根目录指定宿主目录：

```sh
host=darwin-arm64
repo="$PWD/$host"
dest="$PWD/output/$host"
"$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest"
"$repo/bin/cdh-toolchain" check --dest "$dest"
```

Linux可把`host`改为`linux-arm64`或`linux-amd64`。Linux安装程序还提供`select`，用于在完整检查后返回明确工具路径：

```sh
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-clang
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-lld
"$repo/bin/cdh-toolchain" select --dest "$dest" --name ndk
```

Linux arm64的NDK只提供Android arm64目标输入。该平台使用同目录LLVM 22、NDK sysroot和Clang 21 resource目录，不执行Linux amd64 NDK程序。

各平台的详细版本、限制与手工验证命令见对应目录的`README.md`。迁移设计和验证记录见`docs/plans/2026-09-12-预编译工具链单一主分支设计及实施计划.md`。

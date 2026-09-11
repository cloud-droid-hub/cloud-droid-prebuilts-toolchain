# cloud-droid-prebuilts-toolchain

本私有仓库为`cloud-droid-hub`提供本地与Docker构建所需的宿主原生工具链，避免要求开发者通过Homebrew或APT安装指定编译器、Go与JDK。

长期分支只有：

- `darwin-arm64`
- `linux-arm64`
- `linux-amd64`

每个分支最终发布Go 1.26.8、LLVM 22、LLVM 19.1.7、Temurin JDK 21.0.12.1+1和Android NDK r29 revision 29.0.14206865。NDK只保留Android arm64目标输入；Linux arm64使用宿主原生LLVM 22与NDK sysroot，不执行Linux x86_64 NDK程序。

工具链归档以不超过90 MiB的普通Git对象保存。`bin/cdh-toolchain`负责摘要校验、安全展开、宿主架构检查和原子发布。主项目只初始化当前宿主分支，并把展开根放在主仓`output/toolchains/<host>/`；Docker构建从对应Linux分支复制同一份内容。

`cdh-toolchain select`在重新检查完整安装后输出一个明确路径，不修改当前进程环境。可选名称包括`go`、`jdk`、`llvm19`、`llvm22`、`llvm22-clang`、`llvm22-clang++`、`llvm22-lld`、`llvm22-ar`、`ndk`、`ndk-sysroot`与`ndk-resource`；未在当前分支发布的名称会明确失败。

首个提交只登记已批准设计、约束与实施顺序。各工具链会按独立可验证提交逐项发布，完成状态以各分支实际文件、版本和验证结果为准。

## 当前分支状态

`linux-arm64`已发布Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2和Android NDK r29目标输入。当前目录只会安装已登记且已完成摘要、版本与ELF架构验证的内容，不会切换到系统程序。

NDK只提供Android arm64目标输入，并由同分支LLVM 22执行编译与链接。它不含Google发布包里的Linux x86_64程序，因此不能把NDK根目录直接交给仍要求`ndk-build`的旧入口。

在本仓根执行：

```sh
./bin/cdh-toolchain install --repo "$PWD" --dest "$PWD/output/linux-arm64"
./bin/cdh-toolchain check --dest "$PWD/output/linux-arm64"
./bin/cdh-toolchain select --dest "$PWD/output/linux-arm64" --name llvm22-clang
./bin/cdh-toolchain select --dest "$PWD/output/linux-arm64" --name llvm22-lld
./bin/cdh-toolchain select --dest "$PWD/output/linux-arm64" --name ndk-sysroot
./bin/cdh-toolchain select --dest "$PWD/output/linux-arm64" --name ndk-resource
"$PWD/output/linux-arm64/go126/bin/go" version
"$PWD/output/linux-arm64/jdk21/bin/java" -version
"$PWD/output/linux-arm64/llvm19/bin/clang" --version
"$PWD/output/linux-arm64/llvm19/bin/ld.lld" --version
"$PWD/output/linux-arm64/llvm19/bin/llvm-ar" --version
"$PWD/output/linux-arm64/llvm22/bin/clang" --version
"$PWD/output/linux-arm64/llvm22/bin/ld.lld" --version
"$PWD/output/linux-arm64/llvm22/bin/llvm-ar" --version
```

版本输出应分别包含`go version go1.26.8 linux/arm64`、`Temurin-21.0.12.1+1`、LLVM 19.1.7以及Ubuntu LLVM 22.1.2；安装检查还会核对NDK revision `29.0.14206865`。Android编译必须同时使用`select`返回的LLVM 22、NDK sysroot和Clang 21 resource目录，并传入`--target=aarch64-linux-android<API>`、`--rtlib=compiler-rt`及`--ld-path=<selected-ld.lld>`。主仓本地入口与Docker入口会在三个宿主分支发布完成后，于计划Task 5和Task 6接入；接入前不得把本仓探针通过表述为旧入口已经兼容。

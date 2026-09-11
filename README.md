# cloud-droid-prebuilts-toolchain

本私有仓库为`cloud-droid-hub`提供本地与Docker构建所需的宿主原生工具链，避免要求开发者通过Homebrew或APT安装指定编译器、Go与JDK。

长期分支只有：

- `darwin-arm64`
- `linux-arm64`
- `linux-amd64`

每个分支最终发布Go 1.26.8、LLVM 22、LLVM 19.1.7、Temurin JDK 21.0.12.1+1和Android NDK r29 revision 29.0.14206865。NDK只保留Android arm64目标输入；Linux arm64使用宿主原生LLVM 22与NDK sysroot，不执行Linux x86_64 NDK程序。

工具链归档以不超过90 MiB的普通Git对象保存。`bin/cdh-toolchain`负责摘要校验、安全展开、宿主架构检查和原子发布。主项目只初始化当前宿主分支，并把展开根放在主仓`output/toolchains/<host>/`；Docker构建从对应Linux分支复制同一份内容。

`cdh-toolchain select`在重新检查完整安装后输出一个明确路径，不修改当前进程环境。未在当前分支发布的名称会明确失败。

首个提交只登记已批准设计、约束与实施顺序。各工具链会按独立可验证提交逐项发布，完成状态以各分支实际文件、版本和验证结果为准。

## 当前分支状态

`linux-amd64`已发布Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2和Android NDK r29 revision 29.0.14206865。当前目录只会安装已登记且已完成摘要、版本与ELF架构验证的工具，不会切换到系统程序。

本分支按Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2和裁剪后的NDK r29顺序发布。NDK只保留Android arm64目标输入及Linux amd64宿主必需的Clang/LLD 21程序。

在本仓根执行：

```sh
./bin/cdh-toolchain install --repo "$PWD" --dest "$PWD/output/linux-amd64"
./bin/cdh-toolchain check --dest "$PWD/output/linux-amd64"
"$PWD/output/linux-amd64/go126/bin/go" version
"$PWD/output/linux-amd64/jdk21/bin/java" -version
"$PWD/output/linux-amd64/llvm19/bin/clang" --version
"$PWD/output/linux-amd64/llvm19/bin/ld.lld" --version
"$PWD/output/linux-amd64/llvm19/bin/llvm-ar" --version
"$PWD/output/linux-amd64/llvm22/bin/clang" --version
"$PWD/output/linux-amd64/llvm22/bin/ld.lld" --version
"$PWD/output/linux-amd64/llvm22/bin/llvm-ar" --version
"$PWD/output/linux-amd64/ndk29/toolchains/llvm/prebuilt/linux-x86_64/bin/clang" --version
"$PWD/output/linux-amd64/ndk29/toolchains/llvm/prebuilt/linux-x86_64/bin/ld.lld" --version
"$PWD/output/linux-amd64/ndk29/toolchains/llvm/prebuilt/linux-x86_64/bin/llvm-ar" --version
"$PWD/output/linux-amd64/ndk29/ndk-build" --version
./bin/cdh-toolchain select --dest "$PWD/output/linux-amd64" --name ndk
```

版本输出应分别包含`go version go1.26.8 linux/amd64`、`Temurin-21.0.12.1+1`、LLVM 19.1.7、Ubuntu LLVM 22.1.2、Android Clang/LLD/LLVM 21.0.0与GNU Make 4.3对应的版本标识。NDK只含Android arm64目标包装程序、sysroot与运行库；其宿主程序均为Linux x86_64 ELF。主仓本地入口与Docker入口由主仓接入任务消费三个已发布分支。

# cloud-droid-prebuilts-toolchain

本私有仓库为`cloud-droid-hub`提供本地与Docker构建所需的宿主原生工具链，避免要求开发者通过Homebrew或APT安装指定编译器、Go与JDK。

长期分支只有：

- `darwin-arm64`
- `linux-arm64`
- `linux-amd64`

每个分支最终发布Go 1.26.8、LLVM 22、LLVM 19.1.7、Temurin JDK 21.0.12.1+1和Android NDK r29 revision 29.0.14206865。NDK只保留Android arm64目标输入；Linux arm64使用宿主原生LLVM 22与NDK sysroot，不执行Linux x86_64 NDK程序。

工具链归档以不超过90 MiB的普通Git对象保存。`bin/cdh-toolchain`负责摘要校验、安全展开、宿主架构检查和原子发布。主项目只初始化当前宿主分支，并把展开根放在主仓`output/toolchains/<host>/`；Docker构建从对应Linux分支复制同一份内容。

首个提交只登记已批准设计、约束与实施顺序。各工具链会按独立可验证提交逐项发布，完成状态以各分支实际文件、版本和验证结果为准。

## 当前分支状态

`darwin-arm64`已发布Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、LLVM 22.1.8和裁剪后的NDK r29。当前目录只会安装已登记且已完成摘要、版本与Mach-O架构验证的工具，不会切换到系统程序。NDK沿用上游`darwin-x86_64`目录名，其中保留的宿主Mach-O均为单架构arm64。

在本仓根执行：

```sh
./bin/cdh-toolchain install --repo "$PWD" --dest "$PWD/output/darwin-arm64"
./bin/cdh-toolchain check --dest "$PWD/output/darwin-arm64"
"$PWD/output/darwin-arm64/go126/bin/go" version
"$PWD/output/darwin-arm64/jdk21/bin/java" -version
"$PWD/output/darwin-arm64/llvm19/bin/clang" --version
"$PWD/output/darwin-arm64/llvm19/bin/ld64.lld" --version
"$PWD/output/darwin-arm64/llvm19/bin/llvm-ar" --version
"$PWD/output/darwin-arm64/llvm22/bin/clang" --version
"$PWD/output/darwin-arm64/llvm22/bin/ld64.lld" --version
"$PWD/output/darwin-arm64/llvm22/bin/llvm-ar" --version
"$PWD/output/darwin-arm64/ndk29/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang" --version
"$PWD/output/darwin-arm64/ndk29/toolchains/llvm/prebuilt/darwin-x86_64/bin/ld.lld" --version
"$PWD/output/darwin-arm64/ndk29/ndk-build" --version
```

版本输出还应分别包含LLVM 19.1.7、LLVM 22.1.8和Android Clang/LLD 21.0.0对应版本；NDK的`source.properties`固定记录revision `29.0.14206865`。主仓本地入口与Docker入口会在三宿主分支内容全部发布后接入。

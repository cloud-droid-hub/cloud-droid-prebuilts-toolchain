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

`linux-arm64`已发布Go 1.26.8，其它四项按计划继续逐项加入。当前目录只会安装已登记且已完成摘要、版本与ELF架构验证的工具，不会切换到系统程序。

本分支将依次发布Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2和裁剪后的NDK r29。NDK只提供Android arm64目标输入，并由同分支LLVM 22执行编译与链接。

在本仓根执行：

```sh
./bin/cdh-toolchain install --repo "$PWD" --dest "$PWD/output/linux-arm64"
./bin/cdh-toolchain check --dest "$PWD/output/linux-arm64"
"$PWD/output/linux-arm64/go126/bin/go" version
```

最后一条当前应输出`go version go1.26.8 linux/arm64`。主仓本地入口与Docker入口会在三宿主分支内容全部发布后接入。

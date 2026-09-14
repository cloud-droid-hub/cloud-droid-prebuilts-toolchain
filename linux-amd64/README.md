# Linux amd64工具链

本目录由仓库唯一`main`分支维护，发布Linux amd64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2、裁剪后的Android NDK r29 revision 29.0.14206865、Bazel 8.5.1和Rust/Cargo 1.97.1。Rust内部LLVM为22.1.6，并包含`aarch64-linux-android`标准库。

LLVM 22归档内的`libedit2`与`libgnutls30t64`固定取自Ubuntu 24.04 LTS，使工具可在glibc 2.39及更新的amd64宿主运行；LLVM程序与其它随附运行库仍来自登记的Ubuntu 26.04软件包。归档同时携带登记的`libssl`与`libcrypto`，不依赖宿主OpenSSL的小版本补齐`libcurl`。

在仓库根执行：

```sh
repo="$PWD/linux-amd64"
dest="$PWD/output/linux-amd64"
"$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest"
"$repo/bin/cdh-toolchain" check --dest "$dest"
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-clang
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-lld
"$repo/bin/cdh-toolchain" select --dest "$dest" --name ndk
"$dest/go126/bin/go" version
"$dest/jdk21/bin/java" -version
"$dest/llvm19/bin/clang" --version
"$dest/llvm22/bin/clang" --version
"$dest/ndk29/toolchains/llvm/prebuilt/linux-x86_64/bin/clang" --version
"$dest/ndk29/toolchains/llvm/prebuilt/linux-x86_64/bin/ld.lld" --version
"$dest/ndk29/ndk-build" --version
"$dest/bazel851/bin/bazel" --version
"$dest/rust197/bin/rustc" -vV
"$dest/rust197/bin/cargo" -V
"$dest/rust197/bin/rustdoc" --version
```

`install`按组件输出机器可读进度，字段固定为`component`、`index`、`total`、`remaining`、`elapsed_seconds`和`state`：

```text
toolchain_progress component="LLVM 22" index=4 total=7 remaining=3 elapsed_seconds=10 state=installing
```

组件展开期间默认每5秒持续输出`state=installing`；可用`CDH_TOOLCHAIN_PROGRESS_INTERVAL=15s`等大于零的Go duration调整。收到`SIGINT`或`SIGTERM`时返回非零，清理未发布的临时组件并保留此前已完成组件。中断后直接重跑原命令，已经完整安装并通过检查的组件显示`state=skipped`且不重读归档。删除一个组件只补一个，删除多个组件只补对应缺项。组件路径存在但检查失败时明确返回非零，不删除或覆盖该目录；目标根中的用户额外文件保持原样。使用者只有在自行备份并明确移走无效组件后，才能让安装器把该路径作为缺项补齐。目标父目录链必须由当前用户或`root`拥有且不可由组或其它用户写入，否则在创建目标组件前失败。

安装检查验证摘要、版本、文件模式和ELF x86-64架构。任何检查失败都会返回非零退出码，不会搜索或改用系统程序。Linux amd64只用于构建，不表示支持运行云机。

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

安装检查验证摘要、版本、文件模式和ELF x86-64架构。任何检查失败都会返回非零退出码，不会搜索或改用系统程序。Linux amd64只用于构建，不表示支持运行云机。

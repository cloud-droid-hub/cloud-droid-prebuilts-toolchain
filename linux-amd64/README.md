# Linux amd64工具链

本目录由仓库唯一`main`分支维护，发布Linux amd64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2和裁剪后的Android NDK r29 revision 29.0.14206865。

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
```

安装检查验证摘要、版本、文件模式和ELF x86-64架构。任何检查失败都会返回非零退出码，不会搜索或改用系统程序。Linux amd64只用于构建，不表示支持运行云机。

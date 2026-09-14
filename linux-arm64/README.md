# Linux arm64工具链

本目录由仓库唯一`main`分支维护，发布Linux arm64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2、Android NDK r29目标输入、Bazel 8.5.1和Rust/Cargo 1.97.1。Rust内部LLVM为22.1.6，并包含`aarch64-linux-android`标准库。

NDK不包含Google发布的Linux amd64宿主程序。Android arm64编译使用同目录LLVM 22、NDK sysroot和Clang 21 resource目录，不得把NDK根交给要求`ndk-build`的旧入口。

在仓库根执行：

```sh
repo="$PWD/linux-arm64"
dest="$PWD/output/linux-arm64"
"$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest"
"$repo/bin/cdh-toolchain" check --dest "$dest"
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-clang
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-lld
"$repo/bin/cdh-toolchain" select --dest "$dest" --name ndk-sysroot
"$repo/bin/cdh-toolchain" select --dest "$dest" --name ndk-resource
"$dest/go126/bin/go" version
"$dest/jdk21/bin/java" -version
"$dest/llvm19/bin/clang" --version
"$dest/llvm22/bin/clang" --version
"$dest/bazel851/bin/bazel" --version
"$dest/rust197/bin/rustc" -vV
"$dest/rust197/bin/cargo" -V
"$dest/rust197/bin/rustdoc" --version
```

Android编译需传入`--target=aarch64-linux-android<API>`、`--rtlib=compiler-rt`及选定的`--ld-path`。安装检查验证摘要、版本、文件模式和ELF架构，失败时不会改用系统程序。

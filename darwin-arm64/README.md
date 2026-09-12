# Darwin arm64工具链

本目录由仓库唯一`main`分支维护，发布macOS arm64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、LLVM 22.1.8和裁剪后的Android NDK r29 revision 29.0.14206865。

在仓库根执行：

```sh
repo="$PWD/darwin-arm64"
dest="$PWD/output/darwin-arm64"
"$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest"
"$repo/bin/cdh-toolchain" check --dest "$dest"
"$dest/go126/bin/go" version
"$dest/jdk21/bin/java" -version
"$dest/llvm19/bin/clang" --version
"$dest/llvm19/bin/ld64.lld" --version
"$dest/llvm22/bin/clang" --version
"$dest/llvm22/bin/ld64.lld" --version
"$dest/ndk29/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang" --version
"$dest/ndk29/toolchains/llvm/prebuilt/darwin-x86_64/bin/ld.lld" --version
"$dest/ndk29/ndk-build" --version
```

安装检查验证摘要、版本、文件模式和Mach-O架构。任何检查失败都会返回非零退出码，不会搜索或改用系统程序。

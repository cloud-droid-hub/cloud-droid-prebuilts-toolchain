# Darwin arm64工具链

本目录由仓库唯一`main`分支维护，发布macOS arm64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、LLVM 22.1.8、裁剪后的Android NDK r29 revision 29.0.14206865、Bazel 8.5.1和Rust/Cargo 1.97.1。Rust内部LLVM为22.1.6，并包含`aarch64-linux-android`标准库。

LLVM 22.1.8按本仓消费者裁剪为195159848字节、3卷；原归档为1479903744字节、16卷，压缩量减少86.8%。版本、宿主架构和保留文件的原始字节不变，LLVM 19保持原样。新归档名为`LLVM-22.1.8-macOS-ARM64-cdh.tar.xz`，包内根目录仍为`LLVM-22.1.8-macOS-ARM64`；完整SHA-256与每卷大小、摘要由`internal/catalog/catalog.go`登记。

归档保留完整`include/`、`share/`和`lib/clang/`，以及`libc++`、`libc++abi`、`libunwind`的静态库、动态库和包内符号链接、`libclang.dylib`与`libclang-cpp.dylib`。上游许可证与通知均保留，完整LLVM许可证同时放在`share/licenses/LLVM-LICENSE.TXT`。保留的`bin/`程序和别名如下：

```text
clang clang++ clang-22 clang-cpp clang-scan-deps dsymutil
ld.lld ld64.lld lld lld-link wasm-ld
llvm-addr2line llvm-ar llvm-as llvm-bitcode-strip llvm-cov llvm-cxxfilt
llvm-dis llvm-dwp llvm-install-name-tool llvm-libtool-darwin llvm-link
llvm-lipo llvm-nm llvm-objcopy llvm-objdump llvm-otool llvm-profdata
llvm-ranlib llvm-readelf llvm-readobj llvm-size llvm-strings llvm-strip
llvm-symbolizer
```

`TestLLVM22Size`核对实际分卷的字节数、完整摘要、权限和独立文件身份，并限制归档不超过4卷。`TestLLVM22Build`经正式安装后运行C/C++编译链接、字符串与异常、目标文件和归档工具、剥离后执行以及`libclang`实际动态加载；Clang与LLD必须来自同一安装目录。

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
"$dest/bazel851/bin/bazel" --version
"$dest/rust197/bin/rustc" -vV
"$dest/rust197/bin/cargo" -V
"$dest/rust197/bin/rustdoc" --version
```

`install`按组件输出机器可读进度，字段固定为`component`、`index`、`total`、`remaining`、`elapsed_seconds`和`state`：

```text
toolchain_progress component="LLVM 22" index=4 total=7 remaining=3 elapsed_seconds=10 state=installing
```

组件展开期间默认每5秒持续输出`state=installing`；可用`CDH_TOOLCHAIN_PROGRESS_INTERVAL=15s`等大于零的Go duration调整。收到`SIGINT`或`SIGTERM`时返回非零，清理未发布的临时组件并保留此前已完成组件。中断后直接重跑原命令，已经完整安装并通过检查的组件显示`state=skipped`且不重读归档。删除一个组件只补一个，删除多个组件只补对应缺项。组件路径存在但检查失败时明确返回非零，不删除或覆盖该目录；目标根中的用户额外文件保持原样。使用者只有在自行备份并明确移走无效组件后，才能让安装器把该路径作为缺项补齐。

目标父目录链通常必须由当前用户或`root`拥有，且不可由组或其它用户写入。`install`使用的`Run`只对当前UID所有、与父目录跨设备、组可写但非`world`可写的挂载根通过已打开的目录FD追加`sticky`位，并保留原有`setuid`、`setgid`和权限位。`check`不修改权限，缺少`sticky`位时直接拒绝；非跨设备的普通`0775`目录和任何`0777`目录仍会被拒绝。目标目录链与组件路径使用`openat`和`O_NOFOLLOW`打开，权限调整、身份核对与发布均通过已打开的目录或文件FD完成。

安装检查验证摘要、版本、文件模式和Mach-O架构。任何检查失败都会返回非零退出码，不会搜索或改用系统程序。

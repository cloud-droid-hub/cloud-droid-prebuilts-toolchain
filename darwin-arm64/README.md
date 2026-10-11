# Darwin arm64工具链

`bash53/`提供GNU Bash `5.3.15(1)-release`，保留`feicong/bash-static`的原始Mach-O arm64字节和GNU GPLv3许可证。程序使用macOS系统运行库，归档550660字节、单卷。安装器支持`install`与`check`加`--component bash53`单独处理Bash，版本、架构和原始程序摘要由`internal/catalog/bash.go`核验。

本目录由仓库唯一`main`分支维护，发布macOS arm64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 22.1.8、裁剪后的Android NDK r29 revision 29.0.14206865、Bazel 8.5.1和Rust/Cargo 1.97.1。Rust内部LLVM为22.1.6，并包含`aarch64-linux-android`标准库。

JDK的`jdk21-build-v1`包为72392094字节、1卷，安装目录为`jdk21/`。完整上游包为200073404字节；保留完整运行模块与构建程序，移除`jmods/`、`man/`和`lib/src.zip`，保留JNI头文件及全部许可证。主仓`toolchains/jdk-trim/`负责固定输入、自动裁剪和字节重放核验。

LLVM 22.1.8按本仓消费者裁剪并把非登记别名转换为普通文件后，为351706968字节、4卷；原归档为1479903744字节、16卷，压缩量减少76.2%。此前保留20个符号链接的裁剪包为195159848字节、3卷；本次把其中18个别名改为独立普通文件，压缩量增加156547120字节。版本、宿主架构和保留文件的原始字节不变。新归档名为`LLVM-22.1.8-macOS-ARM64-cdh.tar.xz`，包内根目录仍为`LLVM-22.1.8-macOS-ARM64`；完整SHA-256与每卷大小、摘要由`internal/catalog/catalog.go`登记。

`aosp15/`提供固定`android-15.0.0_r36`的`bin/aidl`、`bin/hidl-gen`及`lib64/libc++.dylib`。三份原始通用二进制以登记LLVM22的`llvm-lipo -thin arm64`保留arm64部分，归档2317375字节、单卷、九份许可证，无链接。实际安装与AIDL／HIDL生成已验证，完整设备构建另行验收，见[AOSP15记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/10)。

0.14.1的`sdk37/`归档曾提供Build Tools 37.0.0的`aapt2`、`aidl`、`zipalign`、`lib64/libc++.dylib`、`d8`、`apksigner`与两个jar，以及Platform `android-36` r2的`android.jar`与`optional/libbinder_ndk_cpp`头文件；原生程序与运行库以登记LLVM22的`llvm-lipo -thin arm64`从官方`build-tools_r37_macosx.zip`取arm64部分。该归档46441963字节、单卷，本机实际安装后用该组件与JDK 21完成AIDL、资源、DEX、对齐、签名与回读，见[Android SDK记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/8)与[第二版记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/9)。

`sdk37/`供系统、HAL与源码App共用：Build Tools 34.0.0、35.0.0、35.0.1、36.0.0、37.0.0，Platform `android-34` r3、`android-35` r2、`android-36` r2、`android-37.0` r2，CMake 3.22.1与4.1.2；android-36保留`optional/libbinder_ndk_cpp`头文件。原生程序与运行库以登记LLVM22的`llvm-lipo -thin arm64`从官方通用程序取arm64部分；`platforms/android-37.0`与两个CMake按官方仓库XML条目补齐`package.xml`。归档219827507字节、3卷，本仓分片安装测试核验身份文件、版本探针与全部Mach-O为arm64，见[完整SDK记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/11)。

归档保留完整`include/`、`share/`和`lib/clang/`，以及`libc++`、`libc++abi`、`libunwind`的静态库、动态库与普通文件别名、`libclang.dylib`与`libclang-cpp.dylib`。仅保留登记的`bin/clang -> clang-22`和`bin/ld.lld -> lld`两项符号链接，其余8569个文件都是普通文件，普通文件总量为2211261752字节。上游许可证与通知均保留，完整LLVM许可证同时放在`share/licenses/LLVM-LICENSE.TXT`。保留的`bin/`程序和别名如下：

```text
clang clang++ clang-22 clang-cpp clang-scan-deps dsymutil
ld.lld ld64.lld lld lld-link wasm-ld
llvm-addr2line llvm-ar llvm-as llvm-bitcode-strip llvm-cov llvm-cxxfilt
llvm-dis llvm-dwp llvm-install-name-tool llvm-libtool-darwin llvm-link
llvm-lipo llvm-nm llvm-objcopy llvm-objdump llvm-otool llvm-profdata
llvm-ranlib llvm-readelf llvm-readobj llvm-size llvm-strings llvm-strip
llvm-symbolizer
```

`TestLLVM22Size`核对实际分卷的字节数、完整摘要、权限和独立文件身份，并限制归档不超过4卷。`TestLLVM22Links`只读TAR条目并拒绝非登记链接；`TestLLVM22Copy`可通过`CDH_LLVM22_TEST_ROOT`检查现有安装的18组普通文件别名。`TestLLVM22Build`经正式安装后先核对这18组文件的独立身份、权限与内容，再运行C/C++编译链接、字符串与异常、目标文件和归档工具、剥离后执行以及`libclang`实际动态加载；Clang与LLD必须来自同一安装目录。

在仓库根执行：

```sh
repo="$PWD/darwin-arm64"
dest="$PWD/output/darwin-arm64"
"$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest"
"$repo/bin/cdh-toolchain" check --dest "$dest"
"$dest/go126/bin/go" version
"$dest/bash53/bin/bash" --version
"$dest/jdk21/bin/java" -version
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
toolchain_progress component="LLVM 22" index=4 total=8 remaining=4 elapsed_seconds=10 state=installing
```

组件展开期间默认每5秒持续输出`state=installing`；可用`CDH_TOOLCHAIN_PROGRESS_INTERVAL=15s`等大于零的Go duration调整。收到`SIGINT`或`SIGTERM`时返回非零，清理未发布的临时组件并保留此前已完成组件。中断后直接重跑原命令，已经完整安装并通过检查的组件显示`state=skipped`且不重读归档。删除一个组件只补一个，删除多个组件只补对应缺项。组件路径存在但检查失败时明确返回非零，不删除或覆盖该目录；目标根中的用户额外文件保持原样。使用者只有在自行备份并明确移走无效组件后，才能让安装器把该路径作为缺项补齐。

目标父目录链通常必须由当前用户或`root`拥有，且不可由组或其它用户写入。`install`使用的`Run`只对当前UID所有、与父目录跨设备、组可写但非`world`可写的挂载根通过已打开的目录FD追加`sticky`位，并保留原有`setuid`、`setgid`和权限位。`check`不修改权限，缺少`sticky`位时直接拒绝；非跨设备的普通`0775`目录和任何`0777`目录仍会被拒绝。目标目录链与组件路径使用`openat`和`O_NOFOLLOW`打开，权限调整、身份核对与发布均通过已打开的目录或文件FD完成。

安装检查验证摘要、版本、文件模式和Mach-O架构。任何检查失败都会返回非零退出码，不会搜索或改用系统程序。

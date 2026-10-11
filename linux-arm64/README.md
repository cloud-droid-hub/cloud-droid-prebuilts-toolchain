# Linux arm64工具链

`bash53/`提供GNU Bash `5.3.15(1)-release`，保留`feicong/bash-static`的原始静态AArch64 ELF字节、GNU GPLv3和musl许可证，不依赖宿主动态链接器。归档551787字节、单卷。安装器支持`install`与`check`加`--component bash53`单独处理Bash，版本、架构和原始程序摘要由`internal/catalog/bash.go`核验。

本目录由仓库唯一`main`分支维护，发布Linux arm64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、Ubuntu 26.04 LLVM 22.1.2、Android NDK r29目标输入、Bazel 8.5.1和Rust/Cargo 1.97.1。Rust内部LLVM为22.1.6，并包含`aarch64-linux-android`标准库。

JDK的`jdk21-build-v1`包为75957224字节、1卷，安装目录为`jdk21trim1/`。完整上游包为205641175字节；保留完整运行模块与构建程序，移除`jmods/`、`man/`和`lib/src.zip`，保留JNI头文件及全部许可证。主仓`toolchains/jdk-trim/`负责固定输入、自动裁剪和字节重放核验。制品在macOSarm64生成和核验，本轮未在Linuxarm64执行。

NDK不包含Google发布的Linux amd64宿主程序。Android arm64编译使用同目录LLVM 22、NDK sysroot和Clang 21 resource目录，不得把NDK根交给要求`ndk-build`的旧入口。

`aosp15/`提供固定`android-15.0.0_r36`的原生`bin/aidl`、`bin/hidl-gen`、`lib64/libc++.so`与`lib64/libc_musl.so`，均保持上游`linux_musl-arm64`字节。归档3893445字节、单卷、九份许可证，无链接。Ubuntu26.04 arm64原生容器的实际安装及AIDL／HIDL生成通过，未在开发板执行完整设备构建，见[AOSP15记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/10)。

`sdk37/`提供Build Tools 37.0.0与Platform `android-36` r2的裁剪内容。Google不发布Linux arm64原生程序，`aapt2`、`aidl`、`zipalign`取已登记的Commit451 `android-arm-build-tools` 37.0.0归档原始字节（由`android-17.0.0_r1`源码构建，依赖宿主glibc、`libstdc++.so.6`与`libz.so.1`）；`d8`、`apksigner`、两个jar、`android.jar`与`optional/libbinder_ndk_cpp`头文件取官方归档。归档47392895字节、单卷。Ubuntu26.04 arm64原生容器实际安装并完成APK构建、签名与回读，见[Android SDK记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/8)与[第二版记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/9)。

`sdkapp37/`只含本宿主可登记的部分：Build Tools 37.0.0与Platform `android-34` r3、`android-35` r2、`android-36` r2、`android-37.0` r2。Build Tools 37.0.0的`aapt2`、`aidl`、`zipalign`与`sdk37`相同，取已登记的Commit451 37.0.0原始字节，其余文件取官方Linux zip；Build Tools 34.0.0至36.0.0与CMake没有登记的arm64来源，不在归档内。归档132611877字节、2卷，交叉编译的`TestAppSDK`在arm64 Linux容器内完成安装与核验，见[sdkapp37-v1记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/11)。

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
"$dest/bash53/bin/bash" --version
"$dest/jdk21trim1/bin/java" -version
"$dest/llvm22/bin/clang" --version
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

Android编译需传入`--target=aarch64-linux-android<API>`、`--rtlib=compiler-rt`及选定的`--ld-path`。安装检查验证摘要、版本、文件模式和ELF架构，失败时不会改用系统程序。

## LLVM22归档范围

LLVM22.1.2保留以下35个工具入口、已有Clang resource、共享库、CMake元数据和`lib/cdh/LICENSES/`中的27份许可证及来源说明。裁掉58个项目未使用的分析或辅助工具；LLVM19及其它工具链不变。

```text
clang clang++ clang-22 clang-cpp clang-scan-deps dsymutil
ld.lld ld64.lld lld lld-link wasm-ld
llvm-addr2line llvm-ar llvm-as llvm-bitcode-strip llvm-cov llvm-cxxfilt
llvm-dis llvm-dwp llvm-install-name-tool llvm-libtool-darwin llvm-link llvm-lipo
llvm-nm llvm-objcopy llvm-objdump llvm-otool llvm-profdata llvm-ranlib
llvm-readelf llvm-readobj llvm-size llvm-strings llvm-strip llvm-symbolizer
```

归档中仅`bin/ld.lld -> lld`保留登记链接，`clang`本来就是普通文件。其它45个工具与库别名按最终目标字节和权限保存为独立普通文件。外层`libLLVM`、`libclang`的`RUNPATH`为`$ORIGIN`，因此另加入下表四份普通副本，沿用原库的许可证，不改二进制或宿主依赖允许列表。

| 目标路径 | 原始来源 | 字节 |
| --- | --- | ---: |
| `lib/libffi.so.8` | `lib/cdh/libffi.so.8.2.0` | 199009 |
| `lib/libedit.so.2` | `lib/cdh/libedit.so.2.0.76` | 334905 |
| `lib/libz3.so.4` | `lib/cdh/libz3.so.4` | 17396073 |
| `lib/libxml2.so.16` | `lib/cdh/libxml2.so.16.1.2` | 1614633 |

普通文件合计966508007字节、418个，归档共441个条目。压缩后154451176字节、2卷，SHA-256为`18374afd0db2a4a868d51f21f7aa1cc2646de41fec4e94b4e8e955575cc17027`。旧包为85458612字节、1卷；遵守普通文件别名规则并补齐上述依赖后，压缩包增加68992564字节。四份依赖副本本身增加19544620个展开字节，与未补依赖的普通文件候选相比，压缩包增加5950184字节。

Linux消费者的`LIBCLANG_PATH`仍必须使用真实目录`llvm22/lib/cdh`。本包保持Ubuntu26.04库输入及glibc2.43宿主要求，不声明Ubuntu24.04兼容。`LLVMPolly.so`原样保留，但原包和候选均不能在普通进程中直接`dlopen`；额外通过原包`opt -load`加载也因重复注册Polly选项失败。这不属于登记消费者，不能据此宣称所有保留共享对象均可独立加载。

`TestLLVM22Arc`只读取分片，核对摘要、链接类型、普通文件别名字节与许可证，不建立链接。原有`TestLLVM22`继续核对全部ELF依赖；`TestLLVM22Tools`、`TestLLVM22Copies`与`TestLLVM22Run`分别检查35个原生工具、四份副本的独立身份，以及C/C++编译运行。后三项使用`CDH_LLVM22_TEST_ROOT`指定已安装目录，路径及父目录必须规范且真实。额外诊断`TestLLVM22Load`在独立进程逐项加载68个库，显式跳过`LLVMPolly.so`插件的独立加载；首次69项诊断中该插件失败的日志保留，跳过不计为通过，原有ELF依赖检查仍覆盖该文件。

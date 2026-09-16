# Linux amd64工具链

本目录由仓库唯一`main`分支维护，发布Linux amd64使用的Go 1.26.8、Temurin JDK 21.0.12.1+1、LLVM 19.1.7、Ubuntu 26.04 LLVM 22.1.2、裁剪后的Android NDK r29 revision 29.0.14206865、Bazel 8.5.1和Rust/Cargo 1.97.1。Rust内部LLVM为22.1.6，并包含`aarch64-linux-android`标准库。

LLVM 22归档内的`libedit2`与`libgnutls30t64`固定取自Ubuntu 24.04 LTS，使工具可在glibc 2.39及更新的amd64宿主运行；LLVM程序与其它随附运行库仍来自登记的Ubuntu 26.04软件包。归档同时携带登记的`libssl`与`libcrypto`，不依赖宿主OpenSSL的小版本补齐`libcurl`。

`aosp15/`提供固定`android-15.0.0_r36`的原生`bin/aidl`、`bin/hidl-gen`与`lib64/libc++.so`，均保持上游`linux-x86`字节。归档3462954字节、单卷、九份许可证，无链接。NUC的Ubuntu24.04 amd64实际安装及AIDL／HIDL生成通过，未据此声明完整设备构建或发行通过，见[AOSP15记录](../docs/evidence/aosp15-v1/README.md)。

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
toolchain_progress component="LLVM 22" index=4 total=8 remaining=4 elapsed_seconds=10 state=installing
```

组件展开期间默认每5秒持续输出`state=installing`；可用`CDH_TOOLCHAIN_PROGRESS_INTERVAL=15s`等大于零的Go duration调整。收到`SIGINT`或`SIGTERM`时返回非零，清理未发布的临时组件并保留此前已完成组件。中断后直接重跑原命令，已经完整安装并通过检查的组件显示`state=skipped`且不重读归档。删除一个组件只补一个，删除多个组件只补对应缺项。组件路径存在但检查失败时明确返回非零，不删除或覆盖该目录；目标根中的用户额外文件保持原样。使用者只有在自行备份并明确移走无效组件后，才能让安装器把该路径作为缺项补齐。

目标父目录链通常必须由当前用户或`root`拥有，且不可由组或其它用户写入。`install`使用的`Run`只对当前UID所有、与父目录跨设备、组可写但非`world`可写的挂载根通过已打开的目录FD追加`sticky`位，并保留原有`setuid`、`setgid`和权限位。`check`不修改权限，缺少`sticky`位时直接拒绝；非跨设备的普通`0775`目录和任何`0777`目录仍会被拒绝。目标目录链与组件路径使用`openat`和`O_NOFOLLOW`打开，权限调整、身份核对与发布均通过已打开的目录或文件FD完成。

安装检查验证摘要、版本、文件模式和ELF x86-64架构。任何检查失败都会返回非零退出码，不会搜索或改用系统程序。Linux amd64只用于构建，不表示支持运行云机。

## 脚本探针

Linux脚本探针通过已打开的父目录定位脚本，使`ndk-build`能够按`$0`查找同目录的`build/ndk-build`。执行前核对脚本与目录中的文件身份，取消时终止子进程；二进制探针继续保留登记的调用名称。

2026-09-16在Ubuntu24.04 amd64的NUC上，`TestScriptPeer`先因相邻资源路径错误退出1；修正后与`TestScriptSwap`、`TestProbeCancel`一同退出0。CLI、目录清单和工具选择包测试、`go vet ./...`退出0，使用登记Go1.26.8原生构建的候选程序完成安装及全部七个组件的`check`，退出0。证据保存在NUC主仓`build/ndk-probe-red-v2.log`、`build/ndk-probe-green-v4.log`、`build/ndk-probe-cli-test-v2.log`、`build/ndk-probe-vet.log`、`build/ndk-probe-candidate-install.log`和`build/ndk-probe-final-check.log`。

测试目录链必须满足上述权限要求，本次测试进程使用`umask 022`和仓内独立临时目录。此前扩展回归因测试目录组可写而退出1，日志保留；完整安装测试中仍有会创建符号链接的历史用例，本次未执行。主仓正式入口的复验随组件引用同步继续进行，由[#6](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/6)跟踪。

## LLVM22普通文件归档

本轮归档保留LLVM22.1.2的35个工具入口、Clang resource、已有库与CMake元数据，以及全部28份许可证和来源说明。42个工具与库别名改为最终目标字节的普通文件，仅保留登记的`bin/ld.lld -> lld`；`bin/clang`原本就是普通文件。`LIBCLANG_PATH`继续使用真实目录`lib/cdh`。

外层LLVM与libclang库使用`DT_RPATH=$ORIGIN`，libclang需要`libLLVM.so.22.1`。此前普通文件候选缺少这个外层名称，以及LLVM所需的四个外层依赖名称。按既有库的实际依赖补入以下五份普通副本，不改原库字节或权限：

NUC的Ubuntu24.04.4原始正式安装对照中，外层libclang与真实`lib/cdh`入口分别在新进程加载，均退出0；外层输入是已有符号链接，`ldd`显示其LLVM、ffi、edit、z3来自宿主。这个结果不支持“原包加载必然失败”的推断，也不证明普通文件布局可独立解析依赖。本次对照后不再执行该旧链接，新候选验收使用独立普通文件并核对全部ELF依赖来源。

| 归档内目标 | 归档内来源 | 字节数 |
| --- | --- | ---: |
| `lib/libLLVM.so.22.1` | `lib/cdh/libLLVM.so.22.1` | 149622136 |
| `lib/libffi.so.8` | `lib/cdh/libffi.so.8.2.0` | 67897 |
| `lib/libedit.so.2` | `lib/cdh/libedit.so.2.0.72` | 228425 |
| `lib/libz3.so.4` | `lib/cdh/libz3.so.4` | 19059105 |
| `lib/libxml2.so.16` | `lib/cdh/libxml2.so.16.1.2` | 1491753 |

五份副本增加170469316字节展开量，沿用原包中的`llvm-copyright`、`libffi-copyright`、`libedit-copyright`、`libz3-copyright`及`libxml2-copyright`。Ubuntu24.04的`libedit`、`libgnutls`和已有`libssl`、`libcrypto`保持原始字节；GLIBC需求检查仍拒绝高于2.39的版本，原测试的全部ELF依赖检查及宿主库允许列表保持不变。

候选包含419个普通文件、1个登记链接，共815168159字节普通文件数据。压缩后141057060字节、2卷，比旧正式单卷77077396字节增加63979664字节；普通文件身份要求与补齐依赖使包变大，本项不宣称缩小。完整SHA-256为`55f4bf7ce313c0f4af2ad36152086686ad3e076b7bbfbc27bb0381118b031517`。

主仓`output/llvm22-trim/regular-v3/linux-amd64/`保存候选和分片。`pack`与`verify`使用同一组五项映射，严格路径、模式、字节与许可证检查退出0。macOS上用登记Go1.26.8执行多分片、归档与GLIBC三项行为测试退出0，三个宿主模块的catalog测试和`go vet ./...`退出0。

在NUC的Ubuntu24.04.4 amd64上，从新两卷调用已提交的`llvm22-check`全新安装，109.103秒退出0；正式`RunTo/Check`、同目录Clang/LLD22、C/C++字符串与异常运行、静态归档、LLVM工具、剥离后运行及真实`lib/cdh`下libclang加载均通过，434个受检普通文件保持独立身份。安装根为该机主仓`build/llvm22-amd64-v3/install/llvm22`，探测目录为同级`probe`。

35个工具与68个库分别在独立进程通过，五份普通副本的字节、权限和不同inode检查通过，GLIBC检查52.06秒退出0。`LLVMPolly.so`保留原ELF依赖检查，独立加载明确跳过1项，不计为加载通过。首次聚焦组最后的原`TestLLVM22`因NUC检出的`PACKAGES.md`权限为0664退出1，未到达该项ELF检查；逐文件确认28份许可证为普通单链接后，按授权将它们规范为0644，实际更改`PACKAGES.md`与`libgnutls-copyright`两项权限，内容不变。仅重跑原`TestLLVM22`，完整ELF依赖检查及原有编译链接检查2.29秒退出0；没有重跑已通过的GLIBC测试，也没有放宽允许模式或依赖列表。外层候选的libclang实际加载退出0，依赖解析确认五个新增名称均来自候选目录。

两卷通过普通独立复制、摘要与权限核对及原子目录交换进入正式`payload/llvm22/`，旧单卷保存在组件`build/llvm22-trim/linux-amd64-regular-v1/payload-swap/`。静态x86-64安装程序同样通过独立复制和原子交换进入`bin/cdh-toolchain`，旧程序保存在同级`installer-swap`；新程序SHA-256为`6e722d4d527bd47149fbdb2257a901338b008ab05e16ba814f6cb2007b353c71`。

候选检查用`CDH_LLVM22_PARTS`指向上述`payload`子目录，默认归档测试读取正式分片。发布后清空该变量，`go test ./tests -run '^TestLLVM22Arc$' -count=1 -v`直接检查正式两卷，90.638秒退出0，确认419个普通文件、1个登记链接与28份许可证。原始日志位于主仓`output/llvm22-trim/linux-amd64-formal-v1/`，包括单分片读取、旧归档链接、首次catalog和NUC许可证权限检查的失败记录，正式交换见`formal-publish.log`，正式归档检查见`formal-archive.log`。Ubuntu26.04 amd64、完整QEMU、安全组件或设备运行验收尚未执行。

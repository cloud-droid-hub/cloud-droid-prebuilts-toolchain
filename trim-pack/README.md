# 现有工具链裁剪生成

`trim-pack`从固定摘要的原始发行包或原版文件生成LLVM22、NDK29、SDK37、SDKApp37和AOSP15的三宿主裁剪归档。源文件、文件过滤、工具集合、保留链接、macOS arm64切片和元数据均由编译期Go定义登记。`pack`不读取现有裁剪包；只有`verify --published`可把现有包作为对照。

OpenJDK使用相邻的`jdk-trim`工具。Go使用完整上游发行包；Bazel使用原版可执行程序与许可证；Rust保留官方rustc、Cargo、宿主及安卓标准库组件的完整内容，不属于文件裁剪。Linux arm64的安卓标准库由原版组件内容组装，没有该目标的安装manifest；其它组件的原版安装manifest保留。组件选择和安装元数据与内容裁剪分别核对。这些组件不套用本工具的裁剪策略。Bash的原版构建与安装由其工具链入口维护。

## 正式入口

工作目录为主仓根，先准备`--source`所需的完整原始文件。工具不下载、不调用Git，也不接受手工删文件后的目录。输出父目录须已存在，已有输出保留并明确失败。

```sh
mkdir -p build/trim-inputs build/trim-result
./tools/build/cwd-just.sh test-trim-pack
./tools/build/cwd-just.sh trim-pack sdk37 darwin-arm64 build/trim-inputs build/trim-result/sdk37
```

五个组件统一使用同一参数顺序，宿主为`darwin-arm64`、`linux-arm64`或`linux-amd64`。最后一个可选参数为当前组件分片目录：

```sh
./tools/build/cwd-just.sh trim-pack ndk29 linux-arm64 build/trim-inputs build/trim-result/ndk29 \
  tools/toolchains/linux-arm64/payload/ndk29
```

配方先`pack`，再从相同原始输入`verify`。默认构建本工具；需要用同一个已构建程序重放时，通过绝对路径`CDH_TRIM_BIN`指定该可执行文件。两种方式都先执行正式预编译工具链检查。独立程序的三个操作如下：

```sh
build/trim-pack/trim-pack inputs --component aosp15 --host darwin-arm64
build/trim-pack/trim-pack pack --component aosp15 --host darwin-arm64 \
  --source build/trim-inputs --dest build/trim-result/aosp15
build/trim-pack/trim-pack verify --component aosp15 --host darwin-arm64 \
  --source build/trim-inputs --dest build/trim-result/aosp15 \
  --published tools/toolchains/darwin-arm64/payload/aosp15
```

`inputs`输出以制表符分隔的文件名、SHA-256和来源URL。AOSP Gitiles的`?format=TEXT`响应是Base64，保存输入前须解码；摘要针对解码后的原始字节。其它URL的摘要针对下载文件。Google仓库XML的URL会变化，本工具只接受已登记摘要的完整副本，不能用新版XML或仅抽出的许可文本替代。

## 固定来源与策略

| 组件 | 原始来源 | 裁剪及转换 |
| --- | --- | --- |
| LLVM22 macOS | LLVM官方`llvmorg-22.1.8`的`LLVM-22.1.8-macOS-ARM64.tar.xz`及同tag的完整许可证 | 保留登记的35个工具、头文件、Clang资源、C++/unwind/Clang库与共享数据，删除未保留工具和指定空目录；按类型化规则展开别名，保留`clang`、`ld.lld`链接 |
| LLVM22 Linux | `debs.go`登记的Ubuntu官方DEB版本、大小、摘要及路径；附固定patchelf 0.18.0源码 | 九个LLVM包选取登记工具、头文件、资源和库；按固定DEB集合加入运行库闭包及每类版权；生成来源表；重放相对ELF RPATH变换 |
| NDK29 | Google官方r29 Darwin/Linux ZIP | 保留安卓AArch64头文件、API库、Clang 21资源及版权；macOS/Linux amd64还保留构建工具、API 21–35包装入口、构建定义及cpufeatures；macOS选取arm64切片；Linux arm64为目标数据包 |
| SDK37 | Google Build Tools 37、Platform 36；Linux arm64加入固定Commit451原生Build Tools | 保留AIDL、aapt2、zipalign、D8、apksigner、必需运行库、android.jar、Binder NDK C++数据及完整NOTICE；macOS选取arm64切片 |
| SDKApp37 | Google Build Tools 34/35/35.0.1/36/37、Platform 34/35/36/37、CMake 3.22.1/4.1.2和固定仓库XML；Linux arm64加入同一Commit451输入 | 删除旧RenderScript、非目标链接器、CMake文档及未保留工具、平台皮肤/模板/资源与stub源；保留Java/native构建内容及许可；生成固定版本包元数据；Linux arm64只保留Build Tools 37及四个平台 |
| AOSP15 | AOSP`android-15.0.0_r36`的原版aidl、hidl-gen、宿主共享库和九份许可证 | 按宿主保留程序与运行库，macOS选取arm64切片，生成固定`source.properties` |

Linux LLVM的DEB输入以[Ubuntu快照](https://snapshot.ubuntu.com/)的固定时间路径登记；快照缺少的已登记Z3和libpfm amd64包使用官方ports/archive地址，仍以固定摘要校验。amd64的libedit与GnuTLS使用登记的Ubuntu 24.04版本，其余为Ubuntu 26.04版本。原始输入可以离线保存；来源服务器不再提供某版本时不能换包后继续。

Linux LLVM生成会在职责内`build/`工作目录，从固定源码编译原生patchelf。编译器来自绝对路径`CDH_TOOLCHAIN_ROOT/llvm22/bin/clang++`，须为LLVM22；链接使用同套LLD22。macOS使用出厂`xcrun`提供的SDK，C++、ABI与unwind来自登记LLVM；源码及子进程临时目录均绑定本轮工作根。patchelf对动态ELF文件逐项执行`--set-rpath`，内容为`$ORIGIN`加当前文件目录到`lib/cdh`的相对路径。arm64保留`DT_RUNPATH`；amd64使用`--force-rpath`保留`DT_RPATH`，其中libcrypto、libssl、libedit与GnuTLS的四个登记文件保留`DT_RUNPATH`。amd64的`lib/cdh/`下libLLVM、libclang和libclang-cpp三个登记文件先设置`$ORIGIN/cdh`，再设置最终`$ORIGIN`，同时重放现有包在`.dynstr`中留下的中间变换；其它ELF不增加该步骤。该过程可以在macOS生成两种Linux架构的归档，归档生成与Linux原生消费者运行分别验收。

`lib/cdh/LICENSES/PACKAGES.md`保存现有组件的完整来源表及原制品构建信息，包括原Ubuntu patchelf版本和已有宿主检查范围。生成器保留这份内容，并在当前`report.json`中记录实际重放所用的上游patchelf 0.18.0变换；该来源表不充当本轮Linux原生运行证据。

NDK的根`NOTICE`与`NOTICE.toolchain`保持上游完整字节。Linux amd64不单独复制工具链内层`NOTICE`：该文件130424字节，SHA-256为`f96f763beb66a7ba7a667647fc64c0226ace875e590c831fdd9579ec1c1d91e1`；其完整内容连续出现在799639字节的根`NOTICE.toolchain`中，从零计数的偏移为95918，根合集SHA-256为`789714a9076571c580cf531bdcebec776a4d889c12c3c1108adafc6e4338710a`。省略的是合集已完整覆盖的重复副本，内层文件与根合集并非相同文件。

## 重放、对照与保护

完整输入先复制为独立快照，核对SHA-256、普通文件类型、原路径身份、长度及读取期间变化。归档路径必须规范，选中链接必须指向登记文件并可解析，不允许越界和循环；macOS的出厂libarchive仅把XZ转为TAR流，不把归档路径展开到宿主目录。其它压缩输入由Go解码。

输出为按路径排序的PAX TAR和固定参数gzip：时间为Unix epoch，UID/GID为零，普通文件`0644`，目录与含执行位的文件`0755`。声明保留的符号链接保持目标；其余别名写为普通文件。`payload.tar.gz`同时按90MiB生成连续分片；`report.json`记录输入摘要与来源、策略、转换、移除项、数据长度、条目数、压缩摘要及分片摘要。输出目录通过宿主不覆盖重命名原子发布，已有目标、路径替换、取消和篡改均明确拒绝；当前工作盘可用空间须不少于50GiB。

`verify`从原始源重新生成归档流，逐项核对完整归档、分片与报告，不信任输入报告给出的摘要。可选的现有制品对照先核对登记的完整压缩摘要，再比较路径集合、类型、执行权限、链接目标及所有普通文件的原始字节。旧手工归档的压缩参数、时间及UID/GID可以不同，不能把内容等价写成新旧压缩字节相同。

两次重放使用同一工具、策略和原始输入，创建两个新输出目录；两次`pack`与`verify`均须成功，两个目录的完整归档、分片和报告须逐文件相同。迁移现有组件前，再执行带`--published`的对照及该组件的实际消费者验收。行为测试覆盖确定性输出、原始输入保护、摘要/分片篡改、目标冲突、取消、链接、宿主切片、DEB解码、包许可元数据及新旧压缩格式差异。

## 本机消费者检查

macOS arm64的`TestNative`读取`CDH_TRIM_NATIVE`下的`<组件>-darwin-arm64/payload.tar.gz`及报告，验证摘要后为每个组件建立独立职责内展开目录。它实际运行生成的Clang/LLD22编译并执行宿主程序，使用NDK的Clang/LLD21链接安卓AArch64程序，分别用SDK37和SDKApp37完成AIDL、资源、Java、DEX、对齐及APK签名校验，运行两版CMake与Ninja，并执行AOSP15的AIDL/HIDL接口生成。JDK由登记的`jdk21trim1/`提供，Java临时目录显式绑定测试工作根。安卓ELF只检查链接结果和目标架构，不在macOS执行；Linux原生消费者须单独验证。

两轮分别使用两个完整生成结果目录，测试程序只编译一次：

```sh
export CDH_TOOLCHAIN_ROOT="$PWD/output/toolchains/darwin-arm64"
./tools/build/cwd-go.sh auto -C tools/toolchains/trim-pack test -c -buildvcs=false \
  -o "$PWD/build/trim-native.test" .
CDH_TRIM_NATIVE="$PWD/build/trim-round-1" build/trim-native.test \
  -test.v -test.run '^TestNative$' -test.count=1
CDH_TRIM_NATIVE="$PWD/build/trim-round-2" build/trim-native.test \
  -test.v -test.run '^TestNative$' -test.count=1
```

没有提供生成目录或宿主不支持时，`TestNative`明确报告跳过；普通单元测试通过不能替代实际消费者运行。

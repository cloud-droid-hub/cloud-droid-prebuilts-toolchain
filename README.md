# cloud-droid-prebuilts-toolchain

三个宿主提供CPython`3.14.8`，安装到`python314/`。输入固定为Astral`python-build-standalone`的`20261009`完整`pgo+lto`归档；主仓`tools/toolchains/python-trim/`按`python314-build-v1`生成裁剪包，移除构建对象、libpython静态副本、测试、字节码缓存和手册，保留运行库、标准库、pip、ensurepip、venv、头文件与完整上游许可证。安装器分别核对原生架构、版本、裁剪标记与关键模块实际导入，调用方不改用系统Python。

| 宿主 | 裁剪tar.gz大小 | SHA256 |
| --- | --- | --- |
| darwin-arm64 | 27243497 | `3a417e4a2ca1ce63b2e3dcfa142a1febf13d394e2c29c5d96f9b8a9fa1758ac4` |
| linux-arm64 | 58800714 | `ce3f118977c3b8b0299b55ed24182c4f68d262825ce638eed7ffdb57ffa7115b` |
| linux-amd64 | 76519762 | `7605cd4099a4dc2dc51cb605d26a12f98c11fa0764b1e6c8122155bd9babab91` |

本私有仓库为`cloud-droid-hub`提供本地与Docker构建所需的固定宿主原生工具链。仓库只维护`main`分支，三个原平台分支的内容和历史分别归入以下顶层目录：

```text
darwin-arm64/
linux-arm64/
linux-amd64/
```

每个目录包含自己的安装程序、类型化归档目录、分片、许可证与测试。仓库统一版本由根目录[version.go](version.go)维护，三个宿主共同遵守根目录[AGENTS.md](AGENTS.md)。平台目录固定发布Go 1.26.8、LLVM 22、Temurin JDK 21.0.12.1+1、Android NDK r29 revision 29.0.14206865、Bazel 8.5.1和Rust/Cargo/Rustdoc 1.97.1。macOS LLVM 22为22.1.8，Linux LLVM 22为Ubuntu 26.04的22.1.2；NDK内Clang与LLD主版本为21，Rust内部LLVM为22.1.6。三个宿主目录都包含`aarch64-linux-android`标准库。

归档以不超过90 MiB的普通Git对象保存，不使用Git LFS。`<host>/bin/cdh-toolchain`负责摘要校验、安全展开、宿主架构检查和逐组件原子发布。主项目把本仓放在`tools/toolchains/`，再把当前宿主工具展开到`output/toolchains/<host>/`；缺失、版本错误或架构不符都会直接失败，不会改用系统工具。

本仓同时维护环境与安装包装脚本、AIDL构建输入、ARM数据与源码工具以及LLVM、JDK、Python裁剪器。组件根Go模块覆盖包装入口的行为测试及`python-trim/`，三个宿主安装器和其它生成器保持独立模块。生成器通过相对路径使用主仓的`pkg/devicekit`工作根接口；主仓正式入口负责绑定临时目录及离线依赖，普通源码包按相同布局使用，无需Git元数据。

在主仓根调用`./tools/build/cwd-just.sh test-toolchain test-python-trim test-jdk-trim test-arm-data test-llvm-pack`核验这些工具。组件根测试显式选择根包与`python-trim`，不把`build/`中的上游源码和归档测试数据作为首方包扫描。迁移及入口验证见[工具链目录归属记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/15)。

三个宿主的Temurin JDK使用`jdk21-build-v1`裁剪构建配置，安装到`jdk21trim1/`。保留完整运行模块、全部原版程序、`ct.sym`、JNI头文件、配置与许可证，移除`jmods/`、`man/`和`lib/src.zip`；需要通过`jlink`生成其它模块镜像时，显式从完整上游包提供`--module-path`。三个压缩包合计225043301字节，相比完整上游包的613187926字节减少63.3%，各宿主只需一卷。裁剪由本仓`jdk-trim/`的类型化策略自动执行，输入绑定固定上游SHA-256，输出排序、权限、属主与时间戳规范化；`pack`生成归档、90MiB分片及动态报告，`verify`从同一输入重新计算完整输出，核对归档、分片和报告。具体命令见[jdk-trim/README.md](jdk-trim/README.md)；类型化目录同时核验裁剪标记和`java`、`javac`、`jar`、`keytool`身份，旧的`jdk21/`目录不会被覆盖或复用。

三个宿主还提供固定`android-15.0.0_r36`的AIDL与HIDL生成工具，统一安装到`aosp15/`，使调用方可以直接使用登记输入。三份归档合计9673774字节，均为普通文件；来源、许可证、安装与实际代码生成结果见[AOSP15工具记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/10)。设备构建器接入单独提交，本次工具发布不能代替完整Codec2或发行验收。

三个宿主还提供裁剪后的Android SDK，安装到`sdk37/`并保持标准SDK根布局：Build Tools 37.0.0的`aapt2`、`aidl`、`zipalign`、`d8`、`apksigner`及两个jar，Platform `android-36` r2的`android.jar`与`optional/libbinder_ndk_cpp`头文件，以及上游许可证与版本元数据。主项目把`ANDROID_HOME`指向该目录，不再依赖宿主SDK或构建时下载。macOS只保留官方通用程序的arm64部分；Linux amd64保持官方字节；Google不发布Linux arm64原生程序，该宿主的`aapt2`、`aidl`、`zipalign`取已登记的Commit451 `android-arm-build-tools` 37.0.0归档原始字节。来源、摘要与三宿主APK构建验证见[Android SDK记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/8)；0.14.1补入头文件并改用`sdk37/`安装目录的原因与验证见[第二版记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/9)。0.14.0安装留下的`android-sdk/`不再被引用，安装器不会删除它。

三个宿主另提供主项目源码App构建用的`sdkapp37/`，同样保持标准SDK根布局：macOS arm64与Linux amd64含Build Tools 34.0.0、35.0.0、35.0.1、36.0.0、37.0.0，Platform `android-34` r3、`android-35` r2、`android-36` r2、`android-37.0` r2，CMake 3.22.1与4.1.2；Linux arm64只含Build Tools 37.0.0与这些Platform，原生程序与`sdk37`一样取Commit451原始字节，其它Build Tools版本与CMake没有登记的arm64来源。内容取自Google仓库XML登记的官方zip，macOS只保留arm64部分，不含NDK。主项目只在macOS arm64构建源码App。来源摘要、逐轮补齐过程、裁剪范围与三宿主核验见[sdkapp37-v1记录](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/11)。

本仓证据不存入仓库：每个Step的日志、截图与命令输出作为附件上传到本仓单独的Step Issue，全部Step Issue在索引单[#7](https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/issues/7)中列出。

## 安装当前平台

Bash统一安装到`bash53/`，版本为`5.3.15(1)-release`。三个程序取自[feicong/bash-static](https://github.com/feicong/bash-static)：macOS arm64与Linux arm64来自[成功构建35507495813](https://github.com/feicong/bash-static/actions/runs/35507495813)，Linux amd64来自[成功构建35507468329](https://github.com/feicong/bash-static/actions/runs/35507468329)，保留原始程序字节。Linux程序为静态ELF，macOS程序为arm64 Mach-O并依赖系统`libncurses.5.4`、`libiconv.2`、CoreFoundation与libSystem；不能把macOS程序称为静态链接。三个包包含GNU GPLv3，Linux包还包含musl许可证。归档合计1647101字节，各宿主一卷，完整摘要、程序摘要与版本探针由`internal/catalog/bash.go`登记。

主项目的`tools/toolchains/bash.sh`用POSIX入口调用本仓安装器，只展开Bash后才执行脚本体。单独安装与检查Bash使用：

```sh
host=darwin-arm64
repo="$PWD/$host"
dest="$PWD/output/$host"
"$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest" --component bash53
"$repo/bin/cdh-toolchain" check --dest "$dest" --component bash53
"$dest/bash53/bin/bash" --version
```

`--component`取类型化目录中的安装目录名，未知名称返回退出码2；省略时仍处理全部工具链。Bash冷启动不依赖其它组件已经安装，也不要求完整源码包存在Git元数据。普通源码包须包含对应宿主的安装器与Bash分片。

从仓库根目录指定宿主目录：

```sh
host=darwin-arm64
repo="$PWD/$host"
dest="$PWD/output/$host"
"$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest"
"$repo/bin/cdh-toolchain" check --dest "$dest"
```

## 逐组件进度与继续安装

`install`在检查和展开每个组件时输出稳定、机器可读的进度行，例如：

```text
toolchain_progress component="LLVM 22" index=4 total=8 remaining=4 elapsed_seconds=10 state=installing
```

字段语义如下：

- `component`是当前组件名。
- `index`和`total`分别是从1开始的当前序号及组件总数。
- `remaining`是当前组件之后尚待处理的组件数。
- `elapsed_seconds`是当前组件从开始检查起经过的整秒数。
- `state`依次使用`checking`、`installing`、`installed`或`skipped`。

组件展开期间默认每5秒输出一次`state=installing`心跳。可通过大于零的Go duration调整间隔，非法值会在目标目录发生变化前明确失败：

```sh
CDH_TOOLCHAIN_PROGRESS_INTERVAL=15s \
  "$repo/bin/cdh-toolchain" install --repo "$repo" --dest "$dest"
```

每个组件在临时目录完成摘要、内容、模式和身份检查后才以不替换方式发布。收到`SIGINT`或`SIGTERM`时，CLI返回非零并清理本轮未发布的临时组件；此前已经发布的组件保持不变。安装中断后，使用原命令再次执行即可：已经完整安装并通过组件级检查的目录显示`state=skipped`，其归档不会重新读取；路径不存在的组件才会重新展开。因此完整安装后删除一个组件只补一个，删除多个组件只补对应缺项。

组件路径已经存在但检查失败时，安装器返回非零且不删除、不覆盖该目录；目标根目录中的其它用户文件也保持原样。安装器不会把不明内容当作可自动修复的缺项。需要重装无效组件时，使用者应先自行备份并明确移走该组件目录，再重新执行安装。边界文件`go.mod`已存在但内容或权限无效时同样失败，不会重写。

目标路径的各级父目录通常必须由当前用户或`root`拥有，且不可由组或其它用户写入。`install`使用的`Run`只对当前UID所有、与父目录跨设备、组可写但非`world`可写的挂载根放宽这一限制：它通过已打开的目录FD只追加`sticky`位，并保留原有`setuid`、`setgid`和权限位。`check`不修改权限，遇到缺少`sticky`位的同类挂载根时直接拒绝；非跨设备的普通`0775`目录和任何`0777`目录仍会被拒绝。目标目录链与组件路径使用`openat`和`O_NOFOLLOW`打开，权限调整、身份核对与发布均通过已打开的目录或文件FD完成。

Linux可把`host`改为`linux-arm64`或`linux-amd64`。Linux安装程序还提供`select`，用于在完整检查后返回明确工具路径：

```sh
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-clang
"$repo/bin/cdh-toolchain" select --dest "$dest" --name llvm22-lld
"$repo/bin/cdh-toolchain" select --dest "$dest" --name ndk
```

Linux arm64的NDK只提供Android arm64目标输入。该平台使用同目录LLVM 22、NDK sysroot和Clang 21 resource目录，不执行Linux amd64 NDK程序。

各平台的详细版本、限制与手工验证命令见对应目录的`README.md`。迁移设计和验证记录见`docs/plans/2026-09-12-预编译工具链单一主分支设计及实施计划.md`。

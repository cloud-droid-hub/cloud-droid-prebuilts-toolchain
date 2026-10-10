# OpenJDK自动裁剪与重放

`jdk-trim`从登记的Temurin21LTS`21.0.12.1+1`完整上游包生成`jdk21-build-v1`构建工具链。Linuxarm64、Linuxamd64与macOSarm64共用一个命令、一套参数和同一删减策略，生成过程只处理归档字节，不运行输入里的程序，因此可在任一受支持宿主生成三个平台的制品。原生安装和调用仍需在实际宿主分别验收。

裁剪保留全部`bin/`程序、完整`lib/modules`、跨Java版本编译所需的`lib/ct.sym`、JNI头文件、配置、证书、原生运行库、版本文件、NOTICE和全部许可证。只移除`jmods/`、`man/`与`lib/src.zip`，不改写保留的二进制或模块字节。`jlink`需要显式从完整上游包提供`--module-path <jmods目录>`；本配置不提供JDK源码、手册或可用于模块链接的模块归档。

## 固定输入

`spec.go`编译进三个宿主的源归档名、根目录、完整SHA-256及原始分片长度，不读取外部静态配置。输入可以是完整上游`tar.gz`，也可以是依原始名称保存分片的目录。输入摘要不符、缺少必需构建文件、重复条目、路径越界、异常链接或链接循环都会返回非零；源归档保持原字节、模式和文件身份。

上游文件来自`adoptium/temurin21-binaries`的固定`jdk-21.0.12.1+1`发布：

| 宿主 | 上游文件 | 完整SHA-256 |
| --- | --- | --- |
| `darwin-arm64` | `OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz` | `3623232f33a9c3baadf304480b2535f9a3cba8a58d42ecbb438ba267315d9998` |
| `linux-arm64` | `OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.12.1_1.tar.gz` | `23e37e026f12f3e706f18938ff611db3032d075b09d0879a25d06718c773e223` |
| `linux-amd64` | `OpenJDK21U-jdk_x64_linux_hotspot_21.0.12.1_1.tar.gz` | `ce79869e1307ed8ee1e2baa86a412b1eb5b75d10a01006d788a6f968bcfaee94` |

## 正式命令

在主仓根目录执行，`source`与`dest`均由调用者明确提供。先把完整上游包放到职责内的`build/`目录，输出父目录必须已存在且属于当前用户；`pack`只创建全新的目标目录，保留已有目标。下面两轮使用同一份上游输入：

```sh
mkdir -p build/jdk-trim/input build/jdk-trim/output
./tools/build/cwd-just.sh trim-jdk darwin-arm64 build/jdk-trim/input/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz build/jdk-trim/output/first
./tools/build/cwd-just.sh trim-jdk darwin-arm64 build/jdk-trim/input/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz build/jdk-trim/output/second
```

`trim-jdk`使用登记Go构建CLI，依次执行`pack`和`verify`。CLI本身的等价入口如下，三个宿主仅更改`--host`和对应源归档：

```sh
build/jdk-trim/jdk-trim pack --host darwin-arm64 --source build/jdk-trim/input/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz --dest build/jdk-trim/output/first
build/jdk-trim/jdk-trim verify --host darwin-arm64 --source build/jdk-trim/input/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz --dest build/jdk-trim/output/first
./tools/build/cwd-just.sh test-jdk-trim
```

每次裁剪均自动生成规范化的`tar.gz`、不超过90MiB的分片和动态`report.json`。报告记录输入摘要、裁剪配置、完整输出摘要、裁剪前后载荷大小、删除条目及每卷大小、摘要；它是本次生成的报告，不是构建配置。归档路径排序，时间戳固定为Unix epoch，UID／GID固定为0，普通文件为0644、可执行文件及目录为0755，链接转换为目标原字节的独立普通文件，全部许可证随包保留。压缩实现使用登记Go1.26.8。`verify`从固定源重新计算完整压缩字节，并核对归档、全部分片、报告和目录文件集合；输入、生成时间或目录位置以外的信息不参与输出。

生成使用工作根`build/`下的独立数据缓存，完成或失败后清理；发布候选与目标处于同一父目录，通过macOS的`RENAME_EXCL`或Linux的`RENAME_NOREPLACE`原子发布，已有及并发目标不被覆盖。取消、空间低于50GiB或实际读写失败均返回非零。新对象按0644／0755创建；既有模式差异只警告。

## 制品登记

三个裁剪归档各只需一卷，压缩总量为225043301字节，相比完整上游包的613187926字节减少63.3%。安装组件目录为`jdk21trim1/`，旧的`jdk21/`目录保留。子仓的类型化`internal/catalog/jdk.go`登记完整输出摘要、分片和裁剪标记，并实际核验`java`、`javac`、`jar`及`keytool`的宿主架构、版本和调用结果。

新增或更改裁剪策略时，同行维护类型化策略、行为测试、固定输入、完整自动生成命令和重放核验；不从人工删除后的目录打包。LLVM22、NDK29、SDK37、SDKApp37与AOSP15的源驱动裁剪使用相邻的[trim-pack](../trim-pack/README.md)，从固定原始输入重放文件规则、宿主切片、ELF变换和元数据；安装摘要检查或归档别名转换不能代替完整裁剪的自动生成。

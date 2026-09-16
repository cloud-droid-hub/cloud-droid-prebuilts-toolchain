# AOSP15接口生成工具

2026-09-16继续处理Pixel6-15设备问题#51中的构建工具下载。三个宿主新增`aosp15/`，提供固定`android-15.0.0_r36`的AIDL与HIDL生成工具。目录清单测试先因只存在七个组件而退出1，原始输出分别为三个`*-red.log`；登记新组件后，三个模块的目录清单测试及十三项拒绝子项均退出0，`go vet ./...`均退出0。

| 宿主 | 上游目录 | 压缩字节 | 二进制处理 |
| --- | --- | ---: | --- |
| macOS arm64 | `darwin-x86` | 2317375 | 保留三个通用Mach-O文件中的arm64部分 |
| Linux arm64 | `linux_musl-arm64` | 3893445 | 四份工具和运行库保持原始字节 |
| Linux amd64 | `linux-x86` | 3462954 | 三份工具和运行库保持原始字节 |

上游为`https://android.googlesource.com/platform/prebuilts/build-tools`的固定发布标签。macOS复用已下载且匹配原登记SHA-256的工具，用主仓登记LLVM22的`llvm-lipo -thin arm64`生成独立普通文件；Linux从相同标签读取登记文件并核对原摘要。归档根为`aosp15/`，Gzip与TAR时间戳归零，目录0755、工具和动态库0755、元数据与许可证0644，UID／GID归零；无硬链接或符号链接。每份归档为一个普通Git分片，大小、归档摘要、三个或四个二进制文件摘要及来源文件由对应`internal/catalog/aosp.go`发布。

每个包保留九份许可证与通知：AIDL、libbase、liblog、libhidl、libc++、libc++abi的许可和通知、musl及build-tools的GPL3文本。来源均为相同安卓15标签；已有设备子仓提供的原文件以独立副本保存，缺少的许可文本从固定上游读取。一次许可文本获取因TLS连接提前结束而退出1，保留已取得文件后只补取缺项，第二次退出0。`tools/aosp15/source.properties`是共同来源元数据，二进制身份同时以已核对的摘要和原生执行确定，不能只凭该文本判断版本。

`TestAOSP15Arc`通过正式安装实现展开新组件并再次检查，然后分别调用真实AIDL与HIDL生成器，从独立测试输入生成两个非空、普通单链接头文件。macOS最终退出0、1.56秒，Linux arm64在已有Ubuntu26.04原生容器中退出0、0.18秒，NUC的Ubuntu24.04 amd64退出0、0.13秒，见三个`*-native-final.log`。容器禁网，不使用其它架构程序；这些生成输入只用于宿主工具验证，不是设备源码或Guest验收。

最初两轮macOS和Linux arm64测试分别因测试输入没有放入包路径、没有传入AIDL导入根而退出1，见`*-native.log`和`*-native-v2.log`。修正测试输入和参数后成功；没有为此修改生成器。NUC前两轮被安全目录检查拒绝：手动准备目录受该机umask影响为0775，规范本次候选目录后，Go测试的`001`临时目录仍为0775。测试现显式将自己创建的安装父目录设为0700，保留生产权限检查；失败见`linux-amd64-native-v3.log`和`linux-amd64-native-v4.log`。

macOS候选CLI已在正式`output/toolchains/darwin-arm64/`安装新组件，原有七个组件全部复用，新组件安装退出0，见`darwin-arm64-install.log`。NUC候选CLI同样在正式`output/toolchains/linux-amd64/`复用七个既有组件，完成新增组件安装及全部八项检查，退出0，见`linux-amd64-install.log`。完整Codec2构建、设备调用方迁移、`dist-device`、开发板及Guest未在本项执行；LLVM19归档继续保留。

三个安装程序最终使用登记Go1.26.8、`CGO_ENABLED=0`、`-trimpath`、`-buildvcs=false`和`-ldflags='-s -w'`重建，格式分别为Mach-O arm64、静态ELF aarch64与静态ELF x86-64，见`installer-format.log`。最终macOS安装器与NUC安装器对正式八组件目录的`check`均退出0；macOS结果见`darwin-arm64-final-check.log`，NUC结果随最终原生测试日志保存。

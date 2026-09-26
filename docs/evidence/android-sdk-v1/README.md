# Android SDK组件

2026-09-26按主仓[Android构建工具裁剪并入三平台预编译工具链设计及实施计划](https://github.com/cloud-droid-hub/cloud-droid-hub/blob/main/docs/plans/2026-09-26-Android构建工具裁剪并入三平台预编译工具链设计及实施计划.md)，三个宿主新增`android-sdk/`组件：Build Tools 37.0.0与Platform `android-36` r2的裁剪内容，保持标准SDK根布局，供主项目把`ANDROID_HOME`指向该目录。

## 来源

| 包 | 归档 | 大小 | SHA-1（官方登记） |
| --- | --- | ---: | --- |
| `build-tools;37.0.0` | `build-tools_r37_macosx.zip` | 81933386 | `eb080751b2b2028eb3604f571027d6f7b3c46321` |
| `build-tools;37.0.0` | `build-tools_r37_linux.zip` | 66135704 | `70954e99f4c3d9d46ee70fa32624672fe7cd6ebe` |
| `platforms;android-36` | `platform-36_r02.zip` | 65878410 | `2c1a80dd4d9f7d0e6dd336ec603d9b5c55a6f576` |

三份官方归档从`https://dl.google.com/android/repository/`取得，SHA-1与`repository2-1.xml`一致，另记SHA-256。Google不发布Linux arm64原生build-tools；该宿主`aapt2`、`aidl`、`zipalign`取主仓已登记的Commit451 `android-build-tools-37.0.0-linux-arm64-20260618.tar.xz`（3513876字节，SHA-256 `29b1fa19…fddc`）原始字节，三个程序SHA-256与主仓原登记逐一相同；其`MANIFEST.txt`记录由`android-17.0.0_r1`源码在Linux aarch64构建，保存为`build-tools/37.0.0/ARM64-MANIFEST.txt`。原始输出见[sources.log](sources.log)。

## 裁剪与打包

每个宿主只保留`aapt2`、`aidl`、`zipalign`、`d8`、`apksigner`、`lib/d8.jar`、`lib/apksigner.jar`、`source.properties`、`runtime.properties`、`NOTICE.txt`，Platform的`android.jar`、`source.properties`与`data/NOTICE.txt`；macOS另含`lib64/libc++.dylib`，Linux amd64另含`lib64/libc++.so`（其`aidl`、`zipalign`按`$ORIGIN/lib64`加载）。

- macOS：三个原生程序与`libc++.dylib`用主仓登记LLVM22的`llvm-lipo -thin arm64`取官方通用程序的arm64部分，得到Mach-O thin arm64。
- Linux amd64：保持官方字节。
- Linux arm64：原生程序依赖宿主`libc.so.6`、`libm.so.6`、`libstdc++.so.6`、`libgcc_s.so.1`与`libz.so.1`，由Ubuntu 26.04基础镜像提供。
- 三个宿主的`d8`、`apksigner`、两个jar、`android.jar`字节相同。

归档在`ubuntu:26.04`容器（`--network=none`）内以GNU tar `--sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner`和`gzip -n -9`生成，目录0755、可执行文件与运行库0755、其余0644，无链接。三份归档分别为46422921、47373833、46778641字节，各一个普通Git分片。

## 测试

- 红灯：三个模块先修改`TestCatalog`要求第8项为`Android SDK`并新增`TestAndroidSDK`，改动前分别以`archives=7`与`Android SDK与JDK归档数=1`退出1（`*-red.log`）。
- `TestAndroidSDK`用正式安装实现把JDK 21与`android-sdk`展开到测试目录并执行`Check`，再只用这两个组件：`aidl`生成Java接口，`aapt2 compile/link`以`android.jar`生成资源与APK，`javac`与`d8`生成DEX，`zipalign -f 4`与`-c 4`对齐并校验，`keytool`生成测试密钥，`apksigner sign/verify`签名验证，最后`aapt2 dump badging`回读包名。结果：macOS arm64本机通过（11.81秒），Ubuntu 26.04 arm64原生容器通过（6.01秒），NUC的Ubuntu 24.04 x86_64原生通过（3.61秒），M4的linux/amd64仿真容器同样通过（17.13秒）。
- `go vet ./...`在三个模块均退出0；`cmd`、`internal/catalog`、`internal/install`（Linux另有`internal/selectcfg`）三宿主均退出0（`*-full.log`）。
- 三个`tests/payload_test.go`把固定归档数由7改为8。完整`tests`包未得到有效通过结论：macOS整包在默认10分钟上限超时；聚焦运行中`TestNDK29`、`TestBazel`、`TestRust197`（以及linux-arm64的`TestLLVM22`）按删除LLVM 19前的下标取组件而失败，下标0—6的内容本项未改变，失败路径不经过新增的第8项，属于`3086c46`删除LLVM 19后遗留的测试下标问题，未在本项修改；linux-arm64容器挂载macOS不区分大小写卷时NDK出现`xt_CONNMARK.h`与`xt_connmark.h`同节点而无法展开，属于测试环境限制；linux/amd64仿真容器整包运行被人工停止。见`darwin-arm64-focus.log`与`*-full.log`。

## 安装

三个`bin/cdh-toolchain`以Go 1.26.8、`CGO_ENABLED=0`、`-trimpath`、`-buildvcs=false`、`-ldflags='-s -w'`重建，分别为Mach-O arm64与静态ELF aarch64、x86-64（[installer-format.log](installer-format.log)）。新macOS安装器对主仓正式`output/toolchains/darwin-arm64/`执行`install`：七个既有组件`skipped`，Android SDK `installed`，退出0；随后`check`退出0（`darwin-arm64-install.log`、`darwin-arm64-check.log`）。主仓配方接入、`dist-device gsi-16`与Linux宿主正式安装由主仓另行记录。

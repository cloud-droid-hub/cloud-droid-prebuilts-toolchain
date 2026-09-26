# AGENTS.md

`cloud-droid-prebuilts-toolchain`保存`cloud-droid-hub`本地与Docker构建使用的宿主原生工具链。远端固定为私有`https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain.git`。

## 分支与目录

- 仓库只在`main`分支提交和推送，不创建或维护其它长期分支。
- 三个顶层宿主目录固定为`darwin-arm64/`、`linux-arm64/`和`linux-amd64/`。每个目录只发布与目录名一致的宿主可执行文件。
- macOS只允许arm64宿主。Linux同时保留arm64与amd64原生构建工具；Linux amd64不表示支持运行云机。
- 不得提交跨宿主架构编译器，不得在安装或验证失败时切换到系统编译器。
- 主仓接入时优先复用本机已有Git对象和工作树，不得为了重新组织目录重复下载大归档。
- 本仓只维护根目录这一份规则入口，三个宿主目录共同遵守；平台使用说明保留在各自`README.md`。

## 固定工具链

- Go固定为`1.26.8`，同时保留主项目使用Go 1.27.x验证源码兼容性的能力。
- LLVM 22在macOS使用`22.1.8`，Linux使用Ubuntu 26.04的`22.1.2`。Clang与LLD必须来自同一个LLVM目录。
- 三个宿主目录都不再发布LLVM 19；宿主与Android目标构建统一使用本仓裁剪后的LLVM 22目录，需要旧版本时从Git历史取回，不在本仓重新登记。
- Android NDK固定为r29 revision`29.0.14206865`，NDK内Clang与LLD主版本固定为21。NDK内容只保留Android arm64目标所需的头文件、库、构建元数据、许可证与必要宿主工具。
- Linux arm64没有Google官方宿主包时，只使用本目录LLVM 22与NDK目标sysroot、Clang 21 resource目录，不能运行其它宿主架构的NDK程序。
- JDK固定为Temurin 21 LTS `21.0.12.1+1`。
- Bazel固定为`8.5.1`，三个宿主目录都从本仓归档安装对应原生程序，构建时不得在线下载或改用系统Bazel。
- AOSP15接口生成工具固定为`android-15.0.0_r36`的AIDL与HIDL，统一安装到`aosp15/`。macOS仅保留原始通用程序的arm64部分，Linux沿用对应宿主原始字节；运行库、来源元数据和许可证随包发布。
- Android SDK固定为Build Tools 37.0.0与Platform `android-36` r2，统一安装到`sdk37/`并保持标准SDK根布局，只保留`aapt2`、`aidl`、`zipalign`、`d8`、`apksigner`、两个jar、`android.jar`、`optional/libbinder_ndk_cpp`头文件、运行库、版本元数据与许可证。组件内容变化时改用新的安装目录名，不原地替换已安装目录。macOS仅保留官方通用程序的arm64部分，Linux amd64沿用官方字节；Linux arm64原生程序取已登记的Commit451 `android-arm-build-tools` 37.0.0归档原始字节，其余文件取官方Linux归档。
- Rust、Cargo与Rustdoc固定为`1.97.1`，Rustc内部LLVM固定为`22.1.6`，三个宿主目录都包含`aarch64-linux-android`标准库。构建时必须从本仓`rust197/`安装，不得调用rustup联网安装、改用系统Rust或静默回退。

三个宿主的安装程序和NDK输入分别遵守以下限制；从仓库根调用安装程序时，`--repo`必须指向对应的宿主目录。

| 宿主目录 | `bin/cdh-toolchain`格式 | NDK输入 |
| --- | --- | --- |
| `darwin-arm64/` | Mach-O arm64静态Go程序 | 保留`darwin-x86_64`内部目录名，宿主程序全部为Mach-O arm64 |
| `linux-arm64/` | ELF aarch64静态Go程序 | 仅保留Android arm64目标sysroot、运行库、元数据和Clang 21 resource；编译使用同目录LLVM 22，不能执行amd64程序 |
| `linux-amd64/` | ELF x86-64静态Go程序 | 保留Android arm64目标输入和amd64宿主必需的Clang、LLD 21程序 |

## 归档与安装

- 上游归档或项目裁剪归档按90 MiB上限拆成普通Git对象，不使用Git LFS，也不把GitHub Release附件作为唯一来源。
- 每个归档和分片必须在对应宿主目录的类型化Go目录中登记大小与SHA-256。不得新增JSON、YAML或TOML静态配置。
- `<host>/bin/cdh-toolchain`必须是对应宿主的原生静态Go程序。它从`--repo`指定的宿主目录读取分片，在目标父目录内创建暂存目录，校验后原子发布。
- 安装目录统一为`0755`，普通静态文件统一为`0644`，可执行文件统一为`0755`。许可证与上游通知不得裁掉。
- 禁止使用系统`/tmp`或其它仓外临时目录。构建与验证暂存内容只放本仓`build/`或`output/`。

## 代码与测试

- 三个宿主目录是互相独立的Go模块，最低Go版本为1.26。所有编程语言的自定义类、方法、函数、字段和变量名不超过16个字符，测试函数的`Test`前缀计入长度。
- 先写能观察到失败的Go行为测试并实际取得红灯，再写实现。测试验证输入、输出、退出状态、文件模式、摘要和宿主架构，不读取源码文本断言实现细节。
- Go测试运行时显式把`TMPDIR`设到本仓`build/test-tmp/`。提交前执行三个模块的适用聚焦测试、`go vet`、`git diff --check`和适用宿主的实际安装验证。
- 不创建`.github/workflows/`，本仓不启用GitHub Actions。

## 版本与提交

- 本仓统一版本保存在根目录`version.go`，三个宿主共用，不在宿主目录另设版本文件。使用三段数字；普通修正递增补丁号，功能发布递增次版本并把补丁号归零。
- 版本文件只能在实现、测试、审查、远端同步和提交范围检查完成后修改，并且必须是该提交前最后编辑的文件。修改后立即提交并推送，不再编辑其它文件。
- 一个提交只发布一个工具链、一个安装能力或一个小修正。每项完成后立即推送`main`。
- Git作者与提交者必须是`feicong <fei_cong@hotmail.com>`。
- 使用AI代理时先核验RTK全局Codex配置与遥测关闭状态；项目Shell命令以`rtk`为入口，需要完整或机器输出时使用`rtk proxy`。

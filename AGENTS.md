# AGENTS.md

`cloud-droid-prebuilts-toolchain`保存`cloud-droid-hub`本地与Docker构建使用的宿主原生工具链。远端固定为私有`https://github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain.git`。

## 分支与目录

- 仓库只在`main`分支提交和推送，不创建或维护其它长期分支。
- 三个顶层宿主目录固定为`darwin-arm64/`、`linux-arm64/`和`linux-amd64/`。每个目录只发布与目录名一致的宿主可执行文件。
- macOS只允许arm64宿主。Linux同时保留arm64与amd64原生构建工具；Linux amd64不表示支持运行云机。
- 不得提交跨宿主架构编译器，不得在安装或验证失败时切换到系统编译器。
- 主仓接入时优先复用本机已有Git对象和工作树，不得为了重新组织目录重复下载大归档。

## 固定工具链

- Go固定为`1.26.8`，同时保留主项目使用Go 1.27.x验证源码兼容性的能力。
- LLVM 22在macOS使用`22.1.8`，Linux使用Ubuntu 26.04的`22.1.2`。Clang与LLD必须来自同一个LLVM目录。
- LLVM 19固定为`19.1.7`，用于仍明确要求该版本的宿主构建。
- Android NDK固定为r29 revision`29.0.14206865`，NDK内Clang与LLD主版本固定为21。NDK内容只保留Android arm64目标所需的头文件、库、构建元数据、许可证与必要宿主工具。
- Linux arm64没有Google官方宿主包时，只使用本目录LLVM 22与NDK目标sysroot、Clang 21 resource目录，不能运行其它宿主架构的NDK程序。
- JDK固定为Temurin 21 LTS `21.0.12.1+1`。
- Bazel固定为`8.5.1`，三个宿主目录都从本仓归档安装对应原生程序，构建时不得在线下载或改用系统Bazel。
- Rust、Cargo与Rustdoc固定为`1.97.1`，Rustc内部LLVM固定为`22.1.6`，三个宿主目录都包含`aarch64-linux-android`标准库。构建时必须从本仓`rust197/`安装，不得调用rustup联网安装、改用系统Rust或静默回退。

## 归档与安装

- 上游归档或项目裁剪归档按90 MiB上限拆成普通Git对象，不使用Git LFS，也不把GitHub Release附件作为唯一来源。
- 每个归档和分片必须在对应宿主目录的类型化Go目录中登记大小与SHA-256。不得新增JSON、YAML或TOML静态配置。
- `<host>/bin/cdh-toolchain`必须是对应宿主的原生静态Go程序。它从`--repo`指定的宿主目录读取分片，在目标父目录内创建暂存目录，校验后原子发布。
- 安装目录统一为`0755`，普通静态文件统一为`0644`，可执行文件统一为`0755`。许可证与上游通知不得裁掉。
- 禁止使用系统`/tmp`或其它仓外临时目录。构建与验证暂存内容只放本仓`build/`或`output/`。

## 代码与测试

- 三个宿主目录是互相独立的Go模块，最低Go版本为1.26。新增函数和方法标识符最长16个ASCII字符，测试函数的`Test`前缀计入长度。
- 先写能观察到失败的Go行为测试并实际取得红灯，再写实现。测试验证输入、输出、退出状态、文件模式、摘要和宿主架构，不读取源码文本断言实现细节。
- Go测试运行时显式把`TMPDIR`设到本仓`build/test-tmp/`。提交前执行三个模块的适用聚焦测试、`go vet`、`git diff --check`和适用宿主的实际安装验证。
- 不创建`.github/workflows/`，本仓不启用GitHub Actions。

## 版本与提交

- 每个平台版本保存在`<host>/internal/version/version.go`，使用三段数字。普通修正递增补丁号；功能发布递增次版本并把补丁号归零。
- 版本文件只能在实现、测试、审查、远端同步和提交范围检查完成后修改，并且必须是该提交前最后编辑的文件。修改后立即提交并推送，不再编辑其它文件。
- 一个提交只发布一个工具链、一个安装能力或一个小修正。每项完成后立即推送`main`。
- Git作者与提交者必须是`feicong <fei_cong@hotmail.com>`。
- 使用AI代理时先核验RTK全局Codex配置与遥测关闭状态；项目Shell命令以`rtk`为入口，需要完整或机器输出时使用`rtk proxy`。

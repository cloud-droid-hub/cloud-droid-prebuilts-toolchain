# LLVM 19 与 macOS LLVM 22 归档裁剪设计及实施计划

> 2026-09-16 执行范围：用户授权一个子代理准备三个平台的 LLVM 22 裁剪候选，按 macOS arm64、Linux arm64、Linux amd64 顺序逐项替换、验证和提交。Pixel6 的 macOS LLVM 22 正式构建与测试已通过；其它登记设备的安全分支尚未完成同项验证，因此 LLVM 19 保持原样。

## 目标与边界

减少工具链首次同步和安装量。本计划对应工具链仓 issue #4，不改变工具链版本、宿主架构或项目编译目标。当前先准备 `darwin-arm64` 的 LLVM 22.1.8，以及 `linux-arm64`、`linux-amd64` 的 LLVM 22.1.2 候选；每个平台独立验证、提交并推送 `main`，随后立即同步主仓 gitlink。Linux LLVM 22 已裁过一次，只有明确未使用的内容才继续删减，不以固定缩减比例牺牲能力。

原归档基线：LLVM 19 的 macOS arm64 与 Linux arm64 各约 1.3 GiB、15 卷，Linux amd64 约 1.5 GiB、18 卷；macOS LLVM 22 为1479903744字节、16卷。Linux LLVM 22 已是单卷，不能把它的目录清单直接用于 macOS。最初 macOS `secure_env` 固定 LLVM 19.1.7 并由 Bazel 下载官方包；Pixel6 安全组件本轮已改为显式选择本地 LLVM 22，Linux仍使用22.1.2。归档裁剪与消费者工具链选择分别验收。

## 保留原则

- 按主仓实际调用路径列出原生 Clang、Clang++、同版本 LLD、`llvm-ar`、读取/剥离/归档等宿主辅助工具。保留 Clang resource 目录、编译链接所需库及其传递依赖；macOS LLVM 22 还要保留实际需要的 `libclang` 动态库及加载依赖。
- 保留各归档上游许可证和已有第三方通知；`catalog` 中登记新归档和每卷的实际字节数、SHA-256，任何卷不超过 90 MiB。普通文件独立 inode 且 `st_nlink == 1`，不生成或恢复硬链接；符号链接必须留在包内。
- 不通过系统、Homebrew、APT 或其它架构的工具静默补足裁掉的内容。原始归档作为只读输入保留在 Git 历史；新归档在仓内暂存、验证，再替换该项当前分卷，不碰其它工具链与用户工作区。

## 实施顺序与可观察条件

1. 盘点三个 LLVM 22 包的实际文件、执行文件依赖和主仓消费者，记下原始压缩字节、分卷数与候选清单。检查当前宿主测试和安装能力，记录无法原生运行的宿主验证路径。
2. 先在对应宿主 Go 模块加入制品行为测试：经正式安装后，身份、架构、权限、文件身份、C/C++ 编译链接、归档/读取等所需命令必须通过；macOS LLVM 22 的压缩总量与分卷数目标为较基线至少减半，两个 Linux 包按可证明未使用的内容记录实际缩减量。每项生产归档更改前运行并记录适用的预期红灯，不用源码文本断言。
3. 一个子代理在仓内独立目录准备三个 LLVM 22 候选，主代理完成差异复核后逐个平台替换。每项只改其归档、类型化目录、聚焦测试、许可证记录和必要文档，运行 Go 测试、`go vet`、安装器 `check`、原生编译链接与受影响的主仓入口；版本文件在该项其余编辑和审查之后最后修改并立即提交推送。
4. 每次子仓 push 后，下一写操作仅同步主仓 gitlink，验证主仓从空对象缓存可获取登记对象，再按主仓版本和提交规则独立 push。跨宿主测试不能用 macOS 执行 Linux ELF 冒充原生验证；Linux amd64 可使用同架构 Docker，Linux arm64 使用同架构 Docker 或登记验收机。
5. macOS `secure_env` 的 LLVM 22 正式入口验证完成后，再确定 macOS LLVM 19 是否取消以及两个 Linux LLVM 19 包的处理范围。已授权归档全部完成后复核 `just toolchain-init`、`toolchain-check`、M1 Max `just build-qemu-cdh` 的正式命令与实际结果，向 issue #4 写入压缩前后大小、安装和编译证据，再关闭。任何构建失败只修正该失败所属项，不放宽工具链身份要求。

## 已取得的本机证据

新增 `TestLLVM22Build`，可通过 `CDH_LLVM22_TEST_ROOT` 指定正式安装根；未指定时由安装器展开已登记归档。M1 Max 上使用现有 LLVM 22.1.8 安装运行通过：C/C++ 编译、同目录 LLD 链接、字符串与异常运行、Mach-O arm64 检查、归档与符号读取、剥离后运行、`libclang` 实际动态加载均退出 0。三个宿主 Go 模块的 `./cmd/... ./internal/...` 测试与 `go vet ./...` 在当前 macOS 上通过；这不表示 Linux ELF 已经原生运行。

Pixel6 的正式 `just build-secure-env` 与 `just test-secure-env` 已在完整本地 LLVM 22 安装上退出0，六项原生测试全部通过，产物仅使用系统动态依赖；这一结果由安全组件的本地工具链适配取得，不代表裁剪包已经跑过上述正式入口。体积测试 `CDH_LLVM19_ENFORCE_SIZE=1` 对原 macOS LLVM 19 归档报红，实际为15卷、1413914424字节；它只记录既有基线，不触发本轮 LLVM 19 替换。

## macOS LLVM 22 归档裁剪记录

新增 `TestLLVM22Size` 后，原归档在 `go test ./tests -run '^TestLLVM22Size$' -count=1 -v` 中按预期退出1，报告16卷、1479903744字节。测试读取实际分卷，核对普通文件权限、`st_nlink == 1`、各卷和整包SHA-256，再检查总量与卷数上限。

候选保留完整头文件、Clang resource、上游许可证、`libclang`与`libclang-cpp`、`libc++`、`libc++abi`、`libunwind`以及[平台说明](../../darwin-arm64/README.md)列出的35个工具和别名。裁剪后普通文件总量为1310315208字节，8551个普通文件、20个包内符号链接；保留的原始文件逐字节一致，另加入完整LLVM许可证的独立副本。新压缩包为195159848字节、3卷，较原包减少86.8%。完整SHA-256为`71aadf690e9878fa5f6dfcc87b16cdb7d904e8c12fa26c093e2149ee9d96768c`。

三卷通过普通独立复制进入正式目录，源与目标的inode不同、普通文件链接数均为1，各卷大小和SHA-256与候选一致。原16卷仅从当前工作树替换，其Git历史和已有完整安装保留。使用 `TMPDIR=build/test-tmp`、登记Go 1.26.8和仓内缓存，在macOS模块运行 `go test ./tests -run '^TestLLVM22(Size|Build)$' -count=1 -v` 退出0：体积测试报告3卷、195159848字节，原生构建测试从新正式登记归档全新安装并通过 `check`、C/C++运行和实际动态加载，全程155.235秒。三个模块的 `go test ./cmd/... ./internal/...` 与 `go vet ./...`、组件仓 `git diff --check` 均退出0。

对应日志保存在主仓 `output/llvm22-trim/logs/` 下的 `darwin-size-red.log`、`darwin-formal-test.log`、`darwin-formal-copy.log`、`darwin-formal-compare.log`及各平台 `*-formal-unit.log`、`*-formal-vet.log`。主代理负责版本、逐项提交、同步主仓引用，以及从新正式包重装后执行 `toolchain-init`、`toolchain-check`和 `build-qemu-cdh`。

## 回退

候选归档或原生构建失败时不推送该项；保留上一个已验证分卷和目录。已推送的缺陷通过新的前向小提交修复，不重写 `main` 或移动 tag。M1 Max 现有安装和源码不由本计划直接覆盖，待它自身安装结束后使用 Git 同步。

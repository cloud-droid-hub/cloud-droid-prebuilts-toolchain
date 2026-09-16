# LLVM19与macOS LLVM22归档裁剪设计及实施计划

> 2026-09-16执行范围：用户授权一个子代理准备三个平台的LLVM22裁剪候选，按macOS arm64、Linux arm64、Linux amd64顺序逐项替换、验证和提交。Pixel6的macOS LLVM22正式构建与测试已通过；其它登记设备的安全分支尚未完成同项验证，因此LLVM19保持原样。新增候选安装检查工具供原生验证使用，工具提交不代表归档发布或消费者迁移完成。

## 目标与边界

减少工具链首次同步和安装量。本计划对应工具链仓issue #4，不改变工具链版本、宿主架构或项目编译目标。当前先准备`darwin-arm64`的LLVM22.1.8，以及`linux-arm64`、`linux-amd64`的LLVM22.1.2候选；每个平台独立验证、提交并推送`main`，随后立即同步主仓引用。Linux LLVM22已裁过一次，只有明确未使用的内容才继续删减，不以固定缩减比例牺牲能力。

原归档基线：LLVM19的macOS arm64与Linux arm64各约1.3 GiB、15卷，Linux amd64约1.5 GiB、18卷；macOS LLVM22为1479903744字节、16卷。Linux LLVM22已是单卷，不能把它的目录清单直接用于macOS。最初macOS `secure_env`固定LLVM19.1.7并由Bazel下载官方包；Pixel6安全组件本轮已改为显式选择本地LLVM22，Linux仍使用22.1.2。归档裁剪与消费者工具链选择分别验收。

## 保留原则

- 按主仓实际调用路径列出原生Clang、Clang++、同版本LLD、`llvm-ar`、读取/剥离/归档等宿主辅助工具。保留Clang resource目录、编译链接所需库及其传递依赖；macOS LLVM22还要保留实际需要的`libclang`动态库及加载依赖。
- 保留各归档上游许可证和已有第三方通知；`catalog`中登记新归档和每卷的实际字节数、SHA-256，任何卷不超过90 MiB。普通文件独立inode且`st_nlink == 1`，不生成或恢复硬链接；按最新规则，仅允许登记的`bin/clang -> clang-22`与`bin/ld.lld -> lld`符号链接，其余别名转为独立普通文件。
- 不通过系统、Homebrew、APT或其它架构的工具静默补足裁掉的内容。原始归档作为只读输入保留在Git历史；新归档在仓内暂存、验证，再替换该项当前分卷，不碰其它工具链与用户工作区。

## 实施顺序与可观察条件

1. 盘点三个LLVM22包的实际文件、执行文件依赖和主仓消费者，记下原始压缩字节、分卷数与候选清单。检查当前宿主测试和安装能力，记录无法原生运行的宿主验证路径。
2. 先在对应宿主Go模块加入制品行为测试：经正式安装后，身份、架构、权限、文件身份、C/C++编译链接、归档/读取等所需命令必须通过；macOS LLVM22的压缩总量与分卷数目标为较基线至少减半，两个Linux包按可证明未使用的内容记录实际缩减量。每项生产归档更改前运行并记录适用的预期红灯，不用源码文本断言。
3. 一个子代理在仓内独立目录准备三个LLVM22候选，主代理完成差异复核后逐个平台替换。每项只改其归档、类型化目录、聚焦测试、许可证记录和必要文档，运行Go测试、`go vet`、安装器`check`、原生编译链接与受影响的主仓入口；版本文件在该项其余编辑和审查之后最后修改并立即提交推送。
4. 每次子仓push后，下一写操作仅同步主仓引用，验证主仓从空对象缓存可获取登记对象，再按主仓版本和提交规则独立push。跨宿主测试不能用macOS执行Linux ELF冒充原生验证；Linux amd64可使用同架构Docker或已登记NUC，Linux arm64使用同架构Docker或登记验收机。
5. macOS `secure_env`的LLVM22正式入口验证完成后，再确定macOS LLVM19是否取消以及两个Linux LLVM19包的处理范围。已授权归档全部完成后复核`just toolchain-init`、`toolchain-check`、M1 Max `just build-qemu-cdh`的正式命令与实际结果，向issue #4写入压缩前后大小、安装和编译证据，再关闭。任何构建失败只修正该失败所属项，不放宽工具链身份要求。

## 已取得的本机证据

新增`TestLLVM22Build`，可通过`CDH_LLVM22_TEST_ROOT`指定正式安装根；未指定时由安装器展开已登记归档。M1 Max上使用现有LLVM22.1.8安装运行通过：C/C++编译、同目录LLD链接、字符串与异常运行、Mach-O arm64检查、归档与符号读取、剥离后运行、`libclang`实际动态加载均退出0。三个宿主Go模块的`./cmd/... ./internal/...`测试与`go vet ./...`在当前macOS上通过；这不表示Linux ELF已经原生运行。

Pixel6的正式`just build-secure-env`与`just test-secure-env`已在完整本地LLVM22安装上退出0，六项原生测试全部通过，产物仅使用系统动态依赖；这一结果由安全组件的本地工具链适配取得，不代表裁剪包已经跑过上述正式入口。体积测试`CDH_LLVM19_ENFORCE_SIZE=1`对原macOS LLVM19归档报红，实际为15卷、1413914424字节；它只记录既有基线，不触发本轮LLVM19替换。

## macOS LLVM22归档裁剪记录

新增`TestLLVM22Size`后，原归档在`go test ./tests -run '^TestLLVM22Size$' -count=1 -v`中按预期退出1，报告16卷、1479903744字节。测试读取实际分卷，核对普通文件权限、`st_nlink == 1`、各卷和整包SHA-256，再检查总量与卷数上限。

此前候选保留完整头文件、Clang resource、上游许可证、`libclang`与`libclang-cpp`、`libc++`、`libc++abi`、`libunwind`以及[平台说明](../../darwin-arm64/README.md)列出的35个工具和别名。裁剪后普通文件总量为1310315208字节，8551个普通文件、20个包内符号链接；保留的原始文件逐字节一致，另加入完整LLVM许可证的独立副本。该压缩包为195159848字节、3卷，较原包减少86.8%。完整SHA-256为`71aadf690e9878fa5f6dfcc87b16cdb7d904e8c12fa26c093e2149ee9d96768c`。这些是旧候选的历史证据，后续按最新链接规则重新生成的普通文件候选须独立验收。

三卷通过普通独立复制进入正式目录，源与目标的inode不同、普通文件链接数均为1，各卷大小和SHA-256与候选一致。原16卷仅从当时工作树替换，其Git历史和已有完整安装保留。使用`TMPDIR=build/test-tmp`、登记Go1.26.8和仓内缓存，在macOS模块运行`go test ./tests -run '^TestLLVM22(Size|Build)$' -count=1 -v`退出0：体积测试报告3卷、195159848字节，原生构建测试从新正式登记归档全新安装并通过`check`、C/C++运行和实际动态加载，全程155.235秒。三个模块的`go test ./cmd/... ./internal/...`与`go vet ./...`、组件仓`git diff --check`均退出0。

对应日志保存在主仓`output/llvm22-trim/logs/`下的`darwin-size-red.log`、`darwin-formal-test.log`、`darwin-formal-copy.log`、`darwin-formal-compare.log`及各平台`*-formal-unit.log`、`*-formal-vet.log`。主代理负责版本、逐项提交、同步主仓引用，以及从新正式包重装后执行`toolchain-init`、`toolchain-check`和`build-qemu-cdh`。

## macOS LLVM22普通文件归档记录

按最新链接规则，保留`bin/clang -> clang-22`和`bin/ld.lld -> lld`，其余18个工具与动态库别名均保存为最终目标的独立普通文件。普通文件增加900946544字节，合计2211261752字节、8569个文件；归档仍含9281个条目，工具入口、头文件、Clang resource、库与许可证清单不变。新压缩包为351706968字节、4卷，比此前三卷包增加156547120字节，仍比原始16卷包减少76.2%；完整SHA-256为`76c2be3c57ca741a095a70ccb739148ea57e13033f6c1d39e9b342aac64c2db2`。

替换前，`go test ./tests -run '^TestLLVM22Links$' -count=1 -v`实际退出1，旧正式归档的`bin/clang++ -> clang`被拒绝。使用现有完整安装运行`TestLLVM22Copy`也退出1，仅通过`Lstat`观察到`bin/clang++`不是普通文件，未执行该旧别名。两项红灯未创建链接。

本次重新计算旧包、新候选和四个分卷的SHA-256，与此前完整归档验证的输入一致，因此复用`output/llvm22-trim/regular-v2/logs/darwin-arm64-verify.log`中路径、权限、最终目标字节与许可证保持一致的结果，没有再次运行该完整归档验证。使用已提交的`tools/llvm22-check`从候选四卷全新安装，实际退出0、耗时472.086秒：正式安装器`RunTo`与`Check`、Clang/LLD22.1.8、C/C++字符串与异常运行、静态库链接、LLVM工具、剥离后运行及`libclang.dylib`实际加载均通过。安装和探测目录位于组件`build/llvm22-trim/darwin-regular-v2/`，现有主仓完整LLVM22安装保留。

四卷经普通复制、同步、SHA-256及独立文件身份核对后，以macOS原子交换进入`darwin-arm64/payload/llvm22/`，旧三卷保存在组件`build/llvm22-trim/darwin-regular-v2/payload-swap/`。相应类型化catalog和原生安装程序已重建，旧安装程序保留在同一构建目录。源与目标inode不同、普通文件`st_nlink == 1`，未创建额外链接。新安装中的完整LLVM许可证和`include/llvm/Support/LICENSE.TXT`与既有许可证逐项摘要一致。

在登记Go1.26.8、组件`build/test-tmp/`和仓内缓存下，`go test ./tests -run '^TestLLVM22(Size|Links|Build)$' -count=1 -v`使用候选安装根运行，三项测试退出0、全程222.053秒，确认正式四卷摘要、8569个普通文件、两项登记链接以及18组别名与目标的权限、字节和独立身份。随后清空`CDH_LLVM22_TEST_ROOT`，单独运行`go test ./tests -run '^TestLLVM22Build$' -count=1 -v`，由正式catalog从新分卷全新安装单项LLVM22，安装检查及原生探测退出0、全程249.928秒。三个宿主模块的`go test ./internal/catalog -count=1 -v`与`go vet ./...`在macOS上退出0；这些Go检查不代表运行了Linux程序。

本次原始日志保存在主仓`output/llvm22-trim/darwin-regular-formal/`：`links-red.log`、`copy-red.log`记录替换前行为，`native.log`记录候选全新安装及原生探测，`focused.log`与`formal-install.log`分别记录聚焦检查和正式分卷的全新安装，`input-sha256.log`、`licenses.log`、`formal-ids.log`、`formal-sha256.log`及`previous-ids.log`、`previous-sha256.log`记录输入、正式文件和保留旧文件。该项不修改LLVM19、两份Linux归档或版本；主仓正式消费者验证、版本与发布仍由主代理继续执行。

## Linux arm64 LLVM22普通文件归档记录

Linux arm64保留LLVM22.1.2、35个工具入口、已有Clang resource、库与CMake元数据，以及全部27份许可证和来源说明。裁掉58个未使用的分析或辅助工具后，45个工具及库别名转为最终目标字节的普通文件，只有`bin/ld.lld -> lld`保留登记链接；`clang`本来就是普通文件。LLVM19、macOS和Linux amd64分卷均不在本项修改范围。

外层`lib/libclang-22.so.1`在原包和此前普通文件候选中均因缺少`libz3.so.4`加载失败；完整`TestLLVM22`还识别出`libffi`、`libedit`与`libxml2`从宿主解析。保留该测试的全部ELF依赖检查与宿主库允许列表，按原库`$ORIGIN`路径补入[平台说明](../../linux-arm64/README.md#llvm22归档范围)登记的四份普通副本，来源均为原包`lib/cdh`内实际文件，不改库字节、权限或许可证。新增展开字节19544620，四组源和目标均为独立inode且`st_nlink == 1`。

使用主仓维护的`llvm22-pack`，`pack`与`verify`同时传入四组`dest=source`映射，退出码均为0。严格验证确认所有保留路径、最终目标字节、权限与许可证一致。新归档418个普通文件、1个登记链接、441个条目，普通文件合计966508007字节。压缩后154451176字节、2卷，完整SHA-256为`18374afd0db2a4a868d51f21f7aa1cc2646de41fec4e94b4e8e955575cc17027`。旧正式包为85458612字节、1卷；普通文件别名与补齐依赖使压缩包增加68992564字节，不将此记录为体积缩减。与此前未补四份依赖的普通文件候选相比，压缩包增加5950184字节。

两卷通过普通复制、SHA-256、权限、链接数及源目标inode核对，再以macOS原子交换进入`linux-arm64/payload/llvm22/`。旧单卷保存在组件`build/llvm22-trim/linux-arm64-regular-v1/payload-swap/`。类型化catalog登记新摘要、分卷及四份依赖文件检查，Linux arm64静态Go安装程序已重建；旧安装程序保存在同目录`installer-swap`，供原始制品回读。

在登记Ubuntu26.04 arm64 Docker镜像内，从新正式分片调用已提交的`tools/llvm22-check`，全新安装与原生消费验证退出0，耗时112.089秒。安装根为组件`build/llvm22-trim/linux-arm64-regular-v1/formal-install/llvm22`，探测目录为同级`formal-probe`。正式`install.RunTo/Check`、同目录Clang/LLD22、C/C++字符串与异常运行、静态归档链接、LLVM工具、剥离后运行及真实`lib/cdh`下libclang加载均通过，受检433个普通文件保持独立身份。随后在该安装上运行原`TestLLVM22`及新增工具、副本、路径、C/C++和独立库加载测试，六项顶层测试退出0：35个工具与68个库分别在新进程通过；`LLVMPolly.so`插件的独立加载明确跳过1项，不计为通过，其ELF依赖仍由原测试核对。

`LLVMPolly.so`原包与候选字节相同。早期直接加载缺少LLVM宿主符号，额外用原包`opt -load`加载原包或候选都因重复注册Polly选项失败；本项未证实有登记消费者使用这两种额外加载方式，保留文件及失败日志，不扩展插件用法。此包保持Ubuntu26.04和glibc2.43基线，未声明Ubuntu24.04兼容，也未在该候选上执行主仓完整QEMU、安全组件或设备运行验收。

归档行为测试`TestLLVM22Arc`对旧包退出1，先拒绝前缀外的AppleDouble元数据；对新正式分片退出0，耗时98.234秒，核对两卷真实摘要、418个普通文件、登记链接、14组别名字节和27份许可证。路径检查先在非规范路径及父级链接元数据注入时取得红灯，再补齐完整父目录链检查并退出0；测试未实际创建链接。三个宿主模块各自的`go test ./internal/catalog -count=1 -v`与`go vet ./...`退出0，这些macOS上的Go检查不表示运行了Linux amd64程序。结束时本机可用空间59878624 KiB，高于50 GiB。

原始日志位于主仓`output/llvm22-trim/linux-arm64-formal-v1/`。`outer-old.log`、`outer-red.log`、`deps-red-v2.log`保留既有依赖故障；`layout-red.log`同时记录首次挂载未包含父目录`go.mod`的检查失败，修正挂载后才取得真实依赖红灯。`layout-green.log`虽使用该文件名，实际仍退出1，原因是上述Polly额外加载失败；`polly-old-red.log`还记录原包中的旧链接被普通文件检查拒绝。`polly-old-host.log`与`polly-copy-host.log`保留原包`opt`对照失败。最终依据为`copy-pack.log`、`copy-verify.log`、`copy-parts.log`、`archive-green.log`、`formal-native.log`、`formal-tests.log`、`load-final.log`、`path-red.log`、`path-green.log`、`parts-before-ids.log`及`parts-final-ids.log`，未覆盖早期失败。

## Linux amd64 LLVM22普通文件归档记录

2026-09-16在Linux arm64归档发布后继续准备及验证amd64归档。旧正式归档为77077396字节、1卷，完整SHA-256为`3f1304acb447b9b6ba061e911a988b9915cae5398afe55cc1b3530d3f02b0688`。本项更新类型化catalog、测试、说明、正式分片与静态安装程序，LLVM19、macOS和Linux arm64不在本项修改范围。

保留35个工具入口、Clang resource、已有库与CMake元数据，以及28份许可证和来源说明。42个原始别名转为独立普通文件，只有登记的`bin/ld.lld -> lld`仍为符号链接。静态核对发现外层LLVM与libclang使用`DT_RPATH=$ORIGIN`，转换后的普通文件布局缺少libclang要求的外层`libLLVM.so.22.1`，LLVM另需外层`libffi.so.8`、`libedit.so.2`、`libz3.so.4`与`libxml2.so.16`。按[平台映射](../../linux-amd64/README.md#llvm22普通文件归档)从原包`lib/cdh`复制最终文件字节，增加170469316展开字节；不修改动态库、Ubuntu24.04兼容库或许可证内容。

主仓维护的`llvm22-pack pack`与`verify`使用同一组五项映射，均退出0。候选为419个普通文件、1个登记链接、442个条目，普通文件合计815168159字节。压缩后141057060字节，完整SHA-256为`55f4bf7ce313c0f4af2ad36152086686ad3e076b7bbfbc27bb0381118b031517`，两卷分别为94371840与46685220字节。第一卷SHA-256为`effab4b4666e595bc07531a8ae7a831497fa39f3ddadc2bf88f8fa81e7eea667`，第二卷为`5a3f31c8c8fda3c09c2d809a4054f5efdeda8626e94fe784e0eafcf056307945`。与旧正式包相比增加63979664字节，与此前未补五份依赖的普通文件候选相比增加31650520字节，不记为体积缩减。

多分片读取测试先用真实两段XZ数据取得退出1，再修正旧单卷假设；旧正式归档的`bin/clang++ -> clang`也被新归档行为测试拒绝，退出1。使用登记Go1.26.8、组件`build/test-tmp/`与仓内缓存，令`CDH_LLVM22_PARTS`指向新候选分片，运行`go test ./tests -run '^TestLLVM22(Split|Arc|GLIBC)$' -count=1 -v`退出0，耗时180.100秒；核对两卷摘要、普通文件与登记链接、14组别名字节和28份许可证，并保留GLIBC需求不高于2.39的检查。

首次catalog测试因新增149622136字节的`libLLVM.so.22.1`超过既有`FileCheck`单文件90MiB上限而退出1；撤回这个新增字段后复测退出0，没有放宽验证器。该大文件继续由完整归档摘要、严格打包验证与归档/安装副本行为检查覆盖，其余四个依赖副本登记单文件摘要。三个宿主模块的catalog测试和`go vet ./...`退出0，Linux amd64候选安装器、检查工具和原生测试程序以登记Go交叉编译为静态ELF x86-64，随后用检查工具和测试程序在NUC原生验收。

候选保存在主仓`output/llvm22-trim/regular-v3/linux-amd64/`，原始日志保存在`output/llvm22-trim/linux-amd64-formal-v1/`。`split-red.log`、`archive-red.log`与`linux-amd64-catalog.log`保留上述失败，`copy-pack.log`、`copy-verify.log`、`copy-parts.log`、`archive-green.log`与`linux-amd64-catalog-green.log`记录修正后的结果。工具清单盘点首次命令因AWK表达式错误退出2，保留`tools-static-error.log`，修正后的只读盘点在`tools-static-final.log`记录35个入口；这不是工具链运行失败。

NUC结束正式同步并释放已授权缓存后，可用空间54656436KiB，允许继续候选验收。该机为Ubuntu24.04.4、x86_64、glibc2.39。原始正式安装的外层libclang与真实`lib/cdh`入口分别在新进程加载，均实际退出0，日志为`nuc-original-outer.log`与`nuc-original-cdh.log`；不得把静态缺项推断写成原包加载失败。随后`Lstat`确认外层输入为已有符号链接，`nuc-original-deps.log`显示其`libLLVM.so.22.1`、ffi、edit与z3从宿主解析。对照后不再执行旧链接，也不为取得红灯重建旧普通文件安装。

全新安装按816978591字节及67108864字节探测空间预留，两卷另占141057060字节。已提交的`llvm22-check`从这两卷完成全新安装，109.103秒退出0；正式`RunTo/Check`、同目录Clang/LLD22、C/C++字符串与异常运行、静态归档、LLVM工具、剥离后运行及真实`lib/cdh`下libclang加载通过，434个受检普通文件保持独立身份。安装路径为NUC主仓`build/llvm22-amd64-v3/install/llvm22`，探测路径为同级`probe`，原始日志为`nuc-native.log`。

随后以预先构建的静态x86-64测试程序运行`-test.run '^TestLLVM22(GLIBC|Tools|Load|Copies)?$' -test.count=1 -test.v`。35工具、68库独立进程加载和五份普通副本检查通过，GLIBC检查52.06秒通过；`LLVMPolly.so`独立加载明确跳过1项，其ELF依赖继续由原测试检查。聚焦组最后的`TestLLVM22`因NUC检出的`PACKAGES.md`权限0664退出1，日志`nuc-tests.log`完整保留，不能把这一组合命令写成退出0。

按主代理授权，逐文件`Lstat`确认28份许可证为普通单链接后，仅规范该目录文件权限为0644；实际变化为`PACKAGES.md`与`libgnutls-copyright`两项，内容及inode保持不变，证据为`nuc-license-mode.log`。只重跑`-test.run '^TestLLVM22$' -test.count=1 -test.v`，原有全部ELF依赖断言、工具身份和编译链接检查2.29秒退出0，日志`nuc-elf-final.log`；未修改测试以接受0664，也未重跑已通过的52秒GLIBC检查。

`nuc-candidate-deps.log`确认外层候选libclang为独立普通文件且加载退出0，所需LLVM、ffi、edit、z3和xml2名称均解析到候选目录。NUC结束时可用53709820KiB，高于50GiB；当时NUC旧正式单卷摘要仍为原值。Ubuntu26.04 amd64、完整QEMU、安全组件和设备运行验收未执行。

主代理复核后，两卷经普通独立复制、摘要和权限检查，以原子目录交换进入本机正式`linux-amd64/payload/llvm22/`。原单卷保存在组件`build/llvm22-trim/linux-amd64-regular-v1/payload-swap/`；静态x86-64安装程序通过相同身份检查和原子交换发布，旧程序保存在同级`installer-swap`。新程序2945186字节，SHA-256为`6e722d4d527bd47149fbdb2257a901338b008ab05e16ba814f6cb2007b353c71`，原始交换证据为`formal-publish.log`。

清空`CDH_LLVM22_PARTS`后，使用相同登记Go与仓内缓存执行`go test ./tests -run '^TestLLVM22Arc$' -count=1 -v`，从正式目录读取两卷，90.638秒退出0。检查确认141057060压缩字节、419个普通文件、1个登记链接、14组别名字节及28份许可证，证据为`formal-archive.log`。本项未在NUC替换原正式安装，也未把候选原生检查记成主仓完整构建通过。

## 候选安装检查工具

2026-09-16把候选安装与消费者检查维护在[工具源码](../../tools/llvm22-check/README.md)。它复用正式安装器，安装前核对连续分卷、真实摘要、TAR条目、独立文件身份和空间，拒绝未登记链接及错误架构。安装后检查成对编译器、C/C++异常运行、静态归档、二进制工具和libclang真实加载；失败保留候选与日志。该工具不修改正式catalog，不下载依赖，也不替代安全组件正式构建。

输出写入失败的行为红灯已取得，修正后14项工具测试与`go vet ./...`退出0，包含小型真实归档安装、摘要变化拒绝和目录保护。Linux amd64工具以登记Go交叉编译，随后在NUC原生运行。日志位于组件`build/llvm22-check/`下的`log-red.log`、`tests-final.log`、`vet-final.log`与`native-v2.log`。

NUC首次执行因本次临时目录继承`0775`权限被安装器拒绝，退出1，保留`native.log`；把本次目录修正为`0755`后，用新的安装和探测目录重跑。两卷候选合计109406540字节，完整SHA-256为`45ea087c05892ba44f83628339d893dbbe7d37a20556dfa14589ff1734d6094d`。全新安装、原生Clang/LLD22.1.2、C/C++异常运行、静态库链接、二进制读取与剥离后运行、libclang真实加载均退出0，总耗时84.016秒；受检普通文件保持独立身份，结束可用空间52900984 KiB。该候选尚未替换正式分卷，也未执行NUC完整项目构建。

本项不删除LLVM19，不把现有Pixel6构建结果扩展为其它安全分支已完成。三个平台候选的正式消费者入口、逐项归档发布和其它设备安全分支迁移仍须继续验证。

## 回退

候选归档或原生构建失败时不推送该项；保留上一个已验证分卷和目录。已推送的缺陷通过新的前向小提交修复，不重写`main`或移动tag。M1 Max现有安装和源码不由本计划直接覆盖，待它自身安装结束后使用Git同步。

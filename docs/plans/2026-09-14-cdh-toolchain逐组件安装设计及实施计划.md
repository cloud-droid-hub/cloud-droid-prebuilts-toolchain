# cdh-toolchain逐组件安装设计及实施计划

## 设计

### 目标

三个宿主模块的`cdh-toolchain install`都提供稳定、机器可读的逐组件进度。安装被中断、一个组件被删除或多个组件被删除后，再次执行只补齐不存在的组件；已通过组件级校验的目录保持原样。已存在但校验失败的组件和用户额外内容不得被删除或覆盖。

### 行为边界

- 每个组件的进度行固定包含`component`、`index`、`total`、`remaining`、`elapsed_seconds`和`state`。
- 组件开始检查时输出`state=checking`；有效组件输出`state=skipped`。
- 缺失组件输出`state=installing`，安装期间按`CDH_TOOLCHAIN_PROGRESS_INTERVAL`持续输出心跳，完成发布后输出`state=installed`。
- 心跳间隔默认`5s`；显式值必须是大于零的Go duration，非法值在修改目标目录前失败。
- 组件只在同一文件系统的临时目录完成展开、模式修正和组件级校验后发布。发布使用目标不存在语义，不能替换并发出现的目录。
- 目标目录已存在时必须是普通目录。组件路径存在但校验失败时立即失败并保留现场；只有路径不存在才允许自动补齐。
- `Run`只对当前UID所有、与父目录跨设备、组可写但非`world`可写的挂载根放宽父目录限制，通过已打开的目录FD只追加`sticky`位，并保留原有`setuid`、`setgid`和权限位。
- `Check`不修改目录权限；同类挂载根缺少`sticky`位时直接拒绝。非跨设备的普通`0775`目录和任何`0777`目录仍拒绝。
- 目标目录链与组件路径使用`openat`和`O_NOFOLLOW`打开；权限调整、身份核对、暂存源检查与目标发布均使用已打开的目录或文件FD。
- 安装边界文件不存在时创建；已存在时必须内容与权限均有效，否则失败且不重写。
- CLI将`SIGINT`和`SIGTERM`转换为安装上下文取消；中断只清理本轮尚未发布的临时组件，此前已发布组件留在目标目录，下一次执行重新校验并跳过。

### 文件边界

- 三个`internal/install/install.go`：逐组件状态机、组件级校验、进度心跳和目标边界。
- 三个`internal/install/swap_*.go`：增加跨平台不替换发布操作。
- 三个`internal/install/install_test.go`：单删、多删、中断、心跳、非法间隔和不覆盖行为。
- 三个`cmd/cdh-toolchain/main.go`及测试：把CLI标准输出传给安装进度。
- 根`README.md`和三个宿主`README.md`：记录字段、续跑、补齐、失败与用户内容边界。
- 三个`bin/cdh-toolchain`：用固定Go 1.26.8重新构建对应宿主静态程序。
- 三个`internal/version/version.go`：功能版本递增，且作为提交前最后编辑的文件。

## cdh-toolchain逐组件安装 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为三宿主安装程序交付可观察的逐组件心跳、校验后跳过和缺失组件补齐能力。

**Architecture:** 安装器把完整目录交换改为组件级检查与不替换发布。进度器独立负责稳定字段和周期心跳，安装器只提交状态变化；组件目录通过现有探针验证后才进入跳过或发布路径。

**Tech Stack:** Go 1.26.8、标准库、`golang.org/x/sys/unix`、Go行为测试。

## Global Constraints

- 只修改`cloud-droid-prebuilts-toolchain`的`main`。
- 三个宿主模块保持同一安装能力。
- 新增或修改的函数与方法名最长16个ASCII字符，测试函数的`Test`前缀计入长度。
- 测试`TMPDIR`固定在仓库`build/test-tmp/`，不用系统临时目录。
- 不使用force、stash、reset或clean，不覆盖工作区及安装目标中的用户内容。
- 版本文件在实现、测试、审查、远端同步和范围检查后最后编辑，随后立即提交并推送。

---

### Task 1: 逐组件行为红灯

**Files:**
- Modify: `darwin-arm64/internal/install/install_test.go`
- Modify: `linux-arm64/internal/install/install_test.go`
- Modify: `linux-amd64/internal/install/install_test.go`

**Interfaces:**
- Consumes: 当前`Run(ctx, repo, dest, spec)`安装行为。
- Produces: 对`RunTo(ctx, repo, dest, spec, output)`的可观察行为约束。

- [x] **Step 1: 写失败测试**

用真实小归档组成三个组件，测试：

`TestOneDelete`覆盖单个目录删除，`TestManyDelete`覆盖多个目录删除，`TestInterrupt`覆盖第二组件中断后继续，`TestHeartbeat`覆盖安装期间周期输出，`TestKeepInvalid`覆盖已有无效组件与用户内容保留。`TestRootSwap`、`TestCancelEdge`和`TestCLISignal`补充目标根替换、发布边缘取消及真实CLI信号行为。

已验证组件对应的源分片在重跑前移走；重跑仍成功才证明安装器没有重新读取该组件归档。输出断言只检查运行结果，不读取实现文本。

- [x] **Step 2: 取得RED**

Run: `TMPDIR="$PWD/build/test-tmp" go test ./internal/install -run 'Test(OneDelete|ManyDelete|Interrupt|Heartbeat|KeepInvalid)$'`

Expected: FAIL，失败原因是当前整棵目录重装、缺少进度输出或覆盖了无效组件。

### Task 2: 组件级安装与进度

**Files:**
- Modify: `darwin-arm64/internal/install/install.go`
- Modify: `darwin-arm64/internal/install/swap_darwin.go`
- Modify: `darwin-arm64/internal/install/swap_linux.go`
- Modify: `darwin-arm64/cmd/cdh-toolchain/main.go`
- Apply the same changes under: `linux-arm64/`, `linux-amd64/`

**Interfaces:**
- Consumes: `catalog.Spec`和现有归档校验、展开、探针函数。
- Produces: `RunTo(ctx context.Context, repo, dest string, spec catalog.Spec, output io.Writer) error`；保留`Run`作为静默兼容入口。

- [x] **Step 1: 写最小实现**

`RunTo`逐一调用组件校验。存在且有效时跳过，缺失时展开到临时目录并调用`moveNew`发布，存在但无效时返回错误。进度器用互斥写入器和ticker输出：

```text
toolchain_progress component="LLVM 22" index=4 total=7 remaining=3 elapsed_seconds=10 state=installing
```

- [x] **Step 2: 让聚焦测试转绿**

Run: `TMPDIR="$repo/build/test-tmp" go test ./internal/install -run 'Test(OneDelete|ManyDelete|Interrupt|Heartbeat|KeepInvalid)$'`

Expected: PASS。

- [x] **Step 3: 验证CLI输出**

CLI测试通过注入小型目录运行安装，确认标准输出收到逐组件字段，错误仍写入标准错误。

### Task 3: 文档与三宿主同步

**Files:**
- Modify: `README.md`
- Modify: `darwin-arm64/README.md`
- Modify: `linux-arm64/README.md`
- Modify: `linux-amd64/README.md`
- Modify: three host copies of implementation and tests

**Interfaces:**
- Consumes: Task 2确定的字段与失败语义。
- Produces: 三宿主一致源码与面向使用者的续跑说明。

- [x] **Step 1: 同步三宿主源码与测试**

比较三个模块的安装实现和测试摘要，确认平台差异只保留在目录、catalog与CLI select能力。

- [x] **Step 2: 更新安装说明**

README给出默认间隔、环境变量示例、进度样例、单删/多删补齐、中断续跑、无效目录失败和用户额外内容保留边界。

### Task 4: 完整验证、审查与发布

**Files:**
- Modify: `darwin-arm64/bin/cdh-toolchain`
- Modify: `linux-arm64/bin/cdh-toolchain`
- Modify: `linux-amd64/bin/cdh-toolchain`
- Modify last: three `internal/version/version.go`

**Interfaces:**
- Consumes: 已审查并通过测试的三宿主源码。
- Produces: 三个对应架构的静态程序和一个`main`提交。

- [x] **Step 1: 三模块验证**

对每个模块执行`go test ./cmd/... ./internal/...`、`go test -race ./cmd/... ./internal/...`和`go vet ./cmd/... ./internal/...`，并运行`git diff --check`。

- [ ] **Step 2: 适用实际安装验证**

用Darwin原生目录执行安装、中断续跑、单删补齐、多删补齐和最终`check`；测试输出保存在仓库`build/`。Linux二进制执行交叉编译与ELF架构检查。

当前Darwin只读安装现场已观察到六个有效组件依次输出`checking`和`skipped`，且只进入唯一缺失的Rust组件安装；因本工作树按空间约束没有检出Rust payload，命令按预期失败，不能把该证据记为最终补齐成功。单删、多删和中断续跑已由真实小归档行为测试覆盖；完整payload上的最终补齐与`check`留待同步新版二进制后的M1 Max和M4现场验收。Linux arm64与amd64程序已用固定Go 1.26.8交叉构建，并核对为对应架构的静态ELF。

- [x] **Step 3: 独立代码审查**

审查需求覆盖、安全边界、并发输出、文件系统竞态和三宿主一致性；Critical与Important问题必须清零。

- [x] **Step 4: 同步远端并确认版本递增**

执行`git fetch origin main`，确认可快进且工作区只有本任务文件；三个版本都按功能发布递增次版本、补丁归零。

- [ ] **Step 5: 最后编辑版本、提交并推送**

版本文件作为最后编辑的文件，随后立即提交并推送，不再改其它文件。提交信息引用`#3`，核对父提交、文件列表、作者、状态后正常推送`main`。

## issue #3本轮执行记录

本轮补充挂载根权限兼容与路径替换防护，并按可观察行为完成测试、构建和只读审查。

- `Run`只对当前UID所有、与父目录跨设备、组可写但非`world`可写的挂载根通过已打开的目录FD追加`sticky`位，并保留原有`setuid`、`setgid`和权限位。`Check`不修改权限，缺少`sticky`位时拒绝；非跨设备的普通`0775`目录和任何`0777`目录仍拒绝。
- 目标目录链和组件路径使用`openat`与`O_NOFOLLOW`，后续权限调整、身份核对和发布继续绑定已打开的FD。Linux探针通过固定程序FD执行，Darwin探针通过固定安全父目录FD执行；根内链接解析、取消与路径身份变化均明确失败。
- 实现前取得`TestOwnedMount`、`TestParentMode`、`TestMountFix`、`TestFinalLink`、`TestCompSwap`、`TestProbeRace`、`TestFIFOFast`、`TestMkdirRace`、`TestProbeRelLink`和`TestProbeCancel`的红灯结果，完成实现后三个宿主模块均通过这些行为测试。
- 三个宿主模块的普通测试、`-race`测试和`go vet`均通过；`darwin-arm64`、`linux-arm64`与`linux-amd64`三架构交叉构建均通过。
- 最终只读审查结果为`Critical=0`、`Important=0`。

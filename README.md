# cloud-droid-prebuilts-toolchain

本私有仓库为`cloud-droid-hub`提供本地与Docker构建所需的固定宿主原生工具链。仓库只维护`main`分支，三个原平台分支的内容和历史分别归入以下顶层目录：

```text
darwin-arm64/
linux-arm64/
linux-amd64/
```

每个目录包含自己的安装程序、类型化归档目录、分片、许可证与测试。仓库统一版本由根目录[version.go](version.go)维护，三个宿主共同遵守根目录[AGENTS.md](AGENTS.md)。平台目录固定发布Go 1.26.8、LLVM 19.1.7、LLVM 22、Temurin JDK 21.0.12.1+1、Android NDK r29 revision 29.0.14206865、Bazel 8.5.1和Rust/Cargo/Rustdoc 1.97.1。macOS LLVM 22为22.1.8，Linux LLVM 22为Ubuntu 26.04的22.1.2；NDK内Clang与LLD主版本为21，Rust内部LLVM为22.1.6。三个宿主目录都包含`aarch64-linux-android`标准库。

归档以不超过90 MiB的普通Git对象保存，不使用Git LFS。`<host>/bin/cdh-toolchain`负责摘要校验、安全展开、宿主架构检查和逐组件原子发布。主项目把本仓放在`tools/toolchains/prebuilts/`，再把当前宿主工具展开到`output/toolchains/<host>/`；缺失、版本错误或架构不符都会直接失败，不会改用系统工具。

## 安装当前平台

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
toolchain_progress component="LLVM 22" index=4 total=7 remaining=3 elapsed_seconds=10 state=installing
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

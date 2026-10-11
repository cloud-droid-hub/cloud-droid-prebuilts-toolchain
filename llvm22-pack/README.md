# LLVM22归档条目转换

此工具读取已选定内容的`tar.xz`，将文件别名转换为包含最终目标字节的普通TAR条目，保留目标权限、全部路径与许可证。它不选择裁剪内容，不解包归档，不执行压缩、分卷、安装或正式发布。

接口：

```text
pack <input.tar.xz> <archive-prefix> <new-directory> [dest=source ...]
verify <input.tar.xz> <output.tar.xz> <archive-prefix> [dest=source ...]
```

`pack`只接受尚不存在的输出目录，生成`content.bin`、`<archive-prefix>-cdh.tar`和`entries.tsv`。失败时保留已有输入和部分候选，后续使用另一个新目录。命令记录输入和TAR的SHA-256；`verify`要求输入与输出具有不同文件身份，并对照全部条目的类型、权限、长度、目标路径及普通文件字节摘要。日志短写或写入失败均返回非零退出码。

需要为库加载补充普通副本时，显式传入相对归档根的`dest=source`，例如`lib/libdep.so=lib/cdh/libdep.so`。来源必须能解析到原输入中的普通文件，目标父目录必须已登记为真实目录；目标已存在、重复、越界、依赖另一新增副本或缺少来源均失败。副本保留来源的字节和权限，报告同时登记输出与来源路径，不修改二进制。`verify`必须传入同一组映射，只接受这些新增路径，并继续核对全部原始条目和许可；未登记的额外文件仍失败。

只有归档中的`bin/clang -> clang-22`和`bin/ld.lld -> lld`可以保留符号链接类型。其它符号链接和TAR硬链接条目解析到最终普通文件；不在文件系统中创建任何链接。目录别名、越界、绝对目标、循环、缺失目标、重复路径及特殊文件均拒绝。文件系统输入必须为`nlink=1`的普通文件；输出以独占创建写入并核对不同inode。暂存和输出前检查剩余空间，保留至少50GiB。

使用仓内登记Go1.26.8，正式配方的持久缓存保留在主仓`build/`，普通临时根在Go执行时的实际模块目录下分配。正式测试入口为`just test-llvm-pack`，执行登记工具链检查、单元测试和静态检查，也接入主仓`test`。单独执行模块测试：

```sh
tools/build/cwd-go.sh auto \
  -C toolchains/llvm22-pack test -count=1 ./...
```

默认测试仅使用小型归档和独立普通文件；链接拒绝测试使用归档条目或注入的文件元数据，不建立实际链接。`CDH_PACK_INPUT`与`CDH_PACK_PREFIX`可启用只读归档类型检查。

`TestNativeAlias`由`CDH_PACK_ROOT`显式启用，是额外的库别名诊断，包含同一进程加载多个别名。编译器、链接器及生成的程序必须先通过宿主架构检查。它不代表登记消费者验收：Linux消费者的`LIBCLANG_PATH`必须指向真实`lib/cdh`。既有诊断中macOS同进程多别名加载失败，Linux arm64外层库路径与多别名加载存在失败，日志保留在`output/llvm22-trim/regular-v2/logs/`。

三平台候选的实际安装与编译由组件的[候选检查工具](../prebuilts/tools/llvm22-check/README.md)验证。本工具的普通副本测试直接执行`pack`和`verify`，核对新增条目的类型、权限、字节、来源及未登记条目拒绝；这些小型归档测试不代表完整归档、动态依赖或安全组件构建已通过。

# LLVM 22候选安装与原生检查

本工具只检查单份LLVM 22候选。模块通过本地`replace`复用`darwin-arm64`模块中的公共`internal/catalog`、`internal/install`，不修改正式catalog、版本或其它工具链。

参数全部必填：

```text
llvm22-check --host <darwin-arm64|linux-arm64|linux-amd64> \
  --parts <payload-directory> --prefix <archive-prefix> \
  --install-dir <new-directory> --probe-dir <new-directory>
```

`--parts`必须只包含同一归档从`.tar.xz.part-000`开始的连续分卷。工具核对普通文件身份、大小和真实SHA-256，构造仅含LLVM 22的`catalog.Spec`。安装前只读检查完整TAR；除`bin/clang -> clang-22`与`bin/ld.lld -> lld`外，所有符号链接、硬链接条目和特殊文件均拒绝。登记链接仅由正式安装器处理，此工具不实现解包或创建链接。

两个输出目录必须全新且互不包含，父目录须已存在，并由调用者放在仓内`build/`或`output/`。归档展开空间和探测预留空间之外必须保留50 GiB。失败保留输入和已生成证据；再次执行使用新目录，不覆盖已有安装。日志或最终结果写入失败时返回非零退出码。

正式`install.RunTo/Check`通过后，检查原生二进制架构，使用安装目录内成对的Clang/LLD编译并运行C、带异常的C++和静态库链接程序，调用LLVM归档与二进制工具，实际`dlopen` libclang。Linux的`LIBCLANG_PATH`固定为本次安装的真实`llvm22/lib/cdh`；macOS使用`llvm22/lib`与`xcrun`定位的SDK。不会运行其它宿主架构的编译器或目标程序，也不安装系统依赖。

使用仓内登记Go 1.26.8构建，`TMPDIR`、`GOTMPDIR`指向组件`build/test-tmp/`，Go缓存同样置于组件`build/`。Linux amd64构建示例，工作目录为本组件根目录，`GO_BIN`必须指向登记Go：

```sh
rtk proxy env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOTOOLCHAIN=local \
  TMPDIR="$PWD/build/test-tmp" GOTMPDIR="$PWD/build/test-tmp" \
  GOCACHE="$PWD/build/go-cache" GOMODCACHE="$PWD/build/go-mod-cache" \
  "$GO_BIN" -C tools/llvm22-check build -mod=readonly -trimpath \
  -o "$PWD/build/llvm22-check/llvm22-check-linux-amd64" .
```

小范围测试包含真实小型归档安装、摘要变化拒绝、目录保护、架构检查与输出失败；链接拒绝使用归档元数据或文件信息注入，不实际创建链接。这些测试与交叉编译工具自身不代表真实LLVM候选已完成Linux原生验收。

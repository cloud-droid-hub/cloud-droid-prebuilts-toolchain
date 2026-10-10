# Python构建工具链裁剪

`python-trim`从固定Python`3.14.8`的Astral`python-build-standalone`完整归档生成`python314-build-v1`宿主工具链。三个宿主使用同一参数与裁剪规则，生成tar.gz普通文件，已有目标不覆盖。

输入来自[上游20261009发行](https://github.com/astral-sh/python-build-standalone/releases/tag/20261009)，文件名为`cpython-3.14.8+20261009-<target>-pgo+lto-full.tar.zst`。

| 宿主 | target | 输入字节数 | 输入SHA256 |
| --- | --- | --- | --- |
| darwin-arm64 | aarch64-apple-darwin | 61701302 | `6888111560a89c16e471894cca0f50d66f3bc00e5a16e91fab24c50e567ef5ee` |
| linux-arm64 | aarch64-unknown-linux-gnu | 96827232 | `e8b3b7186d7a11335f3d1f05ee1b19b469cd5bcaeabd6293afa3713f24ff7bf0` |
| linux-amd64 | x86_64-unknown-linux-gnu | 111243845 | `d4d83a2dbb9b7ee12373219113e9aa59ad943b8d381a43837209945bb5596e66` |

保留解释器、动态运行库、标准库、头文件、pip、ensurepip、venv和完整许可证，运行文件保持上游字节。删除`build/`、`libpython3.14.a`及配置目录中的对象副本、标准库下的`test/`与`tests/`、`__pycache__/`、`.pyc`、`.pyo`及`share/man/`。`unittest`等运行模块仍保留。完整归档的上游元数据以`upstream.json`保留，裁剪标记记录宿主、版本、输入摘要和删除规则。

归档处理不在宿主展开输入，不追随归档链接；检查规范路径、重复条目、受控相对链接、链接目标与必要运行文件。读取前后核对原输入的完整摘要及文件身份。输出使用确定性时间、权限和gzip头，经过当前宿主的不覆盖原子重命名发布，单份不得超过90MiB。工作路径必须位于项目`build/`内，受触碰文件系统须至少有50GiB可用空间。

在主仓根执行，输入放在`build/toolchains/python314/input/`，父目录预先建立：

```bash
mkdir -p build/toolchains/python314/input build/toolchains/python314/packed
just trim-python darwin-arm64 build/toolchains/python314/input/cpython-3.14.8-darwin-arm64.tar.zst build/toolchains/python314/packed/cpython-3.14.8-darwin-arm64.tar.gz
just test-python-trim
```

同一输入再次输出到新的目标后，用`cmp`验证完整归档字节一致。归档在预编译组件的`<host>/payload/python314/`保存为`.tar.gz.part-000`，其大小、摘要、许可证、裁剪标记和执行探针由对应类型化Go目录登记。三宿主安装器须从各自Go模块生成原生静态程序；架构检查和归档重放不代替其它宿主实际运行。

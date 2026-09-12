# AGENTS.md

本目录在根`main`分支维护Linux arm64原生工具链，并继承仓库根`AGENTS.md`的固定版本、归档、测试、版本和提交规则。不得要求切换或恢复旧`linux-arm64`分支。

- `bin/cdh-toolchain`只能是ELF aarch64静态Go程序。
- LLVM 22固定为Ubuntu 26.04的22.1.2，LLVM 19固定为19.1.7。
- NDK r29只提供Android arm64目标sysroot、运行库、元数据和Clang 21 resource目录。
- Android编译必须使用本目录LLVM 22与NDK目标输入，不能执行Linux amd64 NDK程序。
- 从仓库根调用安装程序时，`--repo`必须指向`$PWD/linux-arm64`。
- 本目录版本保存在`internal/version/version.go`，版本文件必须是涉及本目录提交前最后编辑的文件。

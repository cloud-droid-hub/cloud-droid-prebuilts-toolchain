# AGENTS.md

本目录在根`main`分支维护macOS arm64原生工具链，并继承仓库根`AGENTS.md`的固定版本、归档、测试、版本和提交规则。不得要求切换或恢复旧`darwin-arm64`分支。

- `bin/cdh-toolchain`只能是Mach-O arm64静态Go程序。
- LLVM 22固定为22.1.8，LLVM 19固定为19.1.7。
- NDK r29沿用`darwin-x86_64`内部目录名，但保留的宿主程序必须都是Mach-O arm64。
- 从仓库根调用安装程序时，`--repo`必须指向`$PWD/darwin-arm64`。
- 本目录版本保存在`internal/version/version.go`，版本文件必须是涉及本目录提交前最后编辑的文件。

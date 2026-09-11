# LLVM 19 Linux amd64附带库来源

LLVM主体来自LLVM项目`llvmorg-19.1.7`发布页的`LLVM-19.1.7-Linux-X64.tar.xz`。原始归档大小为`1653440720`字节，SHA-256为`4a5ec53951a584ed36f80240f6fbf8fdd46b4cf6c7ee87cc2d5018dc37caf679`。Apache License 2.0 with LLVM Exceptions原文取自同一tag的`LICENSE.TXT`，SHA-256为`8d85c1057d742e597985c7d4e6320b015a9139385cff4cbae06ffc0ebe89afee`。

官方`lld`直接需要`libxml2.so.2`，而基础`ubuntu:26.04`镜像不预装该SONAME。为使工具在干净镜像中直接运行，本归档附带以下Ubuntu 24.04 amd64官方包中的三个必要ELF库：

- `libxml2_2.9.14+dfsg-1.3ubuntu3.8_amd64.deb`，大小`763972`字节，SHA-256为`bfd07c01d6e5ab3e327f3ca5819409b1914bbfb3f1a016d53e4dabd5f96143bb`。
- `libicu74_74.2-1ubuntu3.1_amd64.deb`，大小`10860078`字节，SHA-256为`c9a70989678660eed9a1e904c74fa043da8bec8e2036856fc16e31ced79b04f8`。

仅保留`libxml2.so.2.9.14`、`libicuuc.so.74.2`、`libicudata.so.74.2`及其相对符号链接。三个库保持Ubuntu包内原始字节；`lld`使用Ubuntu 24.04的`patchelf 0.18.0`写入`DT_RPATH=$ORIGIN/../lib/cdh`，不要求用户安装额外系统包。构建期`patchelf_0.18.0-1.1build1_amd64.deb`大小为`106300`字节，SHA-256为`962a43e33cd56061522554898557a038ccbb8aa4e1e0f421b2d6f6adf1f80c60`，不随工具链发布。对应Ubuntu版权文件随归档和本目录一同发布。

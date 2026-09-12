# LLVM 19 Linux arm64附带库来源

LLVM主体来自`llvmorg-19.1.7`的官方`clang+llvm-19.1.7-aarch64-linux-gnu.tar.xz`。原始归档大小为`1131066132`字节，SHA-256为`a73d9326e5d756e3937df6a9f621664d76403b59119f741901106b387e53a6ae`；其签名已使用LLVM官网`release-keys.asc`验证，签名指纹为`0FB96824606D15FDD54C3D092FFC629D3F563BDC`。

官方`lld`直接需要`libxml2.so.2`，而基础`ubuntu:26.04`镜像不提供该SONAME。为使工具在干净镜像中直接运行，本归档附带以下Ubuntu 24.04 arm64官方包中的三个必要ELF库：

- `libxml2_2.9.14+dfsg-1.3ubuntu3.8_arm64.deb`，大小`735894`字节，SHA-256为`f833e07c5dffb9f7a26084522ef58854c4297982439a2affc94e20dbb495c696`。
- `libicu74_74.2-1ubuntu3.1_arm64.deb`，大小`10806898`字节，SHA-256为`48f93acf50dcf237a8d58ce366730a28438ce52d3f06d7a2a88b51261dd791f7`。

仅保留`libxml2.so.2.9.14`、`libicuuc.so.74.2`、`libicudata.so.74.2`及其相对符号链接。三个库保持Ubuntu包内原始字节；`lld`使用patchelf 0.18.0写入`DT_RPATH=$ORIGIN/../lib/cdh`，不要求用户安装额外APT包。对应Ubuntu版权文件随归档和本目录一同发布。

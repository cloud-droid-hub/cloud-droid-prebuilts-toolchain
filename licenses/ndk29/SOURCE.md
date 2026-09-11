# Android NDK r29 source record

- Revision: `29.0.14206865`
- Official archive: `https://dl.google.com/android/repository/android-ndk-r29-linux.zip`
- Archive size: `783549481` bytes
- Published SHA-1: `87e2bb7e9be5d6a1c6cdf5ec40dd4e0c6d07c30b`
- Verified SHA-256: `4abbbcdc842f3d4879206e9695d52709603e52dd68d3c1fff04b3b5e7a308ecf`

The branch payload is a deterministic host-tool archive derived from that
official archive. It retains `source.properties`, the upstream notices,
`ndk-build` with its bundled GNU Make, unified headers, every API directory
under `usr/lib/aarch64-linux-android`, Clang 21 headers and target resources,
and only AArch64 Android compiler-rt files from the multi-target runtime
directory.

The retained Linux x86_64 executables are the native Clang, LLD and LLVM tools
needed to compile, link, archive and inspect Android arm64 output. Non-arm64
target wrappers, sysroots, compiler runtimes and x86 target assemblers are
removed. A clean Ubuntu 26.04 amd64 audit found no missing dependency: all
host dependencies are baseline system libraries except the retained upstream
`libxml2.so.2` used by LLD.

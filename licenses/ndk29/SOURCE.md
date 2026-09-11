# Android NDK r29 source record

- Revision: `29.0.14206865`
- Official archive: `https://dl.google.com/android/repository/android-ndk-r29-linux.zip`
- Archive size: `783549481` bytes
- Published SHA-1: `87e2bb7e9be5d6a1c6cdf5ec40dd4e0c6d07c30b`
- Verified SHA-256: `4abbbcdc842f3d4879206e9695d52709603e52dd68d3c1fff04b3b5e7a308ecf`

The branch payload is a deterministic target-data archive derived from that
official archive. It retains `source.properties`, the upstream notices,
unified headers, every API directory under `usr/lib/aarch64-linux-android`,
Clang 21 headers and target resources, and only AArch64 Android compiler-rt
files from the multi-target runtime directory.

The payload contains no Linux x86_64 executable. Linux arm64 builds select the
branch-native LLVM 22 compiler and linker plus the NDK sysroot and Clang 21
resource directory through `cdh-toolchain select`. The current hub repository
entry that expects Google's x86_64 `ndk-build` launcher cannot consume this
target-data layout until its toolchain selection task is completed.

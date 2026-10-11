#!/bin/sh
if [ "${BASH_SOURCE:-$0}" = "$0" ] && [ "${BASH:-bootstrap}" != "${CDH_BASH_BIN:-}" ]; then
	exec "${CDH_BASH_BIN:-$(dirname "$0")/bash.sh}" "$0" "$@"
fi
# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
# 核验预编译sdkapp37组件：Build Tools 37.0.0与Platform android-36 r2的登记文件及设备HAL构建
# 所需的libbinder_ndk_cpp头文件齐全、版本一致、不是链接，原生程序为当前宿主格式。
# 只读检查，不下载、不安装、不回落到宿主SDK。
set -euo pipefail

usage() {
	echo "用法: android-sdk-check.sh <sdk-root> <macosx|linux> <machine>" >&2
	exit 2
}

[[ $# -eq 3 ]] || usage
root="$1"
host_os="$2"
machine="$3"
tools="${root}/build-tools/37.0.0"
platform="${root}/platforms/android-36"

fail() {
	echo "Android SDK组件无效:$*" >&2
	exit 1
}

[[ -d "${root}" && ! -L "${root}" ]] || fail "目录不存在或是链接:${root}"
for path in \
	"${tools}/aapt2" "${tools}/aidl" "${tools}/zipalign" "${tools}/d8" "${tools}/apksigner"; do
	[[ -f "${path}" && ! -L "${path}" && -x "${path}" ]] || fail "缺少可执行普通文件:${path}"
done
for path in \
	"${tools}/lib/d8.jar" "${tools}/lib/apksigner.jar" "${tools}/source.properties" \
	"${platform}/android.jar" "${platform}/source.properties" \
	"${platform}/optional/libbinder_ndk_cpp/android/binder_interface_utils.h" \
	"${platform}/optional/libbinder_ndk_cpp/android/binder_auto_utils.h"; do
	[[ -f "${path}" && ! -L "${path}" ]] || fail "缺少普通文件:${path}"
done
grep -qx 'Pkg.Revision=37.0.0' "${tools}/source.properties" || fail "Build Tools版本不是37.0.0:${tools}"
grep -qx 'AndroidVersion.ApiLevel=36' "${platform}/source.properties" || fail "Platform不是android-36:${platform}"
grep -qx 'Pkg.Revision=2' "${platform}/source.properties" || fail "Platform android-36修订不是2:${platform}"

case "${host_os}:${machine}" in
	macosx:arm64) want='Mach-O 64-bit executable arm64' ;;
	linux:aarch64|linux:arm64) want='ARM aarch64' ;;
	linux:x86_64|linux:amd64) want='x86-64' ;;
	*) fail "不支持的宿主:${host_os}:${machine}" ;;
esac
for name in aapt2 aidl zipalign; do
	kind="$(file -b "${tools}/${name}")"
	[[ "${kind}" == *"${want}"* ]] || fail "${name}不是当前宿主程序:${kind}"
done
echo "android-sdk=${root}"

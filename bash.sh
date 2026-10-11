#!/bin/sh
# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
set -eu

# 冷启动只依赖宿主POSIX入口，Bash脚本体始终由登记制品执行。
self=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
base=$(CDPATH= cd -- "$self/.." && pwd -P)
case "$(/usr/bin/uname -s)/$(/usr/bin/uname -m)" in
	Darwin/arm64) host=darwin-arm64 ;;
	Linux/arm64|Linux/aarch64) host=linux-arm64 ;;
	Linux/amd64|Linux/x86_64) host=linux-amd64 ;;
	*) printf '%s\n' 'Bash工具链不支持当前宿主' >&2; exit 2 ;;
esac
repo="$base/toolchains/$host"
dest=${CDH_TOOLCHAIN_ROOT:-$base/output/toolchains/$host}
case "$dest" in
	/*) ;;
	*) printf '%s\n' 'Bash工具链根必须使用绝对路径' >&2; exit 2 ;;
esac
installer="$repo/bin/cdh-toolchain"
if [ ! -d "$repo" ] && [ -e "$base/.git" ]; then
	git -C "$base" submodule update --init --depth 1 -- toolchains >&2
fi
if [ ! -x "$installer" ]; then
	printf '缺少登记Bash安装器：%s\n' "$installer" >&2
	exit 2
fi
if ! "$installer" check --dest "$dest" --component bash53 >/dev/null 2>&1; then
	"$installer" install --repo "$repo" --dest "$dest" --component bash53 >&2
fi
CDH_BASH_BIN="$dest/bash53/bin/bash"
CDH_BASH_RUN="$base/toolchains/bash.sh"
export CDH_BASH_BIN CDH_BASH_RUN
prefix="$dest/go126/bin:$dest/jdk21/bin:$dest/llvm22/bin:$dest/bazel851/bin:$dest/rust197/bin:$dest/bash53/bin"
case "$PATH" in
	"$prefix"|"$prefix":*|"$dest/bash53/bin"|"$dest/bash53/bin":*) ;;
	*) PATH="$dest/bash53/bin:$PATH" ;;
esac
export PATH
BASH_ENV= ENV=
export BASH_ENV ENV
exec "$CDH_BASH_BIN" "$@"

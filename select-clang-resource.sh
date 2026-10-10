#!/usr/bin/env bash
# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

set -euo pipefail

toolchain=""
major=""

usage() {
	cat <<'EOF'
usage: select-clang-resource.sh --toolchain DIR --major VERSION
EOF
}

while [[ "$#" -gt 0 ]]; do
	case "$1" in
	--toolchain)
		toolchain="${2:-}"
		shift 2
		;;
	--major)
		major="${2:-}"
		shift 2
		;;
	-h|--help)
		usage
		exit 0
		;;
	*)
		echo "未知参数:$1" >&2
		usage >&2
		exit 2
		;;
	esac
done

if [[ -z "${toolchain}" || -z "${major}" ]]; then
	echo "缺少Clang资源目录选择参数" >&2
	usage >&2
	exit 2
fi
if [[ ! "${major}" =~ ^[0-9]+$ ]]; then
	echo "Clang主版本无效:${major}" >&2
	exit 2
fi
if [[ ! -d "${toolchain}" || -L "${toolchain}" ]]; then
	echo "NDK工具链目录无效:${toolchain}" >&2
	exit 3
fi

lib_dir="${toolchain}/lib"
clang_dir="${lib_dir}/clang"
resource_dir="${clang_dir}/${major}"
for directory in "${lib_dir}" "${clang_dir}"; do
	if [[ ! -d "${directory}" || -L "${directory}" ]]; then
		echo "Clang资源目录无效:${directory}" >&2
		exit 3
	fi
done

if [[ -e "${resource_dir}" || -L "${resource_dir}" ]]; then
	if [[ ! -d "${resource_dir}" || -L "${resource_dir}" ]]; then
		echo "Clang资源目录无效:${resource_dir}" >&2
		exit 3
	fi
	printf '%s\n' "${resource_dir}"
	exit 0
fi

shopt -s nullglob
resource_dirs=("${clang_dir}"/*)
if [[ "${#resource_dirs[@]}" -ne 1 ]]; then
	echo "Clang资源目录数量无效:${#resource_dirs[@]}，需要主版本目录或唯一目录" >&2
	exit 3
fi
resource_dir="${resource_dirs[0]}"
if [[ ! -d "${resource_dir}" || -L "${resource_dir}" ]]; then
	echo "Clang资源目录无效:${resource_dir}" >&2
	exit 3
fi

printf '%s\n' "${resource_dir}"

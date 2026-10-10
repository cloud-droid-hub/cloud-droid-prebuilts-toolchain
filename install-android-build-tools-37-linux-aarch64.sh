#!/bin/sh
if [ "${BASH_SOURCE:-$0}" = "$0" ] && [ "${BASH:-bootstrap}" != "${CDH_BASH_BIN:-}" ]; then
	exec "${CDH_BASH_BIN:-$(dirname "$0")/bash.sh}" "$0" "$@"
fi
# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

set -euo pipefail

readonly build_tools_rev="37.0.0"
readonly source_repo="https://github.com/Commit451/android-arm-build-tools"
readonly archive_name="android-build-tools-37.0.0-linux-arm64-20260618.tar.xz"
readonly archive_url="https://github.com/Commit451/android-arm-build-tools/releases/download/platform-tools-37.0.0/${archive_name}"
readonly archive_sha256="29b1fa192c005d42c228b6ad0f0ad29161453be324308c3bfaa5783c7f76fddc"
readonly archive_size="3513876"
readonly metadata_schema="cdh.android-build-tools-linux-aarch64.v1"
readonly tool_names=(aapt2 aidl zipalign split-select)
readonly sha256_aapt2="e292e87995720841e6198ed89b03186b0f99bcdf382e4cf923dc3f39529f1788"
readonly tool_sha256_aidl="835d94ec1dc5a9854fc6a954e0e3f70599eff54fb5f1f79ea324ca166efbe743"
readonly sha256_zipalign="08caed25366460755dd6304136e3c722cdb7228869777eb3fc76138362f4a02d"
readonly sha256_split_sel="4b51dff38502c24b9418c50aff932f18c7d573df2eb9b47288995a52c0c428f9"

destination=""
cache_dir=""
archive_override=""
verify_only=""
download_tmp=""
work_dir=""

cleanup() {
	if [[ -n "${download_tmp}" ]]; then
		rm -f -- "${download_tmp}"
	fi
	if [[ -n "${work_dir}" ]]; then
		rm -rf -- "${work_dir}"
	fi
}
trap cleanup EXIT

usage() {
	cat <<'EOF'
usage:
  install-android-build-tools-37-linux-aarch64.sh --print-metadata
  install-android-build-tools-37-linux-aarch64.sh --verify DIR
  install-android-build-tools-37-linux-aarch64.sh --destination DIR [--cache-dir DIR] [--archive FILE]
EOF
}

print_metadata() {
	printf '{"schema":"%s","revision":"%s","source":"%s","url":"%s","sha256":"%s","archive_size":%s,"host_machine":"AArch64","tools":{"aapt2":"%s","aidl":"%s","zipalign":"%s","split-select":"%s"}}\n' \
		"${metadata_schema}" "${build_tools_rev}" "${source_repo}" "${archive_url}" \
		"${archive_sha256}" "${archive_size}" "${sha256_aapt2}" "${tool_sha256_aidl}" \
		"${sha256_zipalign}" "${sha256_split_sel}"
}

require_command() {
	local command_name="$1"
	if ! command -v "${command_name}" >/dev/null 2>&1; then
		echo "缺少命令:${command_name}" >&2
		exit 2
	fi
}

tool_sha256() {
	case "$1" in
		aapt2) printf '%s\n' "${sha256_aapt2}" ;;
		aidl) printf '%s\n' "${tool_sha256_aidl}" ;;
		zipalign) printf '%s\n' "${sha256_zipalign}" ;;
		split-select) printf '%s\n' "${sha256_split_sel}" ;;
		*) return 1 ;;
	esac
}

verify_archive() {
	local archive_path="$1"
	if [[ ! -f "${archive_path}" || -L "${archive_path}" ]]; then
		echo "Android build-tools归档必须是普通文件:${archive_path}" >&2
		return 1
	fi
	if [[ "$(stat -Lc %s -- "${archive_path}")" != "${archive_size}" ]]; then
		echo "Android build-tools归档大小不匹配:${archive_path}" >&2
		return 1
	fi
	printf '%s  %s\n' "${archive_sha256}" "${archive_path}" | sha256sum -c >/dev/null
}

download_archive() {
	local output_path="$1"
	local output_identity=""
	local current_identity=""
	local actual_size="0"
	local curl_status="0"
	local attempt=""

	output_identity="$(stat -Lc '%d:%i' -- "${output_path}")"
	for attempt in 1 2 3 4 5 6; do
		curl_status=0
		if [[ -s "${output_path}" ]]; then
			curl --http1.1 -L --continue-at - --fail --silent --show-error \
				--connect-timeout 20 --speed-limit 16384 --speed-time 60 --max-time 1800 \
				-o "${output_path}" "${archive_url}" || curl_status=$?
		else
			curl --http1.1 -L --fail --silent --show-error \
				--connect-timeout 20 --speed-limit 16384 --speed-time 60 --max-time 1800 \
				-o "${output_path}" "${archive_url}" || curl_status=$?
		fi
		if [[ ! -f "${output_path}" || -L "${output_path}" ]]; then
			echo "Android build-tools下载临时文件被替换:${output_path}" >&2
			return 1
		fi
		current_identity="$(stat -Lc '%d:%i' -- "${output_path}")"
		if [[ "${current_identity}" != "${output_identity}" ]]; then
			echo "Android build-tools下载临时文件身份变化:${output_path}" >&2
			return 1
		fi
		actual_size="$(stat -Lc %s -- "${output_path}")"
		if [[ "${actual_size}" == "${archive_size}" ]]; then
			return 0
		fi
		if ((actual_size > archive_size)); then
			echo "Android build-tools下载超过固定大小:${actual_size}/${archive_size}" >&2
			return 1
		fi
		echo "Android build-tools下载未完成:attempt=${attempt} curl=${curl_status} bytes=${actual_size}/${archive_size}" >&2
		sleep 2
	done
	return 1
}

validate_paths() {
	local archive_path="$1"
	tar -tJf "${archive_path}" | awk '
		BEGIN {
			expected["./"]=1
			expected["./MANIFEST.txt"]=1
			expected["./aapt2"]=1
			expected["./aidl"]=1
			expected["./zipalign"]=1
			expected["./split-select"]=1
		}
		{
			if (!($0 in expected) || ($0 in seen)) bad=1
			seen[$0]=1
		}
		END {
			for (path in expected) if (!(path in seen)) bad=1
			exit bad
		}'
}

validate_tools() {
	local root="$1"
	local tool=""
	local executable=""
	local expected=""
	local dyn_libraries=""

	if [[ ! -d "${root}" || -L "${root}" ]]; then
		echo "Android build-tools目录无效:${root}" >&2
		return 1
	fi
	for tool in "${tool_names[@]}"; do
		executable="${root}/${tool}"
		expected="$(tool_sha256 "${tool}")"
		if [[ ! -f "${executable}" || -L "${executable}" || ! -x "${executable}" ]]; then
			echo "Android build-tools缺少原生工具:${executable}" >&2
			return 1
		fi
		printf '%s  %s\n' "${expected}" "${executable}" | sha256sum -c >/dev/null
		if ! file "${executable}" | grep -q 'ELF 64-bit.*ARM aarch64'; then
			echo "Android build-tools不是Linux arm64 ELF:${executable}" >&2
			return 1
		fi
		if ! readelf -h "${executable}" | grep -q 'Machine:.*AArch64' || \
			! readelf -l "${executable}" | grep -q '/lib/ld-linux-aarch64.so.1'; then
			echo "Android build-tools ELF约定不匹配:${executable}" >&2
			return 1
		fi
		dyn_libraries="$(ldd "${executable}")"
		if grep -q 'not found' <<<"${dyn_libraries}"; then
			echo "Android build-tools缺少动态库:${executable}" >&2
			return 1
		fi
	done
	"${root}/aapt2" version >/dev/null 2>&1
	"${root}/aidl" --help >/dev/null 2>&1
	"${root}/split-select" --help >/dev/null 2>&1
}

validate_base() {
	local root="$1"
	local properties="${root}/source.properties"
	if [[ ! -d "${root}" || -L "${root}" ]]; then
		echo "Android build-tools目标必须是普通目录:${root}" >&2
		return 1
	fi
	if [[ ! -f "${properties}" || -L "${properties}" ]] || \
		! grep -Eq "^Pkg.Revision[[:space:]]*=[[:space:]]*${build_tools_rev}$" "${properties}"; then
		echo "Android build-tools版本信息无效:${properties}" >&2
		return 1
	fi
	for executable in "${root}/d8" "${root}/apksigner"; do
		if [[ ! -f "${executable}" || -L "${executable}" || ! -x "${executable}" ]]; then
			echo "Android build-tools缺少官方Java入口:${executable}" >&2
			return 1
		fi
	done
	if find "${root}" -xdev -type l -print -quit | grep -q . || \
		find "${root}" -xdev ! -type f ! -type d -print -quit | grep -q .; then
		echo "Android build-tools官方目录包含不支持的文件类型:${root}" >&2
		return 1
	fi
}

validate_install() {
	local root="$1"
	validate_base "${root}"
	validate_tools "${root}"
}

if [[ "${1:-}" == "--print-metadata" ]]; then
	if [[ "$#" -ne 1 ]]; then
		usage >&2
		exit 2
	fi
	print_metadata
	exit 0
fi

while [[ "$#" -gt 0 ]]; do
	case "$1" in
		--destination)
		destination="${2:-}"
		shift 2
		;;
		--cache-dir)
		cache_dir="${2:-}"
		shift 2
		;;
		--archive)
		archive_override="${2:-}"
		shift 2
		;;
		--verify)
		verify_only="${2:-}"
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

if [[ "$(uname -s)" != "Linux" || ! "$(uname -m)" =~ ^(aarch64|arm64)$ ]]; then
	echo "原生Android build-tools安装入口只支持Linux arm64:$(uname -s)/$(uname -m)" >&2
	exit 3
fi
for command_name in awk curl file find grep ldd readelf realpath sha256sum stat sync tar xz; do
	require_command "${command_name}"
done

if [[ -n "${verify_only}" ]]; then
	if [[ -n "${destination}" || -n "${cache_dir}" || -n "${archive_override}" ]]; then
		usage >&2
		exit 2
	fi
	verify_only="$(realpath -m -- "${verify_only}")"
	validate_install "${verify_only}"
	echo "==> Linux arm64 Android build-tools校验通过:${verify_only}"
	exit 0
fi

if [[ -z "${destination}" ]]; then
	echo "缺少--destination" >&2
	usage >&2
	exit 2
fi
destination="$(realpath -m -- "${destination}")"
if [[ "${destination}" == "/" ]]; then
	echo "Android build-tools目标目录不能是根目录" >&2
	exit 3
fi
if validate_install "${destination}" >/dev/null 2>&1; then
	echo "==> 保留已有Linux arm64 Android build-tools:${destination}"
	exit 0
fi
validate_base "${destination}"

if [[ -n "${archive_override}" ]]; then
	archive_path="$(realpath -m -- "${archive_override}")"
else
	if [[ -z "${cache_dir}" ]]; then
		cache_dir="${XDG_CACHE_HOME:-${HOME}/.cache}/cloud-droid-hub/toolchains"
	fi
	cache_dir="$(realpath -m -- "${cache_dir}")"
	install -d -m 0755 -- "${cache_dir}"
	if [[ -L "${cache_dir}" || ! -d "${cache_dir}" ]]; then
		echo "Android build-tools缓存目录无效:${cache_dir}" >&2
		exit 4
	fi
	archive_path="${cache_dir}/${archive_name}"
	if [[ -e "${archive_path}" || -L "${archive_path}" ]] && \
		[[ ! -f "${archive_path}" || -L "${archive_path}" ]]; then
		echo "Android build-tools缓存路径不是普通文件:${archive_path}" >&2
		exit 4
	fi
	if ! verify_archive "${archive_path}" >/dev/null 2>&1; then
		download_tmp="$(mktemp "${cache_dir}/.${archive_name}.download.XXXXXX")"
		download_archive "${download_tmp}"
		verify_archive "${download_tmp}"
		chmod 0644 "${download_tmp}"
		mv -f -- "${download_tmp}" "${archive_path}"
		download_tmp=""
	fi
fi

verify_archive "${archive_path}"
validate_paths "${archive_path}"

parent_dir="$(dirname -- "${destination}")"
if [[ -L "${parent_dir}" || ! -d "${parent_dir}" ]]; then
	echo "Android build-tools目标父目录无效:${parent_dir}" >&2
	exit 4
fi
work_dir="$(mktemp -d "${parent_dir}/.build-tools-37-linux-aarch64.XXXXXX")"
chmod 0700 "${work_dir}"
unpacked="${work_dir}/unpacked"
candidate="${work_dir}/candidate"
mkdir -p "${unpacked}" "${candidate}"
tar -xJf "${archive_path}" --no-same-owner --no-same-permissions --delay-directory-restore -C "${unpacked}"
if [[ "$(find "${unpacked}" -mindepth 1 -maxdepth 1 -type f | wc -l)" != "5" ]] || \
	find "${unpacked}" -mindepth 1 ! -type f -print -quit | grep -q .; then
	echo "Android build-tools解包内容不符合固定清单:${archive_path}" >&2
	exit 5
fi
validate_tools "${unpacked}"

cp -a -- "${destination}/." "${candidate}/"
for tool in "${tool_names[@]}"; do
	install -m 0755 -- "${unpacked}/${tool}" "${candidate}/${tool}"
done
validate_install "${candidate}"
sync -f "${candidate}"

backup="${work_dir}/previous"
mv -- "${destination}" "${backup}"
if ! mv -- "${candidate}" "${destination}"; then
	mv -- "${backup}" "${destination}"
	exit 6
fi
if ! validate_install "${destination}"; then
	mv -- "${destination}" "${work_dir}/invalid"
	mv -- "${backup}" "${destination}"
	exit 6
fi
sync -f "${destination}"
sync -f "${parent_dir}"
rm -rf -- "${backup}"

echo "==> Linux arm64 Android build-tools安装完成"
echo "    ${destination}"

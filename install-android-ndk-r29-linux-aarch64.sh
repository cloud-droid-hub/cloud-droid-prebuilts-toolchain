#!/usr/bin/env bash
# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

set -euo pipefail

readonly ndk_revision="29.0.14206865"
readonly archive_name="android-ndk-r29-linux-aarch64.tar.gz"
readonly archive_url="https://github.com/SnowNF/ndk-aarch64-linux/releases/download/0.0.2/${archive_name}"
readonly archive_sha256="48cb104c28e1ede5e1884b0b34d97d28c4df74cd4e8f7628202a4c2c8de78a50"
readonly archive_size="1857997477"
readonly metadata_schema="cdh.android-ndk-linux-aarch64.v1"
readonly clang_major="22"

destination=""
cache_dir=""
archive_override=""
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
  install-android-ndk-r29-linux-aarch64.sh --print-metadata
  install-android-ndk-r29-linux-aarch64.sh --destination DIR [--cache-dir DIR] [--archive FILE]
EOF
}

print_metadata() {
	printf '{"schema":"%s","revision":"%s","url":"%s","sha256":"%s","archive_size":%s,"host_machine":"AArch64","clang_major":%s}\n' \
		"${metadata_schema}" "${ndk_revision}" "${archive_url}" "${archive_sha256}" "${archive_size}" "${clang_major}"
}

require_command() {
	local command_name="$1"
	if ! command -v "${command_name}" >/dev/null 2>&1; then
		echo "缺少命令:${command_name}" >&2
		exit 2
	fi
}

verify_archive() {
	local archive_path="$1"
	if [[ ! -f "${archive_path}" || -L "${archive_path}" ]]; then
		echo "NDK归档必须是普通文件:${archive_path}" >&2
		return 1
	fi
	if [[ "$(stat -Lc %s -- "${archive_path}")" != "${archive_size}" ]]; then
		echo "NDK归档大小不匹配:${archive_path}" >&2
		return 1
	fi
	printf '%s  %s\n' "${archive_sha256}" "${archive_path}" | sha256sum -c >/dev/null
}

download_archive() {
	local output_path="$1"
	local attempt=""
	local output_identity=""
	local current_identity=""
	local actual_size="0"
	local curl_status="0"

	output_identity="$(stat -Lc '%d:%i' -- "${output_path}")"
	for attempt in 1 2 3 4 5 6 7 8; do
		curl_status=0
		if [[ -s "${output_path}" ]]; then
			curl --http1.1 -L --continue-at - --fail --silent --show-error \
				--connect-timeout 20 --speed-limit 32768 --speed-time 90 --max-time 7200 \
				-o "${output_path}" "${archive_url}" || curl_status=$?
		else
			curl --http1.1 -L --fail --silent --show-error \
				--connect-timeout 20 --speed-limit 32768 --speed-time 90 --max-time 7200 \
				-o "${output_path}" "${archive_url}" || curl_status=$?
		fi
		if [[ ! -f "${output_path}" || -L "${output_path}" ]]; then
			echo "NDK下载临时文件被替换:${output_path}" >&2
			return 1
		fi
		current_identity="$(stat -Lc '%d:%i' -- "${output_path}")"
		if [[ "${current_identity}" != "${output_identity}" ]]; then
			echo "NDK下载临时文件身份变化:${output_path}" >&2
			return 1
		fi
		actual_size="$(stat -Lc %s -- "${output_path}")"
		if [[ "${actual_size}" == "${archive_size}" ]]; then
			return 0
		fi
		if ((actual_size > archive_size)); then
			echo "NDK下载超过固定大小:${actual_size}/${archive_size}" >&2
			return 1
		fi
		echo "NDK下载未完成:attempt=${attempt} curl=${curl_status} bytes=${actual_size}/${archive_size}" >&2
		sleep 3
	done
	return 1
}

verify_tar_paths() {
	local archive_path="$1"
	tar -tzf "${archive_path}" | awk '
		BEGIN { count=0; unsafe=0; top_count=0 }
		{
			count++
			path=$0
			if (path ~ /^\// || path ~ /(^|\/)\.\.($|\/)/) {
				unsafe++
			}
			split(path, parts, "/")
			if (parts[1] != "" && !(parts[1] in top)) {
				top[parts[1]]=1
				top_count++
			}
		}
		END {
			if (count == 0 || unsafe != 0 || top_count != 1 || !("r29" in top)) {
				exit 9
			}
		}'
}

verify_ext_links() {
	local root="$1"
	local link=""
	local target=""
	while IFS= read -r -d '' link; do
		target="$(realpath -m -- "${link}")"
		case "${target}" in
			"${root}"/*)
				;;
			/usr/bin/make|/usr/bin/python3.14|/usr/bin/yasm|/usr/bin/ytasm)
				;;
			*)
				echo "NDK包含未允许的外部链接:${link} -> ${target}" >&2
				return 1
				;;
		esac
	done < <(find "${root}" -xdev -type l -print0)
}

verify_install() {
	local root="$1"
	local toolchain="${root}/toolchains/llvm/prebuilt/linux-x86_64"
	local clang="${toolchain}/bin/clang"
	local lld="${toolchain}/bin/ld.lld"
	local target_clang="${toolchain}/bin/aarch64-linux-android35-clang"
	local properties="${root}/source.properties"
	local clang_version=""

	if [[ ! -d "${root}" || -L "${root}" ]]; then
		echo "NDK安装根目录无效:${root}" >&2
		return 1
	fi
	if [[ ! -f "${properties}" || -L "${properties}" ]] || \
		! grep -qx "Pkg.Revision = ${ndk_revision}" "${properties}"; then
		echo "NDK版本信息无效:${properties}" >&2
		return 1
	fi
	for executable in "${root}/ndk-build" "${clang}" "${lld}" "${target_clang}" \
		"${toolchain}/bin/llvm-ar" "${toolchain}/bin/llvm-nm" \
		"${toolchain}/bin/llvm-objcopy" "${toolchain}/bin/llvm-strip"; do
		if [[ ! -x "${executable}" ]]; then
			echo "NDK缺少可执行工具:${executable}" >&2
			return 1
		fi
	done
	if ! file -L "${clang}" | grep -q 'ELF 64-bit.*ARM aarch64'; then
		echo "NDK clang不是Linux arm64 ELF:${clang}" >&2
		return 1
	fi
	if ! file -L "${lld}" | grep -q 'ELF 64-bit.*ARM aarch64'; then
		echo "NDK ld.lld不是Linux arm64 ELF:${lld}" >&2
		return 1
	fi
	clang_version="$("${clang}" --version)"
	if ! grep -q "clang version ${clang_major}\." <<<"${clang_version}" || \
		! grep -q '^Target: aarch64-unknown-linux-gnu$' <<<"${clang_version}"; then
		echo "NDK clang版本或宿主目标不匹配:${clang}" >&2
		return 1
	fi
	if ! "${lld}" --version | grep -q "^LLD ${clang_major}\."; then
		echo "NDK ld.lld版本不匹配:${lld}" >&2
		return 1
	fi
	if find "${root}" -xdev ! -type f ! -type d ! -type l -print -quit | grep -q .; then
		echo "NDK目录包含不支持的文件类型:${root}" >&2
		return 1
	fi
	verify_ext_links "${root}"
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

if [[ -z "${destination}" ]]; then
	echo "缺少--destination" >&2
	usage >&2
	exit 2
fi
if [[ "$(uname -s)" != "Linux" || ! "$(uname -m)" =~ ^(aarch64|arm64)$ ]]; then
	echo "原生NDK安装入口只支持Linux arm64:$(uname -s)/$(uname -m)" >&2
	exit 3
fi
for command_name in awk curl file find grep realpath sha256sum stat sync tar; do
	require_command "${command_name}"
done

destination="$(realpath -m -- "${destination}")"
if [[ "${destination}" == "/" ]]; then
	echo "NDK目标目录不能是根目录" >&2
	exit 3
fi
if [[ -e "${destination}" || -L "${destination}" ]]; then
	if verify_install "${destination}"; then
		echo "==> 保留已有Linux arm64 NDK:${destination}"
		exit 0
	fi
	if [[ ! -d "${destination}" || -L "${destination}" ]]; then
		echo "拒绝替换非普通目录NDK目标:${destination}" >&2
		exit 4
	fi
fi

if [[ -n "${archive_override}" ]]; then
	archive_path="$(realpath -m -- "${archive_override}")"
else
	if [[ -z "${cache_dir}" ]]; then
		cache_dir="${XDG_CACHE_HOME:-${HOME}/.cache}/cloud-droid-hub/toolchains"
	fi
	cache_dir="$(realpath -m -- "${cache_dir}")"
	install -d -m 0755 -- "${cache_dir}"
	if [[ -L "${cache_dir}" || ! -d "${cache_dir}" ]]; then
		echo "NDK缓存目录无效:${cache_dir}" >&2
		exit 4
	fi
	archive_path="${cache_dir}/${archive_name}"
	if [[ -e "${archive_path}" || -L "${archive_path}" ]]; then
		if [[ ! -f "${archive_path}" || -L "${archive_path}" ]]; then
			echo "NDK缓存路径不是普通文件:${archive_path}" >&2
			exit 4
		fi
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
verify_tar_paths "${archive_path}"

parent_dir="$(dirname -- "${destination}")"
install -d -m 0755 -- "${parent_dir}"
if [[ -L "${parent_dir}" || ! -d "${parent_dir}" ]]; then
	echo "NDK目标父目录无效:${parent_dir}" >&2
	exit 4
fi
work_dir="$(mktemp -d "${parent_dir}/.ndk-r29-linux-aarch64.XXXXXX")"
chmod 0700 "${work_dir}"
tar -xzf "${archive_path}" --no-same-owner --no-same-permissions --delay-directory-restore -C "${work_dir}"
candidate="${work_dir}/r29"
verify_install "${candidate}"

backup=""
if [[ -e "${destination}" ]]; then
	backup="${work_dir}/previous"
	mv -- "${destination}" "${backup}"
fi
if ! mv -- "${candidate}" "${destination}"; then
	if [[ -n "${backup}" ]]; then
		mv -- "${backup}" "${destination}"
	fi
	exit 5
fi
if ! verify_install "${destination}"; then
	mv -- "${destination}" "${work_dir}/invalid"
	if [[ -n "${backup}" ]]; then
		mv -- "${backup}" "${destination}"
	fi
	exit 5
fi
sync -f "${destination}"
sync -f "${parent_dir}"
if [[ -n "${backup}" ]]; then
	rm -rf -- "${backup}"
fi

echo "==> Linux arm64 NDK安装完成"
echo "    ${destination}"

#!/bin/sh
if [ "${BASH_SOURCE:-$0}" = "$0" ] && [ "${BASH:-bootstrap}" != "${CDH_BASH_BIN:-}" ]; then
	exec "${CDH_BASH_BIN:-$(dirname "$0")/bash.sh}" "$0" "$@"
fi
# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
readonly script_dir
project_root="$(cd -- "$script_dir/.." && pwd -P)"
readonly project_root
toolchain_root=""
go_bin=""
python_bin=""
config_json=""
source_tag=""
schema=""
tool_name=""
platform_sdk=""
compiler_major=""
linker_major=""
c_compiler=""
cxx_compiler=""
lld_linker=""
default_cache=""
probe_input=""
probe_import=""
probe_output=""
source_names=()
source_remotes=()

destination=""
source_root=""
cache_root=""
verify_root=""
check_paths=0
check_toolchain=0
dest_identity=""
build_work=""
publish_temp=""
publish_backup=""
publish_target=""
publish_done=0
probe_temp=""
probe_digest=""
remote_names=()
remote_values=()
remote_value=""

cleanup() {
	local status=$?
	trap - EXIT HUP INT TERM
	set +e
	if [[ -n "$publish_backup" && ( -e "$publish_backup" || -L "$publish_backup" ) ]] && ((publish_done == 0)); then
		if [[ -n "$publish_target" && ! -e "$publish_target" && ! -L "$publish_target" ]]; then
			if ! mv -T -n -- "$publish_backup" "$publish_target"; then
				echo "AIDL旧输出回移失败:$publish_backup:$publish_target" >&2
				status=5
			elif [[ -e "$publish_backup" || -L "$publish_backup" ]]; then
				echo "AIDL回移目标被并发占用，新旧输出均已保留:$publish_target:$publish_backup" >&2
				status=5
			fi
		else
			echo "AIDL发布中断，新旧输出均已保留:$publish_target:$publish_backup" >&2
			status=5
		fi
	fi
	if [[ -n "$publish_temp" && ( -e "$publish_temp" || -L "$publish_temp" ) ]]; then
		rm -rf -- "$publish_temp"
	fi
	if [[ -n "$build_work" && ( -e "$build_work" || -L "$build_work" ) ]]; then
		rm -rf -- "$build_work"
	fi
	if [[ -n "$probe_temp" && ( -e "$probe_temp" || -L "$probe_temp" ) ]]; then
		rm -rf -- "$probe_temp"
	fi
	exit "$status"
}

signal_exit() {
	exit "$1"
}

trap cleanup EXIT
trap 'signal_exit 129' HUP
trap 'signal_exit 130' INT
trap 'signal_exit 143' TERM

usage() {
	cat <<'EOF'
用法:
  build-android-aidl36-linux-native.sh --print-metadata
  build-android-aidl36-linux-native.sh --print-plan --host linux/arm64|linux/amd64
  build-android-aidl36-linux-native.sh --print-defaults
  build-android-aidl36-linux-native.sh --verify DIR
  build-android-aidl36-linux-native.sh --check-toolchain
  build-android-aidl36-linux-native.sh --check-paths --destination DIR [--source-dir DIR] [--cache-dir DIR]
  build-android-aidl36-linux-native.sh --destination DIR [--source-dir DIR] [--cache-dir DIR]
EOF
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "缺少构建命令:$1" >&2
		exit 2
	}
}

fixed_tc_root() {
	local host="$1"
	if [[ -f /.dockerenv || -f /run/.containerenv ]]; then
		printf '/opt/cloud-droid-toolchains/%s\n' "$host"
	else
		printf '%s/output/target/common/toolchains/%s\n' "$project_root" "$host"
	fi
}

check_config_go() {
	local host goos goarch description version expected_root expected_go
	[[ -z "${CDH_TOOLCHAINCFG:-}" ]] || {
		echo "AIDL不接受外部类型化配置程序:${CDH_TOOLCHAINCFG}" >&2
		exit 3
	}
	case "$(uname -s)/$(uname -m)" in
		Darwin/arm64) host="darwin-arm64"; goos="darwin"; goarch="arm64" ;;
		Linux/aarch64|Linux/arm64) host="linux-arm64"; goos="linux"; goarch="arm64" ;;
		Linux/x86_64|Linux/amd64) host="linux-amd64"; goos="linux"; goarch="amd64" ;;
		*) echo "AIDL配置宿主不受支持:$(uname -s)/$(uname -m)" >&2; exit 3 ;;
	esac
	expected_root="$(fixed_tc_root "$host")"
	expected_go="$expected_root/go126/bin/go"
	[[ -z "${CDH_TOOLCHAIN_ROOT:-}" || "${CDH_TOOLCHAIN_ROOT}" == "$expected_root" ]] || {
		echo "AIDL配置工具链偏离项目固定安装根:${CDH_TOOLCHAIN_ROOT}:$expected_root" >&2
		exit 3
	}
	[[ -z "${GO_BIN:-}" || "${GO_BIN}" == "$expected_go" ]] || {
		echo "AIDL配置Go偏离项目固定安装根:${GO_BIN}:$expected_go" >&2
		exit 3
	}
	toolchain_root="$expected_root"
	go_bin="$expected_go"
	[[ -d "$toolchain_root" && ! -L "$toolchain_root" ]] || {
		echo "AIDL配置工具链必须来自项目固定安装根:$toolchain_root" >&2
		exit 3
	}
	[[ -f "$go_bin" && ! -L "$go_bin" && -x "$go_bin" ]] || {
		echo "AIDL配置Go必须来自预编译工具链:$go_bin" >&2
		exit 3
	}
	need_cmd file
	description="$(file -Lb "$go_bin")"
	case "$host" in
		darwin-arm64) [[ "$description" == *"Mach-O 64-bit executable arm64"* ]] ;;
		linux-arm64) [[ "$description" == *"ELF 64-bit"*"ARM aarch64"* ]] ;;
		linux-amd64) [[ "$description" == *"ELF 64-bit"*"x86-64"* ]] ;;
	esac || {
		echo "AIDL配置Go宿主架构不匹配:$description" >&2
		exit 3
	}
	version="$(GOTOOLCHAIN=local "$go_bin" version)"
	[[ "$version" == "go version go1.26.8 $goos/$goarch" ]] || {
		echo "AIDL配置Go版本必须是1.26.8:$version" >&2
		exit 3
	}
}

check_config_py() {
	local expected_py description version
	expected_py="$toolchain_root/python314/bin/python3.14"
	[[ -z "${CDH_PYTHON_BIN:-}" || "$CDH_PYTHON_BIN" == "$expected_py" ]] || {
		echo "AIDL配置Python偏离项目固定安装根:${CDH_PYTHON_BIN}:$expected_py" >&2
		exit 3
	}
	python_bin="$expected_py"
	[[ -f "$python_bin" && -x "$python_bin" ]] || {
		echo "AIDL配置Python必须来自预编译工具链:$python_bin" >&2
		exit 3
	}
	description="$(file -Lb "$python_bin")"
	case "$(uname -s)/$(uname -m)" in
		Darwin/arm64) [[ "$description" == *"Mach-O 64-bit executable arm64"* ]] ;;
		Linux/aarch64|Linux/arm64) [[ "$description" == *"ELF 64-bit"*"ARM aarch64"* ]] ;;
		Linux/x86_64|Linux/amd64) [[ "$description" == *"ELF 64-bit"*"x86-64"* ]] ;;
	esac || {
		echo "AIDL配置Python宿主架构不匹配:$description" >&2
		exit 3
	}
	version="$("$python_bin" --version)"
	[[ "$version" == "Python 3.14.8" ]] || {
		echo "AIDL配置Python版本必须是3.14.8:$version" >&2
		exit 3
	}
}

load_config() {
	local fields=() field source_count expected index offset
	config_json="$(cd -- "$project_root" && CDH_TARGET=common GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GONOPROXY=none \
		"$project_root/build/cwd-go.sh" "$go_bin" run -buildvcs=false ./cmd/toolchaincfg aidl36)"
	# NUL分隔字段保留空格和换行，不经Shell重新拆词。
	while IFS= read -r -d '' field; do
		fields+=("$field")
	done < <("$python_bin" - "$config_json" <<'PY'
import json
import sys

config = json.loads(sys.argv[1])
values = [
    config["schema"],
    config["android_tag"],
    config["tool"],
    str(config["platform_sdk"]),
    str(config["compiler_major"]),
    str(config["linker_major"]),
    config["c_compiler"],
    config["cxx_compiler"],
    config["linker"],
    config["default_cache"],
    config["probe_input"],
    config["probe_import"],
    config["probe_output"],
    str(len(config["source_repositories"])),
]
for source in config["source_repositories"]:
    values.extend((source["name"], source["url"]))
for value in values:
    if not isinstance(value, str) or not value or "\0" in value:
        raise SystemExit("AIDL typed configuration contains an invalid value")
    sys.stdout.buffer.write(value.encode("utf-8") + b"\0")
PY
)
	((${#fields[@]} >= 14)) || { echo "AIDL类型化配置字段不完整" >&2; exit 2; }
	schema="${fields[0]}"
	source_tag="${fields[1]}"
	tool_name="${fields[2]}"
	platform_sdk="${fields[3]}"
	compiler_major="${fields[4]}"
	linker_major="${fields[5]}"
	c_compiler="${fields[6]}"
	cxx_compiler="${fields[7]}"
	lld_linker="${fields[8]}"
	default_cache="${fields[9]}"
	probe_input="${fields[10]}"
	probe_import="${fields[11]}"
	probe_output="${fields[12]}"
	source_count="${fields[13]}"
	[[ "$platform_sdk" =~ ^[1-9][0-9]*$ && "$compiler_major" =~ ^[1-9][0-9]*$ && "$linker_major" =~ ^[1-9][0-9]*$ && "$source_count" =~ ^[1-9][0-9]*$ ]] || {
		echo "AIDL类型化配置数值无效" >&2
		exit 2
	}
	expected=$((14 + source_count * 2))
	((${#fields[@]} == expected)) || { echo "AIDL类型化源码仓字段不完整" >&2; exit 2; }
	offset=14
	for ((index = 0; index < source_count; index++)); do
		source_names+=("${fields[offset]}")
		source_remotes+=("${fields[offset + 1]}")
		offset=$((offset + 2))
	done
	cache_root="$project_root/$default_cache"
}

host_info() {
	"$python_bin" - "$config_json" "$(uname -m)" <<'PY'
import json
import sys

config = json.loads(sys.argv[1])
machine = sys.argv[2]
for host in config["hosts"]:
    if machine in host["aliases"]:
        print("\t".join((host["host"], host["facts_machine"], host["file_arch"], host["elf_arch"], host["interpreter"])))
        raise SystemExit(0)
raise SystemExit(f"unsupported Linux host architecture: {machine}")
PY
}

build_machine() {
	"$python_bin" - "$config_json" "$(uname -m)" <<'PY'
import json
import sys

config = json.loads(sys.argv[1])
machine = sys.argv[2]
for host in config["hosts"]:
    if machine in host["aliases"]:
        print(host["machine"])
        raise SystemExit(0)
raise SystemExit(f"unsupported Linux host architecture: {machine}")
PY
}

paths_overlap() {
	[[ "$1" == "$2" || "$1" == "$2/"* || "$2" == "$1/"* ]]
}

remote_tip() {
	local name="$1" url="$2" index response direct peeled remote_commit
	for index in "${!remote_names[@]}"; do
		if [[ "${remote_names[$index]}" == "$name" ]]; then
			remote_value="${remote_values[$index]}"
			return
		fi
	done
	if ! response="$(git ls-remote --exit-code "$url" "refs/tags/$source_tag" "refs/tags/$source_tag^{}")"; then
		echo "无法查询AIDL远端标签:$name:$source_tag" >&2
		return 1
	fi
	direct="$(awk -v ref="refs/tags/$source_tag" '$2 == ref {print $1}' <<<"$response")"
	peeled="$(awk -v ref="refs/tags/$source_tag^{}" '$2 == ref {print $1}' <<<"$response")"
	remote_commit="${peeled:-$direct}"
	[[ "$remote_commit" =~ ^[0-9a-f]{40}([0-9a-f]{24})?$ ]] || {
		echo "无法确认AIDL远端标签:$name:$source_tag" >&2
		return 1
	}
	remote_names+=("$name")
	remote_values+=("$remote_commit")
	remote_value="$remote_commit"
}

canon_dir() {
	local path="$1" lexical physical
	install -d -m 0755 "$path"
	lexical="$(realpath -m -s -- "$path")"
	physical="$(realpath -e -- "$path")"
	[[ "$lexical" == "$physical" && -d "$physical" && ! -L "$physical" ]] || {
		echo "AIDL工作目录必须是无符号链接的普通目录:$path" >&2
		return 1
	}
	printf '%s\n' "$physical"
}

managed_mark() {
	local root="$1"
	[[ -f "$root/build-facts.json" && ! -L "$root/build-facts.json" ]] || return 1
	"$python_bin" - "$root/build-facts.json" "$schema" "$tool_name" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as stream:
    facts = json.load(stream)
if facts.get("schema") != sys.argv[2] or facts.get("tool") != sys.argv[3]:
    raise SystemExit(1)
PY
}

prep_paths() {
	local source_base dest_parent destination_name script_root protected
	cache_root="$(canon_dir "$cache_root")"
	source_base="${source_root:-$cache_root/sources/$source_tag}"
	source_root="$(canon_dir "$source_base")"
	dest_parent="$(dirname -- "$destination")"
	destination_name="$(basename -- "$destination")"
	[[ -n "$destination_name" && "$destination_name" != "." && "$destination_name" != "/" ]] || {
		echo "AIDL输出目录名无效:$destination" >&2
		return 1
	}
	dest_parent="$(canon_dir "$dest_parent")"
	destination="$dest_parent/$destination_name"
	script_root="$(realpath -e -- "$script_dir")"
	for protected in "$cache_root" "$source_root" "$script_root"; do
		if paths_overlap "$destination" "$protected"; then
			echo "AIDL输出目录与受保护目录重叠:$destination:$protected" >&2
			return 1
		fi
	done
	dest_identity=""
	if [[ -e "$destination" || -L "$destination" ]]; then
		[[ -d "$destination" && ! -L "$destination" ]] || {
			echo "AIDL正式输出不是普通目录:$destination" >&2
			return 1
		}
		managed_mark "$destination" || {
			echo "AIDL正式输出不属于本构建器:$destination" >&2
			return 1
		}
		dest_identity="$(stat -Lc '%d:%i' -- "$destination")"
	fi
}

print_meta() {
	printf '%s\n' "$config_json"
}

print_plan() {
	"$python_bin" - "$config_json" "$1" <<'PY'
import json
import sys

config = json.loads(sys.argv[1])
for host in config["hosts"]:
    if host["host"] == sys.argv[2]:
        json.dump(
            {
                "host": host["host"],
                "machine": host["machine"],
                "compiler": config["cxx_compiler"],
                "linker": config["linker"],
                "default_cache": config["default_cache"],
            },
            sys.stdout,
            separators=(",", ":"),
        )
        sys.stdout.write("\n")
        raise SystemExit(0)
raise SystemExit(f"unsupported build host: {sys.argv[2]}")
PY
}

print_defaults() {
	local machine
	[[ "$(uname -s)" == Linux ]] || {
		echo "AIDL默认原生路径只支持Linux" >&2
		return 3
	}
	machine="$(build_machine)"
	[[ -n "$machine" ]] || {
		echo "AIDL宿主事实不完整" >&2
		return 3
	}
	printf '%s\n' "$project_root/$default_cache/output/$machine"
	printf '%s\n' "$project_root/$default_cache/sources/$source_tag"
}

check_llvm_tools() {
	local tool host machine file_arch expected interp executable
	[[ -n "$toolchain_root" && "$toolchain_root" == /* && ! -L "$toolchain_root" && -d "$toolchain_root" ]] || {
		echo "AIDL构建缺少预编译工具链根:CDH_TOOLCHAIN_ROOT" >&2
		exit 3
	}
	[[ "$go_bin" == "$toolchain_root/go126/bin/go" && -x "$go_bin" ]] || {
		echo "AIDL构建Go必须来自预编译工具链:$go_bin" >&2
		exit 3
	}
	c_compiler="$toolchain_root/llvm22/bin/clang"
	cxx_compiler="$toolchain_root/llvm22/bin/clang++"
	lld_linker="$toolchain_root/llvm22/bin/ld.lld"
	for tool in "$c_compiler" "$cxx_compiler" "$lld_linker" file; do need_cmd "$tool"; done
	[[ "$(uname -s)" == Linux ]] || { echo "AIDL原生构建只支持Linux" >&2; exit 3; }
	IFS=$'\t' read -r host machine file_arch expected interp < <(host_info)
	[[ -n "$host" && -n "$machine" && -n "$file_arch" && -n "$expected" && -n "$interp" ]] || { echo "AIDL宿主事实不完整" >&2; exit 3; }
	[[ "$toolchain_root" == "$(fixed_tc_root "${host/\//-}")" ]] || {
		echo "AIDL构建工具链偏离项目固定安装根:$toolchain_root" >&2
		exit 3
	}
	"$c_compiler" --version | grep -Eq "([Cc]lang|LLVM) .*${compiler_major}(\\.|$)" || { echo "需要Clang主版本$compiler_major" >&2; exit 3; }
	"$cxx_compiler" --version | grep -Eq "([Cc]lang|LLVM) .*${compiler_major}(\\.|$)" || { echo "需要Clang主版本$compiler_major" >&2; exit 3; }
	"$lld_linker" --version | grep -Eq "LLD .*${linker_major}(\\.|$)" || { echo "需要LLD主版本$linker_major" >&2; exit 3; }
	for executable in "$c_compiler" "$cxx_compiler" "$lld_linker"; do
		file -L "$(command -v "$executable")" | grep -q "ELF 64-bit.*$file_arch" || {
			echo "工具宿主架构与Linux宿主不一致:$executable:$machine" >&2
			exit 3
		}
	done
}

check_tools() {
	local tool
	check_llvm_tools
	for tool in cmake ninja bison flex git readelf ldd sha256sum install realpath stat mv; do need_cmd "$tool"; done
}

check_verify() {
	local tool
	for tool in file readelf ldd sha256sum mktemp rm; do need_cmd "$tool"; done
	[[ "$(uname -s)" == Linux ]] || { echo "AIDL原生校验只支持Linux" >&2; exit 3; }
	host_info >/dev/null
}

clone_src() {
	local root="$1" index name url target temp tag remote
	for index in "${!source_names[@]}"; do
		name="${source_names[$index]}"
		url="${source_remotes[$index]}"
		target="$root/$name"
		if [[ -e "$target" || -L "$target" ]]; then
			[[ -d "$target" && ! -L "$target" && -d "$target/.git" && ! -L "$target/.git" ]] || {
				echo "AIDL源码路径已存在且不是普通Git目录:$target" >&2
				return 1
			}
			tag="$(git -C "$target" describe --exact-match --tags HEAD 2>/dev/null || true)"
			remote="$(git -C "$target" remote get-url origin 2>/dev/null || true)"
			[[ "$tag" == "$source_tag" && "$remote" == "$url" ]] || {
				echo "AIDL源码路径与固定来源不一致:$target" >&2
				return 1
			}
			continue
		fi
		temp="$(mktemp -d "$root/.${name}.clone.XXXXXX")"
		rmdir -- "$temp"
		if ! git clone --depth 1 --branch "$source_tag" --config advice.detachedHead=false "$url" "$temp"; then
			rm -rf -- "$temp"
			return 1
		fi
		if ! mv -T -n -- "$temp" "$target"; then
			rm -rf -- "$temp"
			return 1
		fi
		if [[ -e "$temp" || -L "$temp" ]]; then
			echo "AIDL源码目标在克隆期间被占用:$target" >&2
			rm -rf -- "$temp"
			return 1
		fi
	done
}

check_sources() {
	local root="$1" index name url target tag remote local_commit remote_commit
	for index in "${!source_names[@]}"; do
		name="${source_names[$index]}"
		url="${source_remotes[$index]}"
		target="$root/$name"
		[[ -d "$target" && ! -L "$target" && -d "$target/.git" && ! -L "$target/.git" ]] || {
			echo "AIDL源码仓无效:$target" >&2
			return 1
		}
		tag="$(git -C "$target" describe --exact-match --tags HEAD 2>/dev/null || true)"
		remote="$(git -C "$target" remote get-url origin 2>/dev/null || true)"
		local_commit="$(git -C "$target" rev-parse HEAD 2>/dev/null || true)"
		remote_tip "$name" "$url"
		remote_commit="$remote_value"
		[[ "$tag" == "$source_tag" && "$remote" == "$url" ]] || {
			echo "AIDL源码仓未固定到登记来源:$target" >&2
			return 1
		}
		[[ "$local_commit" == "$remote_commit" ]] || {
			echo "AIDL源码仓标签与远端正式标签不一致:$target" >&2
			return 1
		}
		[[ -z "$(git -C "$target" status --porcelain --untracked-files=all)" ]] || {
			echo "AIDL源码仓存在本地改动:$target" >&2
			return 1
		}
	done
}

codegen_probe() {
	local root="$1" input generated
	input="$root/probe/$probe_input"
	[[ -f "$input" && ! -L "$input" ]] || {
		echo "AIDL Java代码生成输入无效:$input" >&2
		return 1
	}
	probe_temp="$(CDH_TARGET=common "$project_root/build/cwd-temp.sh" new aidl-codegen)"
	generated="$probe_temp/$probe_output"
	"$root/aidl" --lang=java -o "$probe_temp" -I"$root/probe/$probe_import" "$input"
	[[ -s "$generated" && -f "$generated" && ! -L "$generated" ]] || {
		echo "AIDL Java代码生成探针失败:$generated" >&2
		return 1
	}
	probe_digest="$("$python_bin" - "$generated" <<'PY'
import hashlib
import sys

with open(sys.argv[1], "rb") as stream:
    lines = stream.readlines()
kept = [line for line in lines if not line.startswith(b" * Using: ")]
if len(lines) - len(kept) != 1:
    raise SystemExit("AIDL code generation command line is missing or repeated")
print(hashlib.sha256(b"".join(kept)).hexdigest())
PY
)"
	rm -rf -- "$probe_temp"
	probe_temp=""
}

verify_bin() {
	local root="$1" executable="$1/aidl" host machine file_arch expected interp help_text
	[[ -d "$root" && ! -L "$root" && -f "$executable" && ! -L "$executable" && -x "$executable" ]] || { echo "AIDL输出目录无效:$root" >&2; return 1; }
	IFS=$'\t' read -r host machine file_arch expected interp < <(host_info)
	[[ -n "$host" && -n "$machine" && -n "$file_arch" && -n "$expected" && -n "$interp" ]] || { echo "AIDL宿主事实不完整" >&2; return 1; }
	file -L "$executable" | grep -q "ELF 64-bit.*$expected" || { echo "AIDL输出宿主架构错误:$executable" >&2; return 1; }
	readelf -l "$executable" | grep -q "$interp" || { echo "AIDL输出解释器错误:$executable" >&2; return 1; }
	readelf -p .comment "$executable" | grep -Eq "Clang version ${compiler_major}|clang version ${compiler_major}" || { echo "AIDL输出未记录Clang $compiler_major:$executable" >&2; return 1; }
	readelf -p .comment "$executable" | grep -Eq "LLD ${linker_major}|LLD version ${linker_major}" || { echo "AIDL输出未记录LLD $linker_major:$executable" >&2; return 1; }
	if ldd "$executable" | grep -q 'not found'; then echo "AIDL输出存在缺失动态库:$executable" >&2; return 1; fi
	help_text="$($executable --help 2>&1)"
	grep -Eq "AIDL Compiler: built for platform SDK version ${platform_sdk}([[:space:]]|$)" <<<"$help_text" || { echo "AIDL输出未报告SDK $platform_sdk:$executable" >&2; return 1; }
}

write_facts() {
	local root="$1" build_root="$2" digest host machine compiler linker compiler_path linker_path metadata
	digest="$(sha256sum "$root/aidl" | awk '{print $1}')"
	IFS=$'\t' read -r host machine _ < <(host_info)
	compiler="$("$cxx_compiler" --version | head -1)"
	linker="$("$lld_linker" --version | head -1)"
	compiler_path="$(realpath "$(command -v "$cxx_compiler")")"
	linker_path="$(realpath "$(command -v "$lld_linker")")"
	metadata="$(print_meta)"
	codegen_probe "$root"
	"$python_bin" - "$metadata" "$host" "$machine" "$compiler" "$linker" "$compiler_path" "$linker_path" "$digest" "$build_root/compile_commands.json" "$probe_digest" >"$root/build-facts.json" <<'PY'
import json
import shlex
import sys

metadata = json.loads(sys.argv[1])
with open(sys.argv[9], "r", encoding="utf-8") as stream:
    commands = json.load(stream)
matches = [entry for entry in commands if entry.get("file", "").endswith("/libbase/posix_strerror_r.cpp")]
if len(matches) != 1:
    raise SystemExit("AIDL POSIX compile command is not unique")
arguments = matches[0].get("arguments") or shlex.split(matches[0]["command"])
expected_flags = [
    "-D_GNU_SOURCE",
    "-U_GNU_SOURCE",
    "-U_DEFAULT_SOURCE",
    "-D_POSIX_C_SOURCE=200809L",
]
compile_flags = [argument for argument in arguments if argument in expected_flags]
if compile_flags != expected_flags:
    raise SystemExit("AIDL POSIX compile flags are incomplete or unordered")
json.dump(
    {
        "schema": metadata["schema"],
        "tool": metadata["tool"],
        "android_tag": metadata["android_tag"],
        "platform_sdk": metadata["platform_sdk"],
        "host": sys.argv[2],
        "host_machine": sys.argv[3],
        "compiler": sys.argv[4],
        "compiler_path": sys.argv[6],
        "compiler_major": metadata["compiler_major"],
        "linker": sys.argv[5],
        "linker_path": sys.argv[7],
        "linker_major": metadata["linker_major"],
        "aidl_sha256": sys.argv[8],
        "posix_compile_flags": compile_flags,
        "codegen_language": "java",
        "codegen_source": metadata["probe_input"],
        "codegen_sha256": sys.argv[10],
        "source_repositories": metadata["source_repositories"],
    },
    sys.stdout,
    ensure_ascii=True,
    separators=(",", ":"),
)
sys.stdout.write("\n")
PY
}

verify_aidl() {
	local root="$1" host machine digest metadata
	[[ -d "$root" && ! -L "$root" && -f "$root/aidl" && ! -L "$root/aidl" && -f "$root/build-facts.json" && ! -L "$root/build-facts.json" ]] || {
		echo "AIDL构建事实无效:$root/build-facts.json" >&2
		return 1
	}
	verify_bin "$root"
	codegen_probe "$root"
	IFS=$'\t' read -r host machine _ < <(host_info)
	digest="$(sha256sum "$root/aidl" | awk '{print $1}')"
	metadata="$(print_meta)"
	"$python_bin" - "$root/build-facts.json" "$metadata" "$host" "$machine" "$digest" "$probe_digest" <<'PY'
import json
import re
import sys

with open(sys.argv[1], "r", encoding="utf-8") as stream:
    facts = json.load(stream)
metadata = json.loads(sys.argv[2])
expected = {
    "schema": metadata["schema"],
    "tool": metadata["tool"],
    "android_tag": metadata["android_tag"],
    "platform_sdk": metadata["platform_sdk"],
    "host": sys.argv[3],
    "host_machine": sys.argv[4],
    "compiler_major": metadata["compiler_major"],
    "linker_major": metadata["linker_major"],
    "aidl_sha256": sys.argv[5],
}
for key, value in expected.items():
    if facts.get(key) != value:
        raise SystemExit(f"AIDL build fact mismatch: {key}")
if str(metadata["compiler_major"]) not in facts.get("compiler", "") or str(metadata["linker_major"]) not in facts.get("linker", ""):
    raise SystemExit("AIDL toolchain facts do not report the configured major versions")
if not facts.get("compiler_path", "").startswith("/") or not facts.get("linker_path", "").startswith("/"):
    raise SystemExit("AIDL toolchain paths are incomplete")
if facts.get("posix_compile_flags") != ["-D_GNU_SOURCE", "-U_GNU_SOURCE", "-U_DEFAULT_SOURCE", "-D_POSIX_C_SOURCE=200809L"]:
    raise SystemExit("AIDL POSIX compile facts are incomplete")
if facts.get("codegen_language") != "java" or facts.get("codegen_source") != metadata["probe_input"]:
    raise SystemExit("AIDL code generation facts are incomplete")
if re.fullmatch(r"[0-9a-f]{64}", facts.get("codegen_sha256", "")) is None:
    raise SystemExit("AIDL code generation digest is invalid")
if facts.get("codegen_sha256") != sys.argv[6]:
    raise SystemExit("AIDL code generation digest does not match runtime output")
repos = facts.get("source_repositories")
if repos != metadata["source_repositories"]:
    raise SystemExit("AIDL source repository facts do not match metadata")
PY
}

publish_out() {
	local build_root="$1" out_dir="$2" source_base="$3" parent base backup current_identity
	parent="$(dirname -- "$out_dir")"
	base="$(basename -- "$out_dir")"
	[[ "$base" != "." && "$base" != "/" && -d "$parent" && ! -L "$parent" ]] || {
		echo "AIDL输出父目录无效:$parent" >&2
		return 1
	}
	publish_target="$out_dir"
	publish_temp="$(mktemp -d "$parent/.${base}.new.XXXXXX")"
	install -m 0755 "$build_root/aidl" "$publish_temp/aidl"
	verify_bin "$publish_temp"
	install -d -m 0755 "$(dirname -- "$publish_temp/probe/$probe_input")"
	install -m 0644 "$source_base/aidl/$probe_input" "$publish_temp/probe/$probe_input"
	write_facts "$publish_temp" "$build_root"
	verify_aidl "$publish_temp"
	if [[ -e "$out_dir" || -L "$out_dir" ]]; then
		[[ -n "$dest_identity" && -d "$out_dir" && ! -L "$out_dir" ]] || {
			echo "AIDL正式输出不是普通目录:$out_dir" >&2
			return 1
		}
		current_identity="$(stat -Lc '%d:%i' -- "$out_dir")"
		[[ "$current_identity" == "$dest_identity" ]] || {
			echo "AIDL正式输出在构建期间被替换:$out_dir" >&2
			return 1
		}
		verify_aidl "$out_dir"
		backup="$(mktemp -d "$parent/.${base}.old.XXXXXX")"
		rmdir -- "$backup"
		publish_backup="$backup"
		if ! mv -T -- "$out_dir" "$backup"; then
			return 1
		fi
	elif [[ -n "$dest_identity" ]]; then
		echo "AIDL正式输出在构建期间消失:$out_dir" >&2
		return 1
	fi
	if ! mv -T -n -- "$publish_temp" "$out_dir"; then
		return 1
	fi
	if [[ -e "$publish_temp" || -L "$publish_temp" ]]; then
		echo "AIDL正式输出在发布期间被占用:$out_dir" >&2
		return 1
	fi
	publish_temp=""
	publish_done=1
	if [[ -n "$publish_backup" ]] && ! rm -rf -- "$publish_backup"; then
		echo "AIDL已发布但旧输出未清理:$backup" >&2
		return 4
	fi
	publish_backup=""
	publish_target=""
}

build_aidl() {
	local source_base
	check_tools
	prep_paths
	if [[ -n "$dest_identity" ]]; then verify_aidl "$destination"; fi
	source_base="$source_root"
	build_work="$(mktemp -d "$cache_root/build.XXXXXX")"
	clone_src "$source_base"
	check_sources "$source_base"
	cmake -S "$script_dir/aidl36" -B "$build_work" -G Ninja \
		-DCMAKE_BUILD_TYPE=Release \
		-DCMAKE_EXPORT_COMPILE_COMMANDS=ON \
		-DCMAKE_C_COMPILER="$(command -v "$c_compiler")" \
		-DCMAKE_CXX_COMPILER="$(command -v "$cxx_compiler")" \
		-DCMAKE_EXE_LINKER_FLAGS="-fuse-ld=$(command -v "$lld_linker")" \
		-DCMAKE_CXX_FLAGS='-stdlib=libstdc++' \
		-DCDH_PLATFORM_SDK="$platform_sdk" \
		-DAIDL_SOURCE_ROOT="$source_base"
	cmake --build "$build_work" --target aidl --parallel "${CDH_AIDL_JOBS:-$(nproc)}"
	check_sources "$source_base"
	publish_out "$build_work" "$destination" "$source_base"
}

check_config_go
check_config_py
load_config

while [[ "$#" -gt 0 ]]; do
	case "$1" in
		--print-metadata) [[ "$#" == 1 ]] || { usage >&2; exit 2; }; print_meta; exit 0 ;;
		--print-defaults) [[ "$#" == 1 ]] || { usage >&2; exit 2; }; print_defaults; exit 0 ;;
		--print-plan)
			shift
			if [[ "${1:-}" == "--host" ]]; then shift; fi
			[[ "$#" == 1 ]] || { usage >&2; exit 2; }
			print_plan "$1"
			exit 0
			;;
		--destination) destination="${2:-}"; shift 2 ;;
		--source-dir) source_root="${2:-}"; shift 2 ;;
		--cache-dir) cache_root="${2:-}"; shift 2 ;;
		--verify) verify_root="${2:-}"; shift 2 ;;
		--check-paths) check_paths=1; shift ;;
		--check-toolchain) check_toolchain=1; shift ;;
		-h|--help) usage; exit 0 ;;
		*) echo "未知参数:$1" >&2; usage >&2; exit 2 ;;
	esac
done

if [[ -n "$verify_root" ]]; then
	((check_paths == 0)) || { echo "--verify不能与--check-paths同时使用" >&2; exit 2; }
	check_verify
	verify_aidl "$verify_root"
	echo "==> Linux原生AIDL校验通过:$verify_root"
	exit 0
fi
if ((check_toolchain == 1)); then
	((check_paths == 0)) || { echo "--check-toolchain不能与--check-paths同时使用" >&2; exit 2; }
	check_llvm_tools
	echo "==> AIDL预编译LLVM 22检查通过:$toolchain_root"
	exit 0
fi
[[ -n "$destination" ]] || { echo "缺少--destination" >&2; usage >&2; exit 2; }
if ((check_paths == 1)); then
	for command_name in install realpath stat; do need_cmd "$command_name"; done
	prep_paths
	echo "==> AIDL目录检查通过:$destination"
	exit 0
fi
build_aidl

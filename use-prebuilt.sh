#!/bin/sh
if [ "${BASH_SOURCE:-$0}" = "$0" ] && [ "${BASH:-bootstrap}" != "${CDH_BASH_BIN:-}" ]; then
	exec "${CDH_BASH_BIN:-$(dirname "$0")/bash.sh}" "$0" "$@"
fi
set -euo pipefail

usage() {
  printf '%s\n' 'usage: use-prebuilt.sh <install|check|env|path|require|require-env|revision> [--host HOST] [--repo-root DIR] [--dest DIR] [--print-plan] [--export] [--tool NAME]' >&2
  printf '%s\n' 'tools: bash python go java javac bazel cargo rustc rustdoc clang clang++ ld lld ar readelf' >&2
  exit 2
}

fail() {
  printf '预编译工具链错误:%s\n' "$1" >&2
  exit 2
}

write_env() {
  if [[ "$command_name" == path ]]; then
    if [[ "$1" == "$tool_key" ]]; then
      printf '%s\n' "$2$tool_suffix"
    fi
    return 0
  fi
  local prefix=""
  if (( export_env )); then
    prefix="export "
  fi
  printf "%s%s='%s'\n" "$prefix" "$1" "$2"
}

check_git_index() {
  local repository="$1"
  local record tag
  while IFS= read -r -d '' record; do
    [[ "${#record}" -ge 2 && "${record:1:1}" == " " ]] || fail "无法解析组件仓Git索引标记"
    tag="${record:0:1}"
    if [[ "$tag" == S || "$tag" =~ [a-z] ]]; then
      fail "组件仓Git索引包含隐藏标记:${repository}"
    fi
  done < <(git -C "$repository" ls-files -v -z)
}

detect_host() {
  local system machine
  system="$(uname -s)"
  machine="$(uname -m)"
  case "${system}/${machine}" in
    Darwin/arm64) printf '%s\n' darwin-arm64 ;;
    Linux/arm64|Linux/aarch64) printf '%s\n' linux-arm64 ;;
    Linux/amd64|Linux/x86_64) printf '%s\n' linux-amd64 ;;
    *) fail "不支持的宿主:${system}/${machine}" ;;
  esac
}

detect_native() {
  local system machine
  system="$(/usr/bin/uname -s)"
  machine="$(/usr/bin/uname -m)"
  case "${system}/${machine}" in
    Darwin/arm64) printf '%s\n' darwin-arm64 ;;
    Linux/arm64|Linux/aarch64) printf '%s\n' linux-arm64 ;;
    Linux/amd64|Linux/x86_64) printf '%s\n' linux-amd64 ;;
    *) fail "不支持的宿主:${system}/${machine}" ;;
  esac
}

if (( $# == 0 )); then
  usage
fi

command_name="$1"
shift
case "$command_name" in
  install|check|env|path|require|require-env|revision) ;;
  *) usage ;;
esac

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
project_dir="$(cd "${script_dir}/.." && pwd -P)"
host=""
repo_root="$script_dir"
dest=""
print_plan=0
export_env=0
has_tool=0
tool=""
tool_key=""
tool_suffix=""

while (( $# > 0 )); do
  case "$1" in
    --host)
      (( $# >= 2 )) || usage
      host="$2"
      shift 2
      ;;
    --repo-root)
      (( $# >= 2 )) || usage
      repo_root="$2"
      shift 2
      ;;
    --dest)
      (( $# >= 2 )) || usage
      dest="$2"
      shift 2
      ;;
    --print-plan)
      print_plan=1
      shift
      ;;
    --export)
      export_env=1
      shift
      ;;
    --tool)
      (( $# >= 2 )) || usage
      has_tool=1
      tool="$2"
      shift 2
      ;;
    *) usage ;;
  esac
done

if (( export_env )) && [[ "$command_name" != env ]]; then
  fail "--export只适用于env"
fi
if (( has_tool )) && [[ "$command_name" != path ]]; then
  fail "--tool只适用于path"
fi
if [[ "$command_name" == path ]]; then
  case "$tool" in
    bash) tool_key=CDH_BASH_BIN ;;
    python) tool_key=CDH_PYTHON_BIN ;;
    go) tool_key=GO_BIN ;;
    java) tool_key=JAVA_HOME; tool_suffix=/bin/java ;;
    javac) tool_key=JAVA_HOME; tool_suffix=/bin/javac ;;
    bazel) tool_key=BAZEL_BIN ;;
    cargo) tool_key=CARGO_BIN ;;
    rustc) tool_key=RUSTC ;;
    rustdoc) tool_key=RUSTDOC ;;
    clang) tool_key=CDH_ANDROID_CLANG ;;
    clang++) tool_key=CDH_ANDROID_CXX ;;
    ld) tool_key=CDH_HOST_LD ;;
    lld) tool_key=CDH_ANDROID_LLD ;;
    ar) tool_key=CDH_ANDROID_AR ;;
    readelf) tool_key=CDH_ANDROID_READELF ;;
    "") fail "path需要--tool" ;;
    *) fail "不支持的工具:${tool}" ;;
  esac
fi

if [[ -z "$host" ]]; then
  host="$(detect_host)"
fi
case "$host" in
  darwin-arm64|linux-arm64|linux-amd64) ;;
  *) fail "不支持的宿主目录:${host}" ;;
esac
inside_container=0
if [[ -f /.dockerenv || -f /run/.containerenv ]]; then
  inside_container=1
fi
if [[ -z "$dest" ]]; then
  dest="${CDH_TOOLCHAIN_ROOT:-${project_dir}/output/toolchains/${host}}"
fi

current_host=""
if [[ "$command_name" == require || "$command_name" == require-env ]]; then
  current_host="$(detect_native)"
  canonical_repo="$script_dir"
  canonical_dest="${project_dir}/output/toolchains/${current_host}"
  if (( inside_container )); then
    canonical_dest="/opt/cloud-droid-toolchains/${current_host}"
  fi
  if [[ -n "${CDH_TOOLCHAIN_ROOT:-}" ]]; then
    canonical_dest="$CDH_TOOLCHAIN_ROOT"
  fi
  [[ "$host" == "$current_host" ]] || fail "构建宿主目录被覆盖:${host}:${current_host}"
  [[ "$repo_root" == "$canonical_repo" ]] || fail "构建组件目录被覆盖:${repo_root}:${canonical_repo}"
  [[ "$dest" == "$canonical_dest" ]] || fail "构建安装目录被覆盖:${dest}:${canonical_dest}"
  host_ld="${canonical_dest}/llvm22/bin/ld.lld"
  if [[ "$current_host" == darwin-arm64 ]]; then
    host_ld="${canonical_dest}/llvm22/bin/ld64.lld"
  fi
  check_path() {
    local name="$1" expected="$2" actual
    actual="${!name:-}"
    [[ "$actual" == "$expected" ]] || fail "构建工具路径被覆盖:${name}:${actual}:${expected}"
  }
  check_path GO_BIN "${canonical_dest}/go126/bin/go"
  check_path CDH_BASH_BIN "${canonical_dest}/bash53/bin/bash"
  if [[ -n "${CDH_PYTHON_BIN:-}" ]]; then
    check_path CDH_PYTHON_BIN "${canonical_dest}/python314/bin/python3.14"
  fi
  [[ "${GOTOOLCHAIN:-}" == local ]] || fail "Go工具链选择被覆盖:${GOTOOLCHAIN:-}:local"
  check_path JAVA_HOME "${canonical_dest}/jdk21"
  check_path BAZEL_BIN "${canonical_dest}/bazel851/bin/bazel"
  check_path CDH_RUST_ROOT "${canonical_dest}/rust197"
  check_path CARGO_BIN "${canonical_dest}/rust197/bin/cargo"
  check_path RUSTC "${canonical_dest}/rust197/bin/rustc"
  check_path RUSTDOC "${canonical_dest}/rust197/bin/rustdoc"
  check_path ANDROID_NDK_HOME "${canonical_dest}/ndk29"
  check_path ANDROID_NDK_ROOT "${canonical_dest}/ndk29"
  check_path NDK_HOME "${canonical_dest}/ndk29"
  # Android SDK组件由宿主justfile导出；容器镜像环境未登记该组件，保持原有检查范围。
  if (( ! inside_container )); then
    check_path ANDROID_HOME "${canonical_dest}/sdkapp37"
    check_path ANDROID_SDK_ROOT "${canonical_dest}/sdkapp37"
  fi
  android_bin="${canonical_dest}/llvm22/bin"
  if (( inside_container )); then
    android_bin="${canonical_dest}/selected/bin"
  fi
  check_path CDH_ANDROID_CLANG "${android_bin}/clang"
  check_path CDH_ANDROID_CXX "${android_bin}/clang++"
  check_path CDH_ANDROID_LLD "${android_bin}/ld.lld"
  check_path CDH_ANDROID_READELF "${canonical_dest}/llvm22/bin/llvm-readelf"
  check_path CDH_ANDROID_AR "${canonical_dest}/llvm22/bin/llvm-ar"
  check_path CDH_HOST_LD "$host_ld"
  check_path CDH_TOOLCHAIN_ROOT "$canonical_dest"
  path_prefix="${canonical_dest}/go126/bin:${canonical_dest}/jdk21/bin:${canonical_dest}/llvm22/bin:${canonical_dest}/bazel851/bin:${canonical_dest}/rust197/bin:${canonical_dest}/bash53/bin"
  case "${PATH:-}" in
    "${path_prefix}:"*) ;;
    *) fail "构建工具PATH前缀被覆盖:${PATH:-}:${path_prefix}" ;;
  esac
fi

[[ -d "$repo_root" ]] || fail "组件仓目录不存在:${repo_root}"
repo="${repo_root}/${host}"
[[ -d "$repo" ]] || fail "宿主目录不存在:${repo}"
installer="${repo}/bin/cdh-toolchain"
# 源码与只读安装器允许正常链接，目录、文件及执行检查核对实际目标。
[[ -f "$installer" ]] || fail "安装器不存在:${installer}"
[[ -x "$installer" ]] || fail "安装器不可执行:${installer}"
[[ ! -L "$dest" ]] || fail "目标目录不能是符号链接:${dest}"
if [[ -e "$dest" && ! -d "$dest" ]]; then
  fail "目标路径不是目录:${dest}"
fi

if [[ "$command_name" == revision ]]; then
  # 打包源码没有Git元数据时没有可核对的提交，改用组件仓登记版本标识工具链。
  # 判定按“该目录本身是不是Git工作树根”，不接受落到上级仓的结果。
  probe="${project_dir}/tools/build/git-worktree.sh"
  [[ -x "$probe" ]] || fail "缺少Git工作树判定工具:${probe}"
  if ! command -v git >/dev/null 2>&1 || ! "$probe" "$repo_root"; then
    marker="${repo_root}/version.go"
    [[ -f "$marker" ]] || fail "组件仓缺少登记版本文件:${marker}"
    pinned="$(sed -n 's/^const Value = "\(.*\)"$/\1/p' "$marker")"
    [[ -n "$pinned" ]] || fail "无法读取组件仓登记版本:${marker}"
    printf '组件仓没有Git元数据，按登记版本标识工具链:%s\n' "$pinned" >&2
    printf 'nogit-%s\n' "$pinned"
    exit 0
  fi
  command -v git >/dev/null 2>&1 || fail "缺少Git命令"
  git -C "$repo_root" rev-parse --is-inside-work-tree >/dev/null 2>&1 || fail "组件仓不是Git工作区:${repo_root}"
  [[ -z "$(git -C "$repo_root" status --porcelain --untracked-files=all)" ]] || fail "组件仓存在本地改动:${repo_root}"
	check_git_index "$repo_root"
  branch="$(git -C "$repo_root" symbolic-ref --quiet --short HEAD || true)"
  if [[ -z "$branch" ]]; then
    revision="$(git -C "$repo_root" rev-parse HEAD)"
    # submodule update检出主仓锁定对象；远端发布新版本不改变该源码身份。
    if [[ "$(cd "$repo_root" && pwd -P)" == "${project_dir}/toolchains" ]] && "$probe" "$project_dir"; then
      entry="$(git -C "$project_dir" ls-tree HEAD -- toolchains)"
      read -r mode kind pinned path <<< "$entry"
      if [[ "$mode" == 160000 && "$kind" == commit && "$path" == toolchains && "$revision" == "$pinned" ]]; then
        printf '%s\n' "$revision"
        exit 0
      fi
    fi
    upstream_rev="$(git -C "$repo_root" rev-parse origin/main)" || fail "组件仓缺少origin/main:${repo_root}"
    [[ "$revision" == "$upstream_rev" ]] || fail "detached组件仓不是origin/main:${repo_root}"
    if symbolic_main="$(git -C "$repo_root" symbolic-ref --quiet refs/heads/main 2>/dev/null)"; then
      fail "组件仓本地main不能是符号引用:${symbolic_main}"
    else
      symbolic_status=$?
      [[ "$symbolic_status" -eq 1 ]] || fail "无法检查组件仓本地main引用:${repo_root}"
    fi
    if git -C "$repo_root" show-ref --verify --quiet refs/heads/main; then
      local_main="$(git -C "$repo_root" rev-parse refs/heads/main)"
      if [[ "$local_main" != "$upstream_rev" ]]; then
        git -C "$repo_root" merge-base --is-ancestor "$local_main" "$upstream_rev" || \
          fail "组件仓本地main与origin/main未同步且不能快进:${repo_root}"
        git -C "$repo_root" update-ref --no-deref refs/heads/main "$upstream_rev" "$local_main" || \
          fail "组件仓本地main无法快进到origin/main:${repo_root}"
      fi
      git -C "$repo_root" switch --quiet main || fail "组件仓无法切换到main:${repo_root}"
    else
      git -C "$repo_root" update-ref --no-deref refs/heads/main "$upstream_rev" 0000000000000000000000000000000000000000 || \
        fail "组件仓无法创建main:${repo_root}"
      git -C "$repo_root" switch --quiet main || fail "组件仓无法切换到main:${repo_root}"
      git -C "$repo_root" branch --set-upstream-to=origin/main main >/dev/null || \
        fail "组件仓main无法设置origin/main上游:${repo_root}"
    fi
    branch=main
  fi
  [[ "$branch" == main ]] || fail "组件仓必须位于main分支:${branch}"
  upstream="$(git -C "$repo_root" rev-parse --abbrev-ref --symbolic-full-name '@{upstream}')" || fail "组件仓main缺少上游:${repo_root}"
  [[ "$upstream" == origin/main ]] || fail "组件仓main必须跟踪origin/main:${upstream}"
  revision="$(git -C "$repo_root" rev-parse HEAD)"
  upstream_rev="$(git -C "$repo_root" rev-parse '@{upstream}')"
  [[ "$revision" == "$upstream_rev" ]] || fail "组件仓main与origin/main未同步:${repo_root}"
	check_git_index "$repo_root"
  printf '%s\n' "$revision"
  exit 0
fi

if (( print_plan )); then
  printf 'command=%s\n' "$command_name"
  printf 'host=%s\n' "$host"
  printf 'repo_root=%s\n' "$repo_root"
  printf 'repo=%s\n' "$repo"
  printf 'dest=%s\n' "$dest"
  printf 'installer=%s\n' "$installer"
  if [[ "$command_name" == path ]]; then
    printf 'tool=%s\n' "$tool"
  fi
  exit 0
fi

if [[ "$command_name" == env || "$command_name" == path ]]; then
  case "$dest" in
    *"'"*|*$'\n'*) fail "环境路径含不支持的字符:${dest}" ;;
  esac
  write_env CDH_TOOLCHAIN_ROOT "${dest}"
  write_env GO_BIN "${dest}/go126/bin/go"
  write_env CDH_BASH_BIN "${dest}/bash53/bin/bash"
  write_env CDH_PYTHON_BIN "${dest}/python314/bin/python3.14"
  write_env GOTOOLCHAIN local
  write_env JAVA_HOME "${dest}/jdk21"
  write_env BAZEL_BIN "${dest}/bazel851/bin/bazel"
  write_env CDH_RUST_ROOT "${dest}/rust197"
  write_env CARGO_BIN "${dest}/rust197/bin/cargo"
  write_env RUSTC "${dest}/rust197/bin/rustc"
  write_env RUSTDOC "${dest}/rust197/bin/rustdoc"
  write_env ANDROID_NDK_HOME "${dest}/ndk29"
  write_env ANDROID_NDK_ROOT "${dest}/ndk29"
  write_env NDK_HOME "${dest}/ndk29"
  write_env ANDROID_HOME "${dest}/sdkapp37"
  write_env ANDROID_SDK_ROOT "${dest}/sdkapp37"
  android_bin="${dest}/llvm22/bin"
  if (( inside_container )); then
    android_bin="${dest}/selected/bin"
  fi
  write_env CDH_ANDROID_CLANG "${android_bin}/clang"
  write_env CDH_ANDROID_CXX "${android_bin}/clang++"
  write_env CDH_ANDROID_LLD "${android_bin}/ld.lld"
  write_env CDH_ANDROID_READELF "${dest}/llvm22/bin/llvm-readelf"
  write_env CDH_ANDROID_AR "${dest}/llvm22/bin/llvm-ar"
  if [[ "$host" == darwin-arm64 ]]; then
    write_env CDH_HOST_LD "${dest}/llvm22/bin/ld64.lld"
  else
    write_env CDH_HOST_LD "${dest}/llvm22/bin/ld.lld"
  fi
  exit 0
fi

if [[ -z "$current_host" ]]; then
  current_host="$(detect_host)"
fi
[[ "$host" == "$current_host" ]] || fail "不能在${current_host}执行${host}安装器"

if [[ "$command_name" == require-env ]]; then
  printf 'checked environment %s\n' "$dest"
  exit 0
fi

if [[ "$command_name" == check || "$command_name" == require ]]; then
  # 安装程序只在install阶段修正挂载点模式，check失败时给出可执行的下一步
  if ! /usr/bin/env GOTOOLCHAIN=local "$installer" check --dest "$dest"; then
    printf '工具链检查未通过:%s\n' "$dest" >&2
    printf '把仓库放到新磁盘或新路径后先执行just toolchain-init，安装阶段会按登记归档补齐组件并修正安装根所在挂载点的目录模式\n' >&2
    exit 1
  fi
  exit 0
fi

if [[ -d "$dest" ]] && GOTOOLCHAIN=local "$installer" check --dest "$dest"; then
  printf 'using %s\n' "$dest"
  exit 0
fi

exec /usr/bin/env GOTOOLCHAIN=local "$installer" install --repo "$repo" --dest "$dest"

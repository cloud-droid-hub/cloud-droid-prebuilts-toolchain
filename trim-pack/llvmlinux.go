// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"fmt"
	"strings"
)

func pkgSource(pkg debPkg, host string) (debFile, string, string) {
	file, arch := pkg.ARM, "arm64"
	if host == "linux-amd64" {
		file, arch = pkg.AMD, "amd64"
	}
	ver := pkg.Ver
	if file.Ver != "" {
		ver = file.Ver
	}
	plain := ver
	if index := strings.IndexByte(plain, ':'); index >= 0 {
		plain = plain[index+1:]
	}
	return file, ver, pkg.Name + "_" + plain + "_" + arch + ".deb"
}

func llvmLinux(host string) spec {
	plan := spec{Host: host, Comp: "llvm22", Prefix: "llvm-22.1.2-" + host, Patch: true,
		Links: map[string]string{"bin/ld.lld": "lld"}}
	plan.Drop = []string{"lib/cdh/libbrotlienc.so.1", "lib/cdh/libbrotlienc.so.1.2.0"}
	arch, base, edit := "aarch64-linux-gnu", "https://snapshot.ubuntu.com/ubuntu/20260915T000000Z/pool/", "libedit.so.2.0.76"
	plan.OldSHA = "18374afd0db2a4a868d51f21f7aa1cc2646de41fec4e94b4e8e955575cc17027"
	if host == "linux-amd64" {
		arch, edit = "x86_64-linux-gnu", "libedit.so.2.0.72"
		plan.OldSHA = "55f4bf7ce313c0f4af2ad36152086686ad3e076b7bbfbc27bb0381118b031517"
		plan.ForceR = true
		plan.PreRPath = map[string]string{
			"lib/cdh/libLLVM.so.22.1":      "$ORIGIN/cdh",
			"lib/cdh/libclang-22.so.22":    "$ORIGIN/cdh",
			"lib/cdh/libclang-cpp.so.22.1": "$ORIGIN/cdh",
		}
		plan.RunPaths = map[string]bool{
			"lib/cdh/libcrypto.so.3": true, "lib/cdh/libssl.so.3": true,
			"lib/cdh/libedit.so.2.0.72": true, "lib/cdh/libgnutls.so.30.37.1": true,
		}
	}
	plan.Inputs = append(plan.Inputs, input{Name: "patchelf-0.18.0.tar.gz", Kind: "tar.gz",
		SHA256: "64de10e4c6b8b8379db7e87f58030f336ea747c0515f381132e810dbf84a86e7",
		URL:    "https://github.com/NixOS/patchelf/releases/download/0.18.0/patchelf-0.18.0.tar.gz"})
	for _, pkg := range llvmDebs {
		file, _, name := pkgSource(pkg, host)
		if file.Size == 0 {
			continue
		}
		in := input{Name: name, SHA256: file.SHA, Kind: "deb", URL: base + pkg.Pool + "/" + name}
		if pkg.Name == "libz3-4" {
			base := "https://archive.ubuntu.com/ubuntu/pool/"
			if host == "linux-arm64" {
				base = "https://ports.ubuntu.com/ubuntu-ports/pool/"
			}
			in.URL = base + pkg.Pool + "/" + name
		}
		if pkg.Name == "libpfm4" && host == "linux-amd64" {
			in.URL = "https://archive.ubuntu.com/ubuntu/pool/" + pkg.Pool + "/" + name
		}
		if strings.HasSuffix(pkg.Pool, "llvm-toolchain-22") {
			for _, tool := range llvmTools {
				in.Rules = append(in.Rules, rule{From: "usr/lib/llvm-22/bin/" + tool, To: "bin/" + tool})
			}
			for _, dir := range []string{"include", "lib", "share"} {
				in.Rules = append(in.Rules, rule{From: "usr/lib/llvm-22/" + dir, To: dir, Tree: true})
			}
		}
		in.Rules = append(in.Rules, rule{From: "usr/lib/" + arch, To: "lib/cdh", Glob: "*.so*"})
		if pkg.Lic != "" {
			in.Rules = append(in.Rules, rule{From: "usr/share/doc/" + pkg.Name + "/copyright", To: "lib/cdh/LICENSES/" + pkg.Lic + "-copyright"})
		}
		plan.Inputs = append(plan.Inputs, in)
	}
	plan.Aliases = map[string]string{"lib/libffi.so.8": "lib/cdh/libffi.so.8.2.0", "lib/libedit.so.2": "lib/cdh/" + edit,
		"lib/libz3.so.4": "lib/cdh/libz3.so.4", "lib/libxml2.so.16": "lib/cdh/libxml2.so.16.1.2",
		"lib/libLLVM-22.so": "lib/cdh/libLLVM.so.22.1", "lib/libLLVM.so.22.1": "lib/cdh/libLLVM.so.22.1"}
	if host == "linux-amd64" {
		plan.Drop = append(plan.Drop, "lib/cdh/libLLVM-22.so", "lib/cdh/libclang-22.so.1", "lib/libclang-22.so.1")
		plan.Aliases["lib/libclang-22.so.1"] = "lib/cdh/libclang-22.so.22"
	}
	plan.Extras = []extra{{Path: "lib/cdh/LICENSES/PACKAGES.md", Data: packageMD(host), Mode: 0o644}}
	return plan
}

func packageMD(host string) string {
	arch := strings.TrimPrefix(host, "linux-")
	var body strings.Builder
	fmt.Fprintf(&body, "# LLVM 22 Linux %s软件包来源\n\n", arch)
	if host == "linux-arm64" {
		body.WriteString("本归档仅使用Ubuntu 26.04 `resolute`与`resolute-updates`的官方arm64二进制包。包文件由`http://ports.ubuntu.com/ubuntu-ports`对应APT索引取得，版本、大小与SHA-256如下：\n\n")
	} else {
		body.WriteString("本归档的LLVM 22及其大部分运行库使用Ubuntu 26.04 `resolute`与`resolute-updates`的官方amd64二进制包。`libedit2`与`libgnutls30t64`固定使用Ubuntu 24.04 LTS `noble`版本，使LLVM 22可同时在glibc 2.39和更新宿主上运行；包文件由Ubuntu官方APT索引或其同步镜像取得，版本、大小与SHA-256如下：\n\n")
	}
	body.WriteString("| 包 | 版本 | 字节 | SHA-256 |\n| --- | --- | ---: | --- |\n")
	for _, pkg := range llvmDebs {
		file, ver, _ := pkgSource(pkg, host)
		if file.Size != 0 {
			fmt.Fprintf(&body, "| `%s` | `%s` | %d | `%s` |\n", pkg.Name, ver, file.Size, file.SHA)
		}
	}
	if host == "linux-arm64" {
		body.WriteString("\n归档保留九个LLVM包提供的完整`/usr/lib/llvm-22`目录，并附带全目录动态程序在基础`ubuntu:26.04`镜像中实际缺少的共享库闭包及相对符号链接。每类软件的Ubuntu版权文件同时置于本目录和归档内`lib/cdh/LICENSES/`。\n\n为支持任意安装路径，使用Ubuntu 26.04的patchelf `0.18.0-1.4build1`把动态程序的RPATH设为`$ORIGIN/../lib/cdh`，把附带共享库的RPATH设为其相对库目录。干净`ubuntu:26.04` arm64镜像中的最终检查不设置`LD_LIBRARY_PATH`；其余解析项仅来自镜像已有的C/C++运行库、压缩库、终端库与动态加载器。\n")
	} else {
		body.WriteString("\n归档保留九个LLVM包提供的完整`/usr/lib/llvm-22`目录，并附带全目录动态程序在基础`ubuntu:24.04`与`ubuntu:26.04`环境中需要的共享库及相对符号链接。每类软件的Ubuntu版权文件同时置于本目录和归档内`lib/cdh/LICENSES/`。\n\n为支持任意安装路径，使用Ubuntu 26.04的patchelf `0.18.0-1.4build1`（106098字节，SHA-256 `dd6cde91e0a77a73335a93a4ce41801f21dac36d2158539093c241e46e11b9fc`）把动态程序的RPATH设为`$ORIGIN/../lib/cdh`，把附带共享库的RPATH设为其相对库目录。最终检查不设置`LD_LIBRARY_PATH`；其余解析项仅来自宿主已有的C/C++运行库、压缩库、终端库与动态加载器。\n")
	}
	return body.String()
}

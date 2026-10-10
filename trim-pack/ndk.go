// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import "fmt"

func ndk29(host string) spec {
	plan := spec{Host: host, Comp: "ndk29", Prefix: "android-ndk-r29"}
	platform := "linux-x86_64"
	in := input{Name: "android-ndk-r29-linux.zip", Kind: "zip", Prefix: plan.Prefix,
		SHA256: "4abbbcdc842f3d4879206e9695d52709603e52dd68d3c1fff04b3b5e7a308ecf"}
	if host == "darwin-arm64" {
		platform = "darwin-x86_64"
		in.Name = "android-ndk-r29-darwin.zip"
		in.SHA256 = "ce5e4b100ec5fe5be4eb3edcb2c02528824ff9cda3860f5304619be6c3da34d3"
		plan.OldSHA = "d75ac4144a1abcc23e6152fa1389f00a30347b5eefe1dfcd4504d41c8f869cad"
	} else if host == "linux-arm64" {
		plan.OldSHA = "1b9f8b4b469832bb10b72749690c660f14fcc877950fda504e1402c2c843d77a"
		plan.DataOnly = true
	} else {
		plan.OldSHA = "b8212777164944f1a4c4aa8501fee5d584f31dd7db6e68a2517cdd642d63447a"
	}
	for _, name := range []string{"source.properties", "NOTICE", "NOTICE.toolchain", "CHANGELOG.md", "README.md"} {
		in.Rules = append(in.Rules, rule{From: name, To: name})
	}
	base := "toolchains/llvm/prebuilt/" + platform
	if host == "linux-amd64" {
		in.Rules = append(in.Rules, rule{From: base + "/lib/clang/21/bin", To: base + "/lib/clang/21/bin", Tree: true})
		in.Rules = append(in.Rules, rule{From: base + "/sysroot/NOTICE", To: base + "/sysroot/NOTICE"})
	}
	for _, name := range []string{"AndroidVersion.txt", "BUILD_INFO", "MODULE_LICENSE_APACHE2", "MODULE_LICENSE_BSD_LIKE",
		"MODULE_LICENSE_MIT", "NOTICE", "clang_source_info.md", "manifest_13989888.xml"} {
		if host != "linux-amd64" {
			in.Rules = append(in.Rules, rule{From: base + "/" + name, To: base + "/" + name})
		}
	}
	for _, name := range []string{"sysroot/usr/include", "sysroot/usr/lib/aarch64-linux-android", "lib/clang/21/include", "lib/clang/21/share", "lib/clang/21/lib/linux/aarch64"} {
		in.Rules = append(in.Rules, rule{From: base + "/" + name, To: base + "/" + name, Tree: true})
	}
	for _, arch := range []string{"arm-linux-androideabi", "i686-linux-android", "x86_64-linux-android", "riscv64-linux-android"} {
		in.Drop = append(in.Drop, base+"/sysroot/usr/include/"+arch)
	}
	in.Rules = append(in.Rules, rule{Glob: base + "/lib/clang/21/lib/linux/*aarch64*"})
	if host != "linux-amd64" {
		in.Drop = append(in.Drop, base+"/lib/clang/21/lib/linux/aarch64/lldb-server")
	}
	if host != "linux-arm64" {
		for _, name := range []string{"build", "meta", "sources/android/cpufeatures"} {
			in.Rules = append(in.Rules, rule{From: name, To: name, Tree: true})
		}
		if host == "linux-amd64" {
			in.Drop = append(in.Drop, "build/ndk-build.cmd")
			for _, arch := range []string{"arm-linux-androideabi", "riscv64-linux-android", "x86", "x86_64"} {
				in.Drop = append(in.Drop, "build/core/toolchains/"+arch+"-clang")
			}
		}
		in.Rules = append(in.Rules, rule{From: "ndk-build", To: "ndk-build"},
			rule{From: "prebuilt/" + platform + "/bin/make", To: "prebuilt/" + platform + "/bin/make", Thin: host == "darwin-arm64"})
		for _, name := range []string{"clang", "clang++", "clang-21", "lld", "ld", "ld.lld", "llvm-addr2line", "llvm-ar", "llvm-as",
			"llvm-cov", "llvm-cxxfilt", "llvm-dis", "llvm-dwp", "llvm-link", "llvm-nm", "llvm-objcopy", "llvm-objdump",
			"llvm-profdata", "llvm-ranlib", "llvm-readelf", "llvm-readobj", "llvm-size", "llvm-strings", "llvm-strip", "llvm-symbolizer"} {
			in.Rules = append(in.Rules, rule{From: base + "/bin/" + name, To: base + "/bin/" + name, Thin: host == "darwin-arm64"})
		}
		for api := 21; api <= 35; api++ {
			for _, suffix := range []string{"clang", "clang++"} {
				name := fmt.Sprintf("%s/bin/aarch64-linux-android%d-%s", base, api, suffix)
				in.Rules = append(in.Rules, rule{From: name, To: name})
			}
		}
		plan.Links = map[string]string{}
		for name, target := range map[string]string{"clang": "clang-21", "clang++": "clang", "ld": "ld.lld", "ld.lld": "lld",
			"llvm-addr2line": "llvm-symbolizer", "llvm-readelf": "llvm-readobj", "llvm-strip": "llvm-objcopy", "llvm-ranlib": "llvm-ar"} {
			plan.Links[base+"/bin/"+name] = target
		}
		if host == "darwin-arm64" {
			for _, name := range []string{"libc++.dylib", "libxml2.dylib", "libxml2.2.dylib"} {
				in.Rules = append(in.Rules, rule{From: base + "/lib/" + name, To: base + "/lib/" + name, Thin: true})
			}
			plan.Links[base+"/lib/libxml2.dylib"] = "libxml2.2.dylib"
		} else {
			for _, name := range []string{"libxml2.so", "libxml2.so.2"} {
				in.Rules = append(in.Rules, rule{From: base + "/lib/" + name, To: base + "/lib/" + name})
			}
			plan.Links[base+"/lib/libxml2.so"] = "libxml2.so.2"
		}
	}
	plan.Inputs = []input{in}
	return plan
}

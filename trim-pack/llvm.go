// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

func llvmMac() spec {
	plan := spec{Host: "darwin-arm64", Comp: "llvm22", Prefix: "LLVM-22.1.8-macOS-ARM64",
		OldSHA: "76c2be3c57ca741a095a70ccb739148ea57e13033f6c1d39e9b342aac64c2db2",
		Links:  map[string]string{"bin/clang": "clang-22", "bin/ld.lld": "lld"}}
	plan.Drop = []string{"include/clang-tidy/cert", "include/clang-tidy/misc/ConfusableTable",
		"include/clang-tidy/plugin", "include/clang-tidy/zircon", "include/clang/Basic/Target"}
	in := input{Name: "LLVM-22.1.8-macOS-ARM64.tar.xz", Kind: "tar.xz", Prefix: plan.Prefix,
		SHA256: "f260f4f7c0d430828a81ae8a3826a1d63fc0963ec2459489308cc23b1f7eab4f"}
	for _, name := range llvmTools {
		in.Rules = append(in.Rules, rule{From: "bin/" + name, To: "bin/" + name})
	}
	for _, name := range []string{"include", "share", "lib/clang"} {
		in.Rules = append(in.Rules, rule{From: name, To: name, Tree: true})
	}
	for _, name := range []string{"libc++abi.1.0.dylib", "libunwind.1.0.dylib", "libc++.a", "libc++experimental.a",
		"libc++.1.0.dylib", "libclang-cpp.dylib", "libunwind.a", "libc++abi.a", "libclang.dylib", "libunwind.1.dylib",
		"libunwind.dylib", "libc++.dylib", "libc++.1.dylib", "libc++abi.dylib", "libc++abi.1.dylib", "libc++.modules.json"} {
		in.Rules = append(in.Rules, rule{From: "lib/" + name, To: "lib/" + name})
	}
	plan.Inputs = []input{in, {Name: "LLVM-LICENSE.TXT", Kind: "raw",
		SHA256: "8d85c1057d742e597985c7d4e6320b015a9139385cff4cbae06ffc0ebe89afee",
		Rules:  []rule{{From: "LLVM-LICENSE.TXT", To: "share/licenses/LLVM-LICENSE.TXT"}}}}
	return plan
}

var llvmTools = []string{
	"clang", "clang++", "clang-22", "clang-cpp", "clang-scan-deps", "dsymutil", "ld.lld", "ld64.lld", "lld",
	"lld-link", "wasm-ld", "llvm-addr2line", "llvm-ar", "llvm-as", "llvm-bitcode-strip", "llvm-cov", "llvm-cxxfilt",
	"llvm-dis", "llvm-dwp", "llvm-install-name-tool", "llvm-libtool-darwin", "llvm-link", "llvm-lipo", "llvm-nm",
	"llvm-objcopy", "llvm-objdump", "llvm-otool", "llvm-profdata", "llvm-ranlib", "llvm-readelf", "llvm-readobj",
	"llvm-size", "llvm-strings", "llvm-strip", "llvm-symbolizer",
}

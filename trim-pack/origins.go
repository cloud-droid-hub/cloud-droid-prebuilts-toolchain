// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import "strings"

func origin(in input) string {
	if in.URL != "" {
		return in.URL
	}
	name := in.Name
	for _, prefix := range []string{"build-tools_", "platform-", "cmake-", "android-ndk-", "repository2-"} {
		if strings.HasPrefix(name, prefix) {
			return "https://dl.google.com/android/repository/" + name
		}
	}
	if name == "android-build-tools-37.0.0-linux-arm64-20260618.tar.xz" {
		return "https://github.com/Commit451/android-arm-build-tools/releases/download/platform-tools-37.0.0/" + name
	}
	if name == "LLVM-22.1.8-macOS-ARM64.tar.xz" {
		return "https://github.com/llvm/llvm-project/releases/download/llvmorg-22.1.8/" + name
	}
	if name == "LLVM-LICENSE.TXT" {
		return "https://raw.githubusercontent.com/llvm/llvm-project/llvmorg-22.1.8/LICENSE.TXT"
	}
	if strings.HasPrefix(name, "aosp15-") {
		return aospOrigin(strings.TrimPrefix(name, "aosp15-"))
	}
	return ""
}

func aospOrigin(name string) string {
	base := "https://android.googlesource.com/platform/"
	rev := "/+/refs/tags/android-15.0.0_r36/"
	for _, host := range []string{"darwin-x86", "linux-x86", "linux_musl-arm64"} {
		if strings.HasPrefix(name, host+"-") {
			rel := strings.TrimPrefix(name, host+"-")
			dir := "lib64/"
			if rel == "aidl" || rel == "hidl-gen" {
				dir = "bin/"
			}
			return base + "prebuilts/build-tools" + rev + host + "/" + dir + rel + "?format=TEXT"
		}
	}
	paths := map[string]string{
		"aidl-NOTICE":       "system/tools/aidl" + rev + "NOTICE",
		"libbase-NOTICE":    "system/libbase" + rev + "NOTICE",
		"liblog-NOTICE":     "system/logging" + rev + "liblog/NOTICE",
		"libhidl-NOTICE":    "system/libhidl" + rev + "NOTICE",
		"libcxx-LICENSE":    "external/libcxx" + rev + "LICENSE.TXT",
		"libcxxabi-LICENSE": "external/libcxxabi" + rev + "LICENSE.TXT",
		"libcxxabi-NOTICE":  "external/libcxxabi" + rev + "NOTICE",
		"musl-COPYRIGHT":    "external/musl" + rev + "COPYRIGHT",
		"build-tools-GPL3":  "prebuilts/build-tools" + rev + "LICENSE-GPL-3.0",
	}
	if rel, ok := paths[name]; ok {
		return base + rel + "?format=TEXT"
	}
	return ""
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import "fmt"

func lookup(host, comp string) (spec, error) {
	if host != "darwin-arm64" && host != "linux-arm64" && host != "linux-amd64" {
		return spec{}, fmt.Errorf("未登记的宿主：%s", host)
	}
	if comp == "sdk37" {
		return sdk37(host), nil
	}
	if comp == "llvm22" {
		if host == "darwin-arm64" {
			return llvmMac(), nil
		}
		return llvmLinux(host), nil
	}
	if comp == "sdkapp37" {
		return sdkApp(host), nil
	}
	if comp == "ndk29" {
		return ndk29(host), nil
	}
	if comp == "aosp15" {
		return aosp15(host), nil
	}
	return spec{}, fmt.Errorf("未登记的裁剪组件：%s", comp)
}

func sdk37(host string) spec {
	plan := spec{Host: host, Comp: "sdk37", Prefix: "android-sdk"}
	tools := input{Name: "build-tools_r37_linux.zip", Kind: "zip", Prefix: "android-37.0",
		SHA256: "01af179347cbcd9c208b7f8171f7b21f6dd1d2f85bcd15e88caa51d5d7b86060"}
	if host == "darwin-arm64" {
		tools.Name = "build-tools_r37_macosx.zip"
		tools.SHA256 = "b5b1ac529028a49f11b596b89d9b34252e0f39388ee7dbd16ae3110f1c9c5722"
		plan.OldSHA = "74932274c545806ece4633e68d1b589b011ce9da29b7eab844acb7f318930d1b"
	} else if host == "linux-arm64" {
		plan.OldSHA = "569a33ce4dcd1f95f4a256fdb76eb31d7e2705f5565a6bc4b22f4dc258724011"
	} else {
		plan.OldSHA = "2551142947b39fa76d26698f99dd626586d279ca9184e7cb61c54e61a2ac3fd4"
	}
	for _, name := range []string{"d8", "apksigner", "lib/d8.jar", "lib/apksigner.jar", "source.properties", "runtime.properties", "NOTICE.txt"} {
		tools.Rules = append(tools.Rules, rule{From: name, To: "build-tools/37.0.0/" + name})
	}
	if host != "linux-arm64" {
		for _, name := range []string{"aapt2", "aidl", "zipalign"} {
			tools.Rules = append(tools.Rules, rule{From: name, To: "build-tools/37.0.0/" + name, Exec: true, Thin: host == "darwin-arm64"})
		}
		lib := "lib64/libc++.so"
		if host == "darwin-arm64" {
			lib = "lib64/libc++.dylib"
		}
		tools.Rules = append(tools.Rules, rule{From: lib, To: "build-tools/37.0.0/" + lib, Thin: host == "darwin-arm64"})
	}
	plan.Inputs = append(plan.Inputs, tools)
	if host == "linux-arm64" {
		arm := input{Name: "android-build-tools-37.0.0-linux-arm64-20260618.tar.xz", Kind: "tar.xz",
			SHA256: "29b1fa192c005d42c228b6ad0f0ad29161453be324308c3bfaa5783c7f76fddc"}
		for _, name := range []string{"aapt2", "aidl", "zipalign"} {
			arm.Rules = append(arm.Rules, rule{From: name, To: "build-tools/37.0.0/" + name, Exec: true})
		}
		arm.Rules = append(arm.Rules, rule{From: "MANIFEST.txt", To: "build-tools/37.0.0/ARM64-MANIFEST.txt"})
		plan.Inputs = append(plan.Inputs, arm)
	}
	platform := input{Name: "platform-36_r02.zip", Kind: "zip", Prefix: "android-36",
		SHA256: "37607369a28c5b640b3a7998868d45898ebcb777565a0e85f9acf36f29631d2e"}
	for _, name := range []string{"android.jar", "source.properties", "data/NOTICE.txt"} {
		platform.Rules = append(platform.Rules, rule{From: name, To: "platforms/android-36/" + name})
	}
	platform.Rules = append(platform.Rules, rule{From: "optional/libbinder_ndk_cpp/android", To: "platforms/android-36/optional/libbinder_ndk_cpp/android", Tree: true})
	plan.Inputs = append(plan.Inputs, platform)
	return plan
}

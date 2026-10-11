// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import "strings"

type sdkInput struct {
	Mac    string
	Lin    string
	MSum   string
	LSum   string
	Ver    string
	Prefix string
	Lic    string
}

var buildInputs = []sdkInput{
	{"build-tools_r34-macosx.zip", "build-tools_r34-linux.zip", "76bf668fe037b1a69197e298ddae5633d4d7f0f41af7ed17e537c80c1ed8a6f3", "e858c4b60069d0431051b225d384413b1643e1289b00a4825aed347f25bd510f", "34.0.0", "android-14", "license-280F93AC"},
	{"build-tools_r35_macosx.zip", "build-tools_r35_linux.zip", "530cdbd1ec315e1477624d7ed2f0f2962108d69f36eddba5894cef9ea2cedb48", "bd3a4966912eb8b30ed0d00b0cda6b6543b949d5ffe00bea54c04c81e1561d88", "35.0.0", "android-15", "license-2A9D8C2F"},
	{"build-tools_r35.0.1_macosx.zip", "build-tools_r35.0.1_linux.zip", "c01e4b763da96ae5ef67e8bdf2abc94fb6cb3e73a42209581feb6a7019a51b9c", "5993499f3229a021b89f87088c57242aeefaa62316bf3d69da7de40bfd5350f1", "35.0.1", "android-15", "license-2A9D8C2E"},
	{"build-tools_r36_macosx.zip", "build-tools_r36_linux.zip", "04e7f3a72044de4926fa038fa0e251a37bba1e1c3fb8beab6f8401bfd9eb4bf3", "5d9ac77fb6ff43d9da518a337b4fcf8f9097113df531d99ccefe80ef7ce8250b", "36.0.0", "android-16", "license-2AAB78AE"},
	{"build-tools_r37_macosx.zip", "build-tools_r37_linux.zip", "b5b1ac529028a49f11b596b89d9b34252e0f39388ee7dbd16ae3110f1c9c5722", "01af179347cbcd9c208b7f8171f7b21f6dd1d2f85bcd15e88caa51d5d7b86060", "37.0.0", "android-37.0", "license-2AB95129"},
}

var cmakeInputs = []sdkInput{
	{"cmake-3.22.1-darwin.zip", "cmake-3.22.1-linux.zip", "c25d4072e70bbae1bfb0483a85c9237e4c7ceebfd67fe77bbe87d79571f0394f", "9196644852a978012caf7a4067ba1898debf6cc204c3341562771e31080d6869", "3.22.1", "", ""},
	{"cmake-4.1.2-darwin.zip", "cmake-4.1.2-linux.zip", "130b80221abda282b96e3669cef5dccf0d916e988e08051f5daccdd03360a1ac", "46bda18f0f3c28a55f88da4ba05fec21abe3d9bca0a78a8b18d444987b3dc817", "4.1.2", "", ""},
}

func sdkApp(host string) spec {
	plan := spec{Host: host, Comp: "sdkapp37", Prefix: "android-sdk"}
	if host == "darwin-arm64" {
		plan.OldSHA = "0b1196706226209f8c70fcbaffb878fea4bd88a9a7ea06e4bde0a1fdef0a37d8"
	} else if host == "linux-arm64" {
		plan.OldSHA = "2924de75063ebf0e17f0409f2ddd0913ae5176f8c569aed4735bd31b6397a561"
	} else {
		plan.OldSHA = "7f17aa46e3f96aa86646125c5ad6d1da8d0b92e99983ccb0e1fa345a4bbbc44b"
	}
	plan.Inputs = []input{{Name: "repository2-3.xml", Kind: "raw", License: true,
		SHA256: "13dd16e8984cc6f3202a72c8eb269bd7ad03be4247b41b4412b315c5b8bf6d4b"}}
	for _, src := range buildInputs {
		if host == "linux-arm64" && src.Ver != "37.0.0" {
			continue
		}
		base := "build-tools/" + src.Ver
		in := input{Name: src.Lin, SHA256: src.LSum, Kind: "zip", Prefix: src.Prefix,
			Rules: []rule{{To: base, Tree: true, Thin: host == "darwin-arm64"}}}
		if host == "darwin-arm64" {
			in.Name, in.SHA256 = src.Mac, src.MSum
		}
		for _, name := range []string{"renderscript", "lib64/libLLVM.dylib", "lib64/libbcc.dylib", "lib64/libbcinfo.dylib",
			"lib64/libLLVM.so", "lib64/libbcc.so", "lib64/libbcinfo.so", "lib64/libc++.1.dylib", "lib64/libc++abi.1.dylib",
			"lib64/libc++.so.1", "lld", "lld-bin", "llvm-rs-cc", "bcc_compat", "package.xml",
			"lib64/libLLVM_android.dylib", "lib64/libclang_android.dylib",
			"lib64/libLLVM_android.so", "lib64/libclang_android.so",
			"aarch64-linux-android-ld", "arm-linux-androideabi-ld", "i686-linux-android-ld", "x86_64-linux-android-ld", "mipsel-linux-android-ld"} {
			in.Drop = append(in.Drop, name)
		}
		if host == "linux-arm64" {
			for _, name := range []string{"aapt", "aapt2", "aidl", "dexdump", "split-select", "zipalign", "lib64"} {
				in.Drop = append(in.Drop, name)
			}
			plan.Inputs = append(plan.Inputs, armSDK())
		}
		plan.Inputs = append(plan.Inputs, in)
		display := "Android SDK Build-Tools " + strings.Split(src.Ver, ".")[0]
		if src.Ver == "35.0.1" {
			display = "Android SDK Build-Tools 35.0.1"
		}
		plan.Packages = append(plan.Packages, pkgDef{Path: "build-tools;" + src.Ver, Kind: "generic", Version: src.Ver,
			Display: display, LicID: src.Lic, Compact: host == "darwin-arm64"})
	}
	for _, src := range []struct {
		Name, Sum, Api, Ext, Rev, Lic string
	}{
		{"platform-34-ext7_r03.zip", "16fdb74c55e59ae3ef52def135aec713508467bd56d7dabcd8c9be31fa8b20f3", "34", "7", "3", "license-2802BC3D"},
		{"platform-35_r02.zip", "0988cacad01b38a18a47bac14a0695f246bc76c1b06c0eeb8eb0dc825ab0c8e0", "35", "13", "2", "license-2802B622"},
		{"platform-36_r02.zip", "37607369a28c5b640b3a7998868d45898ebcb777565a0e85f9acf36f29631d2e", "36", "17", "2", "license-2802B623"},
		{"platform-37.0_r02.zip", "840b23e827f96e64aea4c89a1194aac3dc5f6bad37edb231c5c795d890330e8d", "37.0", "22", "2", ""},
	} {
		base := "platforms/android-" + src.Api
		plan.Inputs = append(plan.Inputs, input{Name: src.Name, SHA256: src.Sum, Kind: "zip", Prefix: "android-" + src.Api,
			Rules: []rule{{To: base, Tree: true}}})
		for _, name := range []string{"skins", "templates", "data/res", "android-stubs-src.jar", "package.xml"} {
			plan.Drop = append(plan.Drop, base+"/"+name)
		}
		display := "Android SDK Platform " + src.Api
		if src.Api != "37.0" {
			display += ", rev " + src.Rev
		}
		plan.Packages = append(plan.Packages, pkgDef{Path: "platforms;android-" + src.Api, Kind: "platform", Api: src.Api,
			Ext: src.Ext, Rev: src.Rev, Display: display, LicID: src.Lic, Compact: src.Api != "37.0", Deps: src.Api != "37.0"})
	}
	if host != "linux-arm64" {
		for _, src := range cmakeInputs {
			base := "cmake/" + src.Ver
			in := input{Name: src.Lin, SHA256: src.LSum, Kind: "zip", Rules: []rule{{To: base, Tree: true, Thin: host == "darwin-arm64"}}}
			if host == "darwin-arm64" {
				in.Name, in.SHA256 = src.Mac, src.MSum
			}
			plan.Inputs = append(plan.Inputs, in)
			for _, name := range []string{"doc", "bin/ccmake", "bin/cpack", "bin/ctest", "package.xml"} {
				plan.Drop = append(plan.Drop, base+"/"+name)
			}
			plan.Drop = append(plan.Drop, base+"/share/cmake-"+strings.Join(strings.Split(src.Ver, ".")[:2], ".")+"/Modules/Internal/CPack/CPack.OSXScriptLauncher.in")
			plan.Packages = append(plan.Packages, pkgDef{Path: "cmake;" + src.Ver, Kind: "generic", Version: src.Ver, Display: "CMake " + src.Ver})
		}
	}
	return plan
}

func armSDK() input {
	arm := input{Name: "android-build-tools-37.0.0-linux-arm64-20260618.tar.xz", Kind: "tar.xz",
		SHA256: "29b1fa192c005d42c228b6ad0f0ad29161453be324308c3bfaa5783c7f76fddc"}
	for _, name := range []string{"aapt2", "aidl", "zipalign"} {
		arm.Rules = append(arm.Rules, rule{From: name, To: "build-tools/37.0.0/" + name, Exec: true})
	}
	arm.Rules = append(arm.Rules, rule{From: "MANIFEST.txt", To: "build-tools/37.0.0/ARM64-MANIFEST.txt"})
	return arm
}

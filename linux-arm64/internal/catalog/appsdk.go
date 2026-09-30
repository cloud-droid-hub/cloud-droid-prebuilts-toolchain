// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// appSDK发布主项目源码App构建使用的Android SDK组件在本宿主的可登记部分：Build Tools 37.0.0与Platform
// android-34 r3、android-35 r2、android-36 r2、android-37.0 r2，保持标准SDK根布局。Google不发布Linux arm64
// 原生程序：Build Tools 37.0.0的aapt2、aidl、zipalign与sdk37相同，取已登记的Commit451 android-arm-build-tools
// 37.0.0归档原始字节，其余文件取官方Linux zip；Build Tools 34.0.0至36.0.0与CMake没有登记的arm64来源，不在归档内。
// Build Tools的Linux zip与platforms/android-37.0没有package.xml，按官方仓库XML的条目补齐。
func appSDK() Archive {
	return Archive{
		Name: "Android SDK for source apps", Version: "37.0.0", Dir: "sdkapp37",
		Format: TarGzip, Prefix: "android-sdk",
		SHA256: "2924de75063ebf0e17f0409f2ddd0913ae5176f8c569aed4735bd31b6397a561",
		Parts: []Part{{
			Path: "payload/sdkapp37/android-sdk-apps-37.0.0-linux-arm64.tar.gz.part-000",
			Size: 94371840, SHA256: "d6468af49de580d333750b6a78362a0d4621e8a0bd69bdd67abaf13c643ef999",
		}, {
			Path: "payload/sdkapp37/android-sdk-apps-37.0.0-linux-arm64.tar.gz.part-001",
			Size: 38240037, SHA256: "7f0d2050f085049013b1e8667e8cd87c87e1791f34a4a0b89389a1d0150a411e",
		}},
		Files: []FileCheck{{
			Path: "build-tools/37.0.0/source.properties", Size: 38,
			SHA256: "fbd87c8567afbc6dc78e140097fcde234f4a61fa7065e85081d43e442ccd3d24",
			Want:   []string{"Pkg.Revision=37.0.0"},
		}, {
			Path: "build-tools/37.0.0/package.xml", Size: 17721,
			SHA256: "139c451f0b0309b5af0ddbad8bf81b537e4ec36243472a9bab81ddca17d1b3de",
			Want:   []string{`path="build-tools;37.0.0"`},
		}, {
			Path: "build-tools/37.0.0/aapt2", Size: 8414424,
			SHA256: "e292e87995720841e6198ed89b03186b0f99bcdf382e4cf923dc3f39529f1788",
		}, {
			Path: "build-tools/37.0.0/aidl", Size: 2853296,
			SHA256: "835d94ec1dc5a9854fc6a954e0e3f70599eff54fb5f1f79ea324ca166efbe743",
		}, {
			Path: "build-tools/37.0.0/zipalign", Size: 251376,
			SHA256: "08caed25366460755dd6304136e3c722cdb7228869777eb3fc76138362f4a02d",
		}, {
			Path: "platforms/android-34/source.properties", Size: 256,
			SHA256: "a2e2cf71d36c90fb1a2ab5af9d0cad186e4604147f3fb37416c214881bf58489",
			Want:   []string{"AndroidVersion.ApiLevel=34", "Pkg.Revision=3"},
		}, {
			Path: "platforms/android-35/source.properties", Size: 257,
			SHA256: "2c3764446f335ad2cc44383a0360fe247620b7c774ec100d5087771ac8ed3b28",
			Want:   []string{"AndroidVersion.ApiLevel=35", "Pkg.Revision=2"},
		}, {
			Path: "platforms/android-36/source.properties", Size: 257,
			SHA256: "72291ff6611dbfc46fdfb6de2821c4a62c87befa5200713847eda426e3a501c2",
			Want:   []string{"AndroidVersion.ApiLevel=36", "Pkg.Revision=2"},
		}, {
			Path: "platforms/android-37.0/source.properties", Size: 343,
			SHA256: "fa89029fc0cee414cb5e44a37cd3033771217eeecb5086577bb6d9b1b03b5b5f",
			Want:   []string{"AndroidVersion.ApiLevel=37.0", "Pkg.Revision=2"},
		}, {
			Path: "platforms/android-37.0/package.xml", Size: 17871,
			SHA256: "e256168fd8186a78da10cfdfa4c376de7f80af8f987a0f983bf602bc4e5491a4",
			Want:   []string{`path="platforms;android-37.0"`},
		}, {
			Path: "platforms/android-34/android.jar", Size: 26361808,
			SHA256: "6cea1df3efb77103ac3e2beb9bf4718964b0e0869ab16d39d29d5cbae1c147ad",
		}, {
			Path: "platforms/android-35/android.jar", Size: 27092450,
			SHA256: "4566663c3876e022b4fa4ced8c8697c4ab1688267f090114fd92d027b32e619b",
		}, {
			Path: "platforms/android-36/android.jar", Size: 27768026,
			SHA256: "d9eb9da824d9e247a352f570f01e1169e725b2954bca9e283a71786c59b59f9a",
		}, {
			Path: "platforms/android-37.0/android.jar", Size: 43044472,
			SHA256: "bf1b4387cc7ca94fc6ef684f040d9d16fbf16248e181819f020736ea2053f177",
		}},
		Probes: []Probe{{
			Path: "build-tools/37.0.0/aapt2", Args: []string{"version"},
			Want: []string{"Android Asset Packaging Tool (aapt) 2."}, Machine: ELFARM64,
		}, {
			Path: "build-tools/37.0.0/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler:"}, Machine: ELFARM64,
		}},
	}
}

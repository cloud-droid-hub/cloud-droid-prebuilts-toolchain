// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// appSDK发布主项目源码App构建使用的Android SDK组件：Build Tools 34.0.0、35.0.0、35.0.1、36.0.0、37.0.0，
// Platform android-34 r3、android-35 r2、android-36 r2、android-37.0 r2，CMake 3.22.1与4.1.2，保持标准SDK根布局。
// 内容取自Google仓库XML登记的官方Linux zip，原生程序保持官方x86-64字节，裁掉RenderScript、链接器脚本、
// 界面皮肤与模板、框架资源、源码jar和CMake里只含ppc与i386的CPack模板；Build Tools的Linux zip、
// platforms/android-37.0与两个CMake的官方zip没有package.xml，按官方仓库XML的条目补齐。
func appSDK() Archive {
	return Archive{
		Name: "Android SDK for source apps", Version: "37.0.0", Dir: "sdkapp37",
		Format: TarGzip, Prefix: "android-sdk",
		SHA256: "7f17aa46e3f96aa86646125c5ad6d1da8d0b92e99983ccb0e1fa345a4bbbc44b",
		Parts: []Part{{
			Path: "payload/sdkapp37/android-sdk-apps-37.0.0-linux-amd64.tar.gz.part-000",
			Size: 94371840, SHA256: "d2c97069b64f1c068086f01f472a704c2adc5ab19ef6677b24ebce4eaa6daa65",
		}, {
			Path: "payload/sdkapp37/android-sdk-apps-37.0.0-linux-amd64.tar.gz.part-001",
			Size: 94371840, SHA256: "a88158c9478c64b2a4b0e9ac8ecbede884fd05065a778c6174038df831ff69ad",
		}, {
			Path: "payload/sdkapp37/android-sdk-apps-37.0.0-linux-amd64.tar.gz.part-002",
			Size: 51473541, SHA256: "ea8bc9892a963c6d986174fbd2415016de677f9ce34bd5f32977d91ed8a79bd8",
		}},
		Files: []FileCheck{{
			Path: "build-tools/34.0.0/source.properties", Size: 63,
			SHA256: "c727885baf3695c8fe4fbe566f092e8e114c6cee35ecb9fa95a7ff2ef058de04",
			Want:   []string{"Pkg.Revision=34.0.0"},
		}, {
			Path: "build-tools/35.0.0/source.properties", Size: 63,
			SHA256: "084847d70abc41284feee7ea717e7c92eab0d1be05f048c27445a359cfe109d8",
			Want:   []string{"Pkg.Revision=35.0.0"},
		}, {
			Path: "build-tools/35.0.1/source.properties", Size: 81,
			SHA256: "1ce878b52b027d9b7aba4dfb85d4be3c601e27849fc2b99b1db54b4c25d682a5",
			Want:   []string{"Pkg.Revision=35.0.1"},
		}, {
			Path: "build-tools/36.0.0/source.properties", Size: 62,
			SHA256: "7dee6632e9ad6cb111da2bb99d747211e27927061b1276d040bb1d71fded5ebb",
			Want:   []string{"Pkg.Revision=36.0.0"},
		}, {
			Path: "build-tools/37.0.0/source.properties", Size: 38,
			SHA256: "fbd87c8567afbc6dc78e140097fcde234f4a61fa7065e85081d43e442ccd3d24",
			Want:   []string{"Pkg.Revision=37.0.0"},
		}, {
			Path: "build-tools/34.0.0/package.xml", Size: 17721,
			SHA256: "30593a2569b3a72558cd6bfdb0a46e67542990d9ba0e36f8ba523d2dc63a84d3",
			Want:   []string{`path="build-tools;34.0.0"`},
		}, {
			Path: "build-tools/35.0.0/package.xml", Size: 17721,
			SHA256: "f7ab2d7deab011f8e44f70bb37f8c2cdb6a43eaa00c3f2a59ea1837f88537af1",
			Want:   []string{`path="build-tools;35.0.0"`},
		}, {
			Path: "build-tools/35.0.1/package.xml", Size: 17725,
			SHA256: "1b998ea6be372881d6ed64c4aacd378332357a2c84d903225085f11fe05fe7f2",
			Want:   []string{`path="build-tools;35.0.1"`},
		}, {
			Path: "build-tools/36.0.0/package.xml", Size: 17721,
			SHA256: "d3ab816d09cc2db0a82ef1a4a0479271d2a9c7d76fb4f1731691d00b44539a6f",
			Want:   []string{`path="build-tools;36.0.0"`},
		}, {
			Path: "build-tools/37.0.0/package.xml", Size: 17721,
			SHA256: "139c451f0b0309b5af0ddbad8bf81b537e4ec36243472a9bab81ddca17d1b3de",
			Want:   []string{`path="build-tools;37.0.0"`},
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
			Path: "cmake/3.22.1/source.properties", Size: 70,
			SHA256: "3f61a0382a6c9d14b1d2bbef580f5c058b9281babbc0d36b18e5c36aa9a0b5dc",
			Want:   []string{"Pkg.Revision = 3.22.1"},
		}, {
			Path: "cmake/3.22.1/package.xml", Size: 17701,
			SHA256: "41739eefae94172b4b7a40fe37a9c3f8a6c71a5208affd782c9b3c5fdd6a55cf",
			Want:   []string{`path="cmake;3.22.1"`},
		}, {
			Path: "cmake/4.1.2/source.properties", Size: 67,
			SHA256: "e6308fd6f9d829f3f7f4c80035d516af5bbbe8ecc55d8d1660c244d7ad076f05",
			Want:   []string{"Pkg.Revision = 4.1.2"},
		}, {
			Path: "cmake/4.1.2/package.xml", Size: 17698,
			SHA256: "c148e3497ce785544cb933b2bb275fef973f129b6a60ec68ca0b779e68dc621c",
			Want:   []string{`path="cmake;4.1.2"`},
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
		}, {
			Path: "build-tools/34.0.0/aapt2", Size: 6076216,
			SHA256: "760aa057e89e3940c3e944e6bec61eda864ade42fd9c0a3113735e329657705f",
		}, {
			Path: "build-tools/35.0.0/aapt2", Size: 6299424,
			SHA256: "d1096e11aba9c974644369ee3c50d239acac3f3428ffa928e5b9c14dfb7a57de",
		}, {
			Path: "build-tools/35.0.1/aapt2", Size: 6299424,
			SHA256: "5bca541bb691be2783137821b4304beabdf02daacd159ae56e1a4a3a52980637",
		}, {
			Path: "build-tools/36.0.0/aapt2", Size: 5735384,
			SHA256: "1a6a396b9cd071f7040071fdd108718cb98c3c9f4960044f373b288993d19eb7",
		}, {
			Path: "build-tools/37.0.0/aapt2", Size: 6274200,
			SHA256: "0f1c0e560f99db510a76ebc99ad5c3e68438f62e8e8617018a716ca4a589a699",
		}, {
			Path: "cmake/3.22.1/bin/cmake", Size: 14435544,
			SHA256: "c8c39aee0443330e9b1866e1d85cc2405a4eec5dfbb468c5017c3eaecb4964f5",
		}, {
			Path: "cmake/4.1.2/bin/cmake", Size: 21443392,
			SHA256: "7f9c6c6df5774621b6dbd749402f27d5404bc47ad6eb4fdc855aceb66f1d05ef",
		}},
		Probes: []Probe{{
			Path: "build-tools/37.0.0/aapt2", Args: []string{"version"},
			Want: []string{"Android Asset Packaging Tool (aapt) 2."}, Machine: ELFAMD64,
		}, {
			Path: "build-tools/37.0.0/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler:"}, Machine: ELFAMD64,
		}, {
			Path: "cmake/3.22.1/bin/cmake", Args: []string{"--version"},
			Want: []string{"cmake version 3.22.1"}, Machine: ELFAMD64,
		}, {
			Path: "cmake/4.1.2/bin/cmake", Args: []string{"--version"},
			Want: []string{"cmake version 4.1.2"}, Machine: ELFAMD64,
		}},
	}
}

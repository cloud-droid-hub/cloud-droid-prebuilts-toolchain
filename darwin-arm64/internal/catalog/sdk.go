// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// sdk37发布系统、HAL与源码App共用的Android SDK组件：Build Tools 34.0.0、35.0.0、35.0.1、36.0.0、37.0.0，
// Platform android-34 r3、android-35 r2、android-36 r2、android-37.0 r2，CMake 3.22.1与4.1.2，保持标准SDK根布局。
// 内容取自Google仓库XML登记的官方zip，原生程序只保留官方通用程序的arm64部分，裁掉RenderScript、链接器脚本、
// 仅x86_64的运行库、界面皮肤与模板、框架资源和源码jar；platforms/android-37.0与两个CMake的官方zip
// 没有package.xml，按官方仓库XML的条目补齐。
func sdk37() Archive {
	return Archive{
		Name: "Android SDK", Version: "37.0.0", Dir: "sdk37",
		Format: TarGzip, Prefix: "android-sdk",
		SHA256: "0b1196706226209f8c70fcbaffb878fea4bd88a9a7ea06e4bde0a1fdef0a37d8",
		Parts: []Part{{
			Path: "payload/sdk37/android-sdk-37.0.0-darwin-arm64.tar.gz.part-000",
			Size: 94371840, SHA256: "3e39bfd9d9b2d883b9598c746f138c0cab1f9c0b235b31b32b1ae14d738a77d4",
		}, {
			Path: "payload/sdk37/android-sdk-37.0.0-darwin-arm64.tar.gz.part-001",
			Size: 94371840, SHA256: "b6be42caa13aa108a7bfe56b05246d242148a0527b6604c29c52e0316f852109",
		}, {
			Path: "payload/sdk37/android-sdk-37.0.0-darwin-arm64.tar.gz.part-002",
			Size: 31083827, SHA256: "2add27beaa93bb052caf65856e9b27138f9daa26345d51747b006d71724f90ac",
		}},
		Files: []FileCheck{{
			Path: "build-tools/37.0.0/aidl", Size: 3425904,
			SHA256: "b1ccd59cf4e3ca35826b2ddfb112fa222adc7835bc478c71fabed73106551f93",
		}, {
			Path: "build-tools/37.0.0/zipalign", Size: 276528,
			SHA256: "2f5e81c48421a4a2af6d62883f3b57ec6643a4d2430561c83940a2bd2078ad06",
		}, {
			Path: "build-tools/37.0.0/d8", Size: 2678,
			SHA256: "b773a721be3d4988dea9660815a9e441b76100e5b0f5c72b5893cfadadc76c6f",
		}, {
			Path: "build-tools/37.0.0/apksigner", Size: 2959,
			SHA256: "b47549e373b895ce6ca620d0c7887e674d9615ffa837a86ac601dcfd04adb0f0",
		}, {
			Path: "build-tools/37.0.0/lib/d8.jar", Size: 20528803,
			SHA256: "a26ee56252adcff6752c379ae255a0ad538ebec60189e078ec6855487107637d",
		}, {
			Path: "build-tools/37.0.0/lib/apksigner.jar", Size: 1181416,
			SHA256: "2defad215d7ff52968a409cde528cdaef7918b115e276b8e3378ca7a178e4180",
		}, {
			Path: "build-tools/37.0.0/lib64/libc++.dylib", Size: 1319696,
			SHA256: "c2ec753b9b12b0eae72acd0d72a50586e40235149335ab403805a85e6fe35e59",
		}, {
			Path: "platforms/android-36/optional/libbinder_ndk_cpp/android/binder_interface_utils.h", Size: 13955,
			SHA256: "9898e1d9b7e35072cc411a4ef419d1fa2d498b356e1c45943b23381497b8e8e3",
			Want:   []string{"asBinderReference"},
		}, {
			Path: "platforms/android-36/optional/libbinder_ndk_cpp/android/binder_auto_utils.h", Size: 13362,
			SHA256: "8de4018f17330e3cb68eacd66ef2f10da19ae22b2da0e7f6f6ad4d100607a5f4",
			Want:   []string{"class SpAIBinder"},
		}, {
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
			Path: "build-tools/34.0.0/aapt2", Size: 4932032,
			SHA256: "261b4fa1ed366f0ffb26c73f7e87135f09394ddf83c4c8ab303aee0f63c2b208",
		}, {
			Path: "build-tools/35.0.0/aapt2", Size: 5013376,
			SHA256: "564e452dab309d88eacc1b467613b346b2fc37af00dbe9e315a96f1b243be555",
		}, {
			Path: "build-tools/35.0.1/aapt2", Size: 5013376,
			SHA256: "72c945f4a7e7bc5dd9a8b49b5ba207cf052a1fbe3f2b580b1ceee95c01694055",
		}, {
			Path: "build-tools/36.0.0/aapt2", Size: 5160880,
			SHA256: "7d758f52b737942d8885dd35fab5df5b3488b8ddc5f951a36d261780bd715858",
		}, {
			Path: "build-tools/37.0.0/aapt2", Size: 5440544,
			SHA256: "43d128fcb2d8555bbb27a626d8d29250abd17db7d96acc346b9dfc2aa70d1f0e",
		}, {
			Path: "cmake/3.22.1/bin/cmake", Size: 9017680,
			SHA256: "41b41ce6d6192620c6fe4497fa9816e866d34f712f12b44e7c5dbe9078324cf9",
		}, {
			Path: "cmake/4.1.2/bin/cmake", Size: 11081856,
			SHA256: "a72790d32940184e49e5535ba4bb2d6e59ed0981ff6fdf5096f850dd44effc4f",
		}},
		Probes: []Probe{{
			Path: "build-tools/34.0.0/aapt2", Args: []string{"version"},
			Want: []string{"Android Asset Packaging Tool (aapt) 2.19-10229193"}, Machine: MachARM64,
		}, {
			Path: "build-tools/37.0.0/aapt2", Args: []string{"version"},
			Want: []string{"Android Asset Packaging Tool (aapt) 2.20-15087165"}, Machine: MachARM64,
		}, {
			Path: "build-tools/37.0.0/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler: built for platform SDK version 37"}, Machine: MachARM64,
		}, {
			Path: "cmake/3.22.1/bin/cmake", Args: []string{"--version"},
			Want: []string{"cmake version 3.22.1"}, Machine: MachARM64,
		}, {
			Path: "cmake/4.1.2/bin/cmake", Args: []string{"--version"},
			Want: []string{"cmake version 4.1.2"}, Machine: MachARM64,
		}},
	}
}

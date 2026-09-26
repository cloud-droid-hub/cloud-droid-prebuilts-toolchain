// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// androidSDK发布裁剪后的Android SDK Build Tools 37.0.0与Platform android-36 r2，
// 目录保持标准SDK根布局。Google不发布Linux arm64原生程序，aapt2、aidl、zipalign
// 取已登记的Commit451 android-arm-build-tools归档原始字节，其余文件取官方Linux归档。
// Platform另保留optional/libbinder_ndk_cpp头文件，供设备仓HAL构建按SDK布局取用。
func androidSDK() Archive {
	return Archive{
		Name: "Android SDK", Version: "37.0.0", Dir: "sdk37",
		Format: TarGzip, Prefix: "android-sdk",
		SHA256: "569a33ce4dcd1f95f4a256fdb76eb31d7e2705f5565a6bc4b22f4dc258724011",
		Parts: []Part{{
			Path: "payload/sdk37/android-sdk-37.0.0-android-36-r2-linux-arm64.tar.gz.part-000",
			Size: 47392895, SHA256: "569a33ce4dcd1f95f4a256fdb76eb31d7e2705f5565a6bc4b22f4dc258724011",
		}},
		Files: []FileCheck{{
			Path: "build-tools/37.0.0/source.properties", Size: 38,
			SHA256: "fbd87c8567afbc6dc78e140097fcde234f4a61fa7065e85081d43e442ccd3d24",
			Want:   []string{"Pkg.Revision=37.0.0"},
		}, {
			Path: "platforms/android-36/source.properties", Size: 257,
			SHA256: "72291ff6611dbfc46fdfb6de2821c4a62c87befa5200713847eda426e3a501c2",
			Want:   []string{"AndroidVersion.ApiLevel=36", "Pkg.Revision=2"},
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
			Path: "platforms/android-36/android.jar", Size: 27768026,
			SHA256: "d9eb9da824d9e247a352f570f01e1169e725b2954bca9e283a71786c59b59f9a",
		}, {
			Path: "platforms/android-36/optional/libbinder_ndk_cpp/android/binder_interface_utils.h", Size: 13955,
			SHA256: "9898e1d9b7e35072cc411a4ef419d1fa2d498b356e1c45943b23381497b8e8e3",
			Want:   []string{"asBinderReference"},
		}, {
			Path: "platforms/android-36/optional/libbinder_ndk_cpp/android/binder_auto_utils.h", Size: 13362,
			SHA256: "8de4018f17330e3cb68eacd66ef2f10da19ae22b2da0e7f6f6ad4d100607a5f4",
			Want:   []string{"class SpAIBinder"},
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

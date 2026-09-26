// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// androidSDK发布裁剪后的Android SDK Build Tools 37.0.0与Platform android-36 r2，
// 目录保持标准SDK根布局；原生程序只保留官方通用程序的arm64部分。
func androidSDK() Archive {
	return Archive{
		Name: "Android SDK", Version: "37.0.0", Dir: "android-sdk",
		Format: TarGzip, Prefix: "android-sdk",
		SHA256: "802ae68136a919d0ed77e159c40487ad572363ae89ec96e0b1228e36ec2b8203",
		Parts: []Part{{
			Path: "payload/android-sdk/android-sdk-37.0.0-android-36-r2-darwin-arm64.tar.gz.part-000",
			Size: 46422921, SHA256: "802ae68136a919d0ed77e159c40487ad572363ae89ec96e0b1228e36ec2b8203",
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
			Path: "build-tools/37.0.0/aapt2", Size: 5440544,
			SHA256: "43d128fcb2d8555bbb27a626d8d29250abd17db7d96acc346b9dfc2aa70d1f0e",
		}, {
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
			Path: "platforms/android-36/android.jar", Size: 27768026,
			SHA256: "d9eb9da824d9e247a352f570f01e1169e725b2954bca9e283a71786c59b59f9a",
		}},
		Probes: []Probe{{
			Path: "build-tools/37.0.0/aapt2", Args: []string{"version"},
			Want: []string{"Android Asset Packaging Tool (aapt) 2."}, Machine: MachARM64,
		}, {
			Path: "build-tools/37.0.0/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler:"}, Machine: MachARM64,
		}},
	}
}

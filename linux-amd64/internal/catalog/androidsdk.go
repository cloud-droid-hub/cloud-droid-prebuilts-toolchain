// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// androidSDK发布裁剪后的Android SDK Build Tools 37.0.0与Platform android-36 r2，
// 目录保持标准SDK根布局；原生程序与所需libc++.so保持官方Linux归档原始字节。
func androidSDK() Archive {
	return Archive{
		Name: "Android SDK", Version: "37.0.0", Dir: "android-sdk",
		Format: TarGzip, Prefix: "android-sdk",
		SHA256: "02123b365110d4cd022d67595c0ecfa125a1afb39a82083298f98dce13efe53d",
		Parts: []Part{{
			Path: "payload/android-sdk/android-sdk-37.0.0-android-36-r2-linux-amd64.tar.gz.part-000",
			Size: 46778641, SHA256: "02123b365110d4cd022d67595c0ecfa125a1afb39a82083298f98dce13efe53d",
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
			Path: "build-tools/37.0.0/aapt2", Size: 6274200,
			SHA256: "0f1c0e560f99db510a76ebc99ad5c3e68438f62e8e8617018a716ca4a589a699",
		}, {
			Path: "build-tools/37.0.0/aidl", Size: 3754432,
			SHA256: "a4ad150a7cf6ea297c299cb779da7a58c537cfc07923d1e15ccbc554a30cd4ba",
		}, {
			Path: "build-tools/37.0.0/zipalign", Size: 256480,
			SHA256: "0859ed6e59d3ded7519cd6e6409c0d9ce914af0e09dadc2dce7965ee2e991852",
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
			Path: "build-tools/37.0.0/lib64/libc++.so", Size: 1545448,
			SHA256: "9d4c9036e8eca07cb7cf161128c362e8e5550ede7cd7f99078d559c14a5fdf9a",
		}, {
			Path: "platforms/android-36/android.jar", Size: 27768026,
			SHA256: "d9eb9da824d9e247a352f570f01e1169e725b2954bca9e283a71786c59b59f9a",
		}},
		Probes: []Probe{{
			Path: "build-tools/37.0.0/aapt2", Args: []string{"version"},
			Want: []string{"Android Asset Packaging Tool (aapt) 2."}, Machine: ELFAMD64,
		}, {
			Path: "build-tools/37.0.0/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler:"}, Machine: ELFAMD64,
		}},
	}
}

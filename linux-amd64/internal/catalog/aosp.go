// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

func aosp15() Archive {
	return Archive{
		Name: "AOSP 15 tools", Version: "android-15.0.0_r36", Dir: "aosp15",
		Format: TarGzip, Prefix: "aosp15",
		SHA256: "97af66c08ff2cf6a987f92fbb70343af97d7bef536cfd2d319c47f1bc2652631",
		Parts: []Part{{
			Path: "payload/aosp15/aosp-tools-android-15.0.0_r36-linux-amd64.tar.gz.part-000",
			Size: 3462954, SHA256: "97af66c08ff2cf6a987f92fbb70343af97d7bef536cfd2d319c47f1bc2652631",
		}},
		Files: []FileCheck{{
			Path: "source.properties", Size: 144,
			SHA256: "1b4db4a8eba1b12ed61fb96f5d90bc74be5d7973c0d1dc6cfb772b9a7d590e58",
			Want:   []string{"Pkg.Revision=android-15.0.0_r36"},
		}, {
			Path: "bin/aidl", Size: 3884608,
			SHA256: "2d2b352d891e05b0de11654a64638e9a3f84b093a78e54746308466718837645",
		}, {
			Path: "bin/hidl-gen", Size: 5196360,
			SHA256: "5e60bff0e2f9aec70110a9c37db51dea8478b5aec7c7e5d3a4df262ceedd35ee",
		}, {
			Path: "lib64/libc++.so", Size: 1306352,
			SHA256: "7b91de1cba2ad902aad871b77e58af40ec339aac6d514452f71475e895e1fba6",
		}},
		Probes: []Probe{{
			Path: "bin/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler:"}, Machine: ELFAMD64,
		}},
	}
}

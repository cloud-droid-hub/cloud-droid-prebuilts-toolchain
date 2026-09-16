// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

func aosp15() Archive {
	return Archive{
		Name: "AOSP 15 tools", Version: "android-15.0.0_r36", Dir: "aosp15",
		Format: TarGzip, Prefix: "aosp15",
		SHA256: "1b2110fd2ca5f3c8e72c801ecd2fc0e97fe15f0b9c92361bf8b7ffc347c53969",
		Parts: []Part{{
			Path: "payload/aosp15/aosp-tools-android-15.0.0_r36-darwin-arm64.tar.gz.part-000",
			Size: 2317375, SHA256: "1b2110fd2ca5f3c8e72c801ecd2fc0e97fe15f0b9c92361bf8b7ffc347c53969",
		}},
		Files: []FileCheck{{
			Path: "source.properties", Size: 144,
			SHA256: "1b4db4a8eba1b12ed61fb96f5d90bc74be5d7973c0d1dc6cfb772b9a7d590e58",
			Want:   []string{"Pkg.Revision=android-15.0.0_r36"},
		}, {
			Path: "bin/aidl", Size: 3610376,
			SHA256: "e9b95ee3a5cdbd646acf66d95fed208125f99232be914a6d27d702ebd9c1d591",
		}, {
			Path: "bin/hidl-gen", Size: 3800600,
			SHA256: "099e1113b9908f57559be64f6574f7bbb085a3ecb7d8f5da9312fb92deca566d",
		}, {
			Path: "lib64/libc++.dylib", Size: 1139168,
			SHA256: "59a3b915feff155f5c776580ca19f0b46326099dd561ff1448bdc3fcb123e9d9",
		}},
		Probes: []Probe{{
			Path: "bin/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler:"}, Machine: MachARM64,
		}},
	}
}

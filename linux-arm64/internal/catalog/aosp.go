// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

func aosp15() Archive {
	return Archive{
		Name: "AOSP 15 tools", Version: "android-15.0.0_r36", Dir: "aosp15",
		Format: TarGzip, Prefix: "aosp15",
		SHA256: "bac8654b05df602dc790c30fcb4ea6511984d0247b7442560b77a349aab3cd61",
		Parts: []Part{{
			Path: "payload/aosp15/aosp-tools-android-15.0.0_r36-linux-arm64.tar.gz.part-000",
			Size: 3893445, SHA256: "bac8654b05df602dc790c30fcb4ea6511984d0247b7442560b77a349aab3cd61",
		}},
		Files: []FileCheck{{
			Path: "source.properties", Size: 144,
			SHA256: "1b4db4a8eba1b12ed61fb96f5d90bc74be5d7973c0d1dc6cfb772b9a7d590e58",
			Want:   []string{"Pkg.Revision=android-15.0.0_r36"},
		}, {
			Path: "bin/aidl", Size: 3982600,
			SHA256: "8289fd699cf5bc2fa7bf0a974346697ea3a62ca272591bcc1474223c46c5275e",
		}, {
			Path: "bin/hidl-gen", Size: 5303192,
			SHA256: "889df8d8548fd2cc12ffaa95e1dee8df8f1748820cb7f66fc98ed8e0e3bb2691",
		}, {
			Path: "lib64/libc++.so", Size: 1307352,
			SHA256: "90b834537f65a803f69db6bc5aee83eb05f21c155c0e3095d9ad2f55febc4d44",
		}, {
			Path: "lib64/libc_musl.so", Size: 716544,
			SHA256: "385d99768a63d801d8dc4fb166a2b9117959fe8d84147292c5ded1394cf891da",
		}},
		Probes: []Probe{{
			Path: "bin/aidl", Args: []string{"--help"},
			Want: []string{"AIDL Compiler:"}, Machine: ELFARM64,
		}},
	}
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

func aosp15(host string) spec {
	plan := spec{Host: host, Comp: "aosp15", Prefix: "aosp15",
		Extras: []extra{{Path: "source.properties", Mode: 0o644, Data: "Pkg.Desc=AOSP 15 AIDL and HIDL tools\nPkg.Revision=android-15.0.0_r36\nPkg.Source=https://android.googlesource.com/platform/prebuilts/build-tools\n"}}}
	var files []struct{ Name, Sum, To string }
	switch host {
	case "darwin-arm64":
		plan.OldSHA = "1b2110fd2ca5f3c8e72c801ecd2fc0e97fe15f0b9c92361bf8b7ffc347c53969"
		files = []struct{ Name, Sum, To string }{
			{"aosp15-darwin-x86-aidl", "a9088503f89fdef630895bccef2a5bd03310b35fce8b90073a916d5590e7d980", "bin/aidl"},
			{"aosp15-darwin-x86-hidl-gen", "67f2a93691879ee827901f44e1b4217fd457b79fe7b0cd39feb4d75f2d32a8d9", "bin/hidl-gen"},
			{"aosp15-darwin-x86-libc++.dylib", "581770d70a3632cb9945c7ca825ce7d538ae0c0540ef1ce085c8b624758eef6b", "lib64/libc++.dylib"},
		}
	case "linux-arm64":
		plan.OldSHA = "bac8654b05df602dc790c30fcb4ea6511984d0247b7442560b77a349aab3cd61"
		files = []struct{ Name, Sum, To string }{
			{"aosp15-linux_musl-arm64-aidl", "8289fd699cf5bc2fa7bf0a974346697ea3a62ca272591bcc1474223c46c5275e", "bin/aidl"},
			{"aosp15-linux_musl-arm64-hidl-gen", "889df8d8548fd2cc12ffaa95e1dee8df8f1748820cb7f66fc98ed8e0e3bb2691", "bin/hidl-gen"},
			{"aosp15-linux_musl-arm64-libc++.so", "90b834537f65a803f69db6bc5aee83eb05f21c155c0e3095d9ad2f55febc4d44", "lib64/libc++.so"},
			{"aosp15-linux_musl-arm64-libc_musl.so", "385d99768a63d801d8dc4fb166a2b9117959fe8d84147292c5ded1394cf891da", "lib64/libc_musl.so"},
		}
	case "linux-amd64":
		plan.OldSHA = "97af66c08ff2cf6a987f92fbb70343af97d7bef536cfd2d319c47f1bc2652631"
		files = []struct{ Name, Sum, To string }{
			{"aosp15-linux-x86-aidl", "2d2b352d891e05b0de11654a64638e9a3f84b093a78e54746308466718837645", "bin/aidl"},
			{"aosp15-linux-x86-hidl-gen", "5e60bff0e2f9aec70110a9c37db51dea8478b5aec7c7e5d3a4df262ceedd35ee", "bin/hidl-gen"},
			{"aosp15-linux-x86-libc++.so", "7b91de1cba2ad902aad871b77e58af40ec339aac6d514452f71475e895e1fba6", "lib64/libc++.so"},
		}
	}
	for _, file := range files {
		plan.Inputs = append(plan.Inputs, input{Name: file.Name, SHA256: file.Sum, Kind: "raw",
			Rules: []rule{{From: file.Name, To: file.To, Exec: true, Thin: host == "darwin-arm64"}}})
	}
	for _, file := range []struct{ Name, Sum string }{
		{"aidl-NOTICE", "38751245389e1e23f73e6f5384b5cbe7fa972cc4410c5adc9c04b082a0b9561a"},
		{"libbase-NOTICE", "e0afbb47be661ce2076d21a1d4dc02caa58b7c215ab7ff9d151b6724be341f77"},
		{"liblog-NOTICE", "f09747e3bc87e0ec3bdea983b37e18eeda223e01e678f76418a8a731ed8f590f"},
		{"libhidl-NOTICE", "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30"},
		{"libcxx-LICENSE", "c1938c0e9e90639d1e8f5a7f6bebaab83a3642e0ee44295a4c69934686d0bda8"},
		{"libcxxabi-LICENSE", "8b756f8a0b731b67d18714ee3d11b2e182e5c8dac49ddad542e6237ce513d7c6"},
		{"libcxxabi-NOTICE", "10072aa5e7cde74c6b19f554552a6e78f3c0404a966f83041b2cbcf746622b4f"},
		{"musl-COPYRIGHT", "f9bc4423732350eb0b3f7ed7e91d530298476f8fec0c6c427a1c04ade22655af"},
		{"build-tools-GPL3", "e73d7dbbb975482a81dc3b40f202857499fe77be15f68ae68482e6fb7a4aa712"},
	} {
		name := "aosp15-" + file.Name
		plan.Inputs = append(plan.Inputs, input{Name: name, SHA256: file.Sum, Kind: "raw",
			Rules: []rule{{From: name, To: "licenses/" + file.Name}}})
	}
	return plan
}

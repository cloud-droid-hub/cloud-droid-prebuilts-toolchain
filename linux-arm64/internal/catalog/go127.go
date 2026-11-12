// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// Optional由显式组件安装选择，默认核心工具链不要求GUI编译器。
func Optional() []Archive {
	return []Archive{{
		Name: "Go1.27", Version: "1.27.3", Dir: "go127", Format: TarGzip, Prefix: "go",
		SHA256: "a06e99309dbcbdca1a8ccd2282c3ee146e55930886e26b4f1193269587165f49",
		Parts: []Part{{
			Path: "payload/go127/go1.27.3.linux-arm64.tar.gz.part-000", Size: 40000000,
			SHA256: "9d93e63454add14c226e2860399592946956a1385b6ce85e947f75327bbac80d",
		}, {
			Path: "payload/go127/go1.27.3.linux-arm64.tar.gz.part-001", Size: 27076044,
			SHA256: "c2ccfedb43bc769dd7d6b8a31441075bfd6822c16bb3093475638682191a552d",
		}},
		Probes: []Probe{{Path: "bin/go", Args: []string{"version"},
			Want: []string{"go version go1.27.3 linux/arm64"}, Machine: ELFARM64,
			Env: []string{"GOTOOLCHAIN=local"}}},
	}}
}

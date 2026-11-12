// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// Optional由显式组件安装选择，默认核心工具链不要求GUI编译器。
func Optional() []Archive {
	return []Archive{{
		Name: "Go1.27", Version: "1.27.3", Dir: "go127", Format: TarGzip, Prefix: "go",
		SHA256: "65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40",
		Parts: []Part{{
			Path: "payload/go127/go1.27.3.linux-amd64.tar.gz.part-000", Size: 40000000,
			SHA256: "51574a1c8684812a7f5071c01bbc69319e0fc0f6bf6f40d76212d3bf16507ed1",
		}, {
			Path: "payload/go127/go1.27.3.linux-amd64.tar.gz.part-001", Size: 30588678,
			SHA256: "666365b89d8d6a3124f3285a475741bff34896d502669c631ec0f88dc3213a99",
		}},
		Probes: []Probe{{Path: "bin/go", Args: []string{"version"},
			Want: []string{"go version go1.27.3 linux/amd64"}, Machine: ELFAMD64,
			Env: []string{"GOTOOLCHAIN=local"}}},
	}}
}

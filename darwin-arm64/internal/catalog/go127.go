// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// Optional由显式组件安装选择，默认核心工具链不要求GUI编译器。
func Optional() []Archive {
	return []Archive{{
		Name: "Go1.27", Version: "1.27.3", Dir: "go127", Format: TarGzip, Prefix: "go",
		SHA256: "dd3908cf20959d798c1e06ebe5b72213f0505ce01373b40ce3110975a6cd99c2",
		Parts: []Part{{
			Path: "payload/go127/go1.27.3.darwin-arm64.tar.gz.part-000", Size: 40000000,
			SHA256: "cb67d1f039345fae5212bdfa82e9c23ed63225b8f7975cbd160f8940f40ac22e",
		}, {
			Path: "payload/go127/go1.27.3.darwin-arm64.tar.gz.part-001", Size: 28196737,
			SHA256: "46d17e0f02bf0019cf2a35d5d10a66f18f420cc86846fd35ec01058c350d4366",
		}},
		Probes: []Probe{{Path: "bin/go", Args: []string{"version"},
			Want: []string{"go version go1.27.3 darwin/arm64"}, Machine: MachARM64,
			Env: []string{"GOTOOLCHAIN=local"}}},
	}}
}

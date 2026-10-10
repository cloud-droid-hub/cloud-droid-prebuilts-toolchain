// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// bash53登记feicong/bash-static构建任务35507495813的原始arm64程序。
func bash53() Archive {
	return Archive{
		Name: "Bash", Version: "5.3.15", Dir: "bash53", Format: TarGzip, Prefix: "bash53",
		SHA256: "e8012763702074c0a280dad431d3122aa30f1687fd2afd42415fc18621d8fd31",
		Parts: []Part{{
			Path: "payload/bash53/bash-5.3.15-darwin-arm64.tar.gz.part-000", Size: 550660,
			SHA256: "e8012763702074c0a280dad431d3122aa30f1687fd2afd42415fc18621d8fd31",
		}},
		Files: []FileCheck{{
			Path: "bin/bash", Size: 1133736,
			SHA256: "f57e8ab646029fe352633cbdfe1e2ca478e59b5c18fab6fbe839145caeb1e0db",
		}, {
			Path: "licenses/COPYING", Size: 35147,
			SHA256: "8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903",
		}, {
			Path: "licenses/MUSL-COPYRIGHT", Size: 6196,
			SHA256: "b870108ec5e7790e9f9919064f1b9421d62d5f9b0e6c230c6adf7ea2da62e97b",
		}},
		Probes: []Probe{{
			Path: "bin/bash", Args: []string{"--version"},
			Want: []string{"GNU bash, version 5.3.15(1)-release"}, Env: []string{"BASH_ENV=", "ENV="}, Machine: MachARM64,
		}},
	}
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// bash53登记feicong/bash-static构建任务35507495813的原始AArch64静态程序。
func bash53() Archive {
	return Archive{
		Name: "Bash", Version: "5.3.15", Dir: "bash53", Format: TarGzip, Prefix: "bash53",
		SHA256: "48d1d5d98baa952445d25f94d07e5434e469ed627103f641dacb86821e1d33be",
		Parts: []Part{{
			Path: "payload/bash53/bash-5.3.15-linux-arm64.tar.gz.part-000", Size: 551787,
			SHA256: "48d1d5d98baa952445d25f94d07e5434e469ed627103f641dacb86821e1d33be",
		}},
		Files: []FileCheck{{
			Path: "bin/bash", Size: 535312,
			SHA256: "4859ec799023a04e1f3a4cf75de757e74d205409ee852442dbc96195fea5467e",
		}, {
			Path: "licenses/COPYING", Size: 35147,
			SHA256: "8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903",
		}, {
			Path: "licenses/MUSL-COPYRIGHT", Size: 6196,
			SHA256: "b870108ec5e7790e9f9919064f1b9421d62d5f9b0e6c230c6adf7ea2da62e97b",
		}},
		Probes: []Probe{{
			Path: "bin/bash", Args: []string{"--version"},
			Want: []string{"GNU bash, version 5.3.15(1)-release"}, Env: []string{"BASH_ENV=", "ENV="}, Machine: ELFARM64,
		}},
	}
}

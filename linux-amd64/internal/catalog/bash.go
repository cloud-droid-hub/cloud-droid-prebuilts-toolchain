// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// bash53登记feicong/bash-static构建任务35507468329的原始x86-64静态程序。
func bash53() Archive {
	return Archive{
		Name: "Bash", Version: "5.3.15", Dir: "bash53", Format: TarGzip, Prefix: "bash53",
		SHA256: "1b5915bf236ab5fa1ffdfb4781094b981180ca3e1a53739a6f9845c696929b77",
		Parts: []Part{{
			Path: "payload/bash53/bash-5.3.15-linux-amd64.tar.gz.part-000", Size: 544654,
			SHA256: "1b5915bf236ab5fa1ffdfb4781094b981180ca3e1a53739a6f9845c696929b77",
		}},
		Files: []FileCheck{{
			Path: "bin/bash", Size: 529236,
			SHA256: "2aafebdeea9f43a39ff02d986bea13035f50a113b44cb43a17fe32f438b9fe36",
		}, {
			Path: "licenses/COPYING", Size: 35147,
			SHA256: "8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903",
		}, {
			Path: "licenses/MUSL-COPYRIGHT", Size: 6196,
			SHA256: "b870108ec5e7790e9f9919064f1b9421d62d5f9b0e6c230c6adf7ea2da62e97b",
		}},
		Probes: []Probe{{
			Path: "bin/bash", Args: []string{"--version"},
			Want: []string{"GNU bash, version 5.3.15(1)-release"}, Env: []string{"BASH_ENV=", "ENV="}, Machine: ELFAMD64,
		}},
	}
}

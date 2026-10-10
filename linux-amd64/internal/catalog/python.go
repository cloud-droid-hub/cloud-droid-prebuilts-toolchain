// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// python314由主仓python-trim从20261009完整上游归档生成。
func python314() Archive {
	return Archive{
		Name: "Python", Version: "3.14.8", Dir: "python314", Format: TarGzip, Prefix: "python",
		SHA256: "7605cd4099a4dc2dc51cb605d26a12f98c11fa0764b1e6c8122155bd9babab91",
		Parts: []Part{{
			Path: "payload/python314/cpython-3.14.8-cdh-linux-amd64.tar.gz.part-000", Size: 76519762,
			SHA256: "7605cd4099a4dc2dc51cb605d26a12f98c11fa0764b1e6c8122155bd9babab91",
		}},
		Files: []FileCheck{{
			Path: "cdh-python-trim.txt", Size: 227,
			SHA256: "7c2f28c22ad7791bbe5d6e1e6ee3f0dde8e65c04dc5e6aba4bb98d195736e27c",
			Want:   []string{"profile=python314-build-v1", "version=3.14.8", "host=linux-amd64"},
		}, {
			Path: "licenses/LICENSE.cpython.txt", Size: 36164,
			SHA256: "86e61415828a8b5b06ec8d024e6f086ce155a8b85fd0c419c0ba4dc004e74fdd",
		}},
		Probes: []Probe{{
			Path: "bin/python3.14", Args: []string{"-I", "--version"},
			Want: []string{"Python 3.14.8"}, Machine: ELFAMD64,
		}, {
			Path: "bin/python3.14",
			Args: []string{"-I", "-c", "import sys, ssl, sqlite3, bz2, lzma, zlib, ctypes, venv, ensurepip, pip, pyexpat, xml.etree.ElementTree; assert sys.version_info[:3] == (3, 14, 8); print('Python 3.14.8 runtime OK')"},
			Want: []string{"Python 3.14.8 runtime OK"}, Machine: ELFAMD64,
		}},
	}
}

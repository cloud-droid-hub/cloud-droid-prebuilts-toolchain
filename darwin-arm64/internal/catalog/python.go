// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// python314由主仓python-trim从20261009完整上游归档生成。
func python314() Archive {
	return Archive{
		Name: "Python", Version: "3.14.8", Dir: "python314", Format: TarGzip, Prefix: "python",
		SHA256: "3a417e4a2ca1ce63b2e3dcfa142a1febf13d394e2c29c5d96f9b8a9fa1758ac4",
		Parts: []Part{{
			Path: "payload/python314/cpython-3.14.8-cdh-darwin-arm64.tar.gz.part-000", Size: 27243497,
			SHA256: "3a417e4a2ca1ce63b2e3dcfa142a1febf13d394e2c29c5d96f9b8a9fa1758ac4",
		}},
		Files: []FileCheck{{
			Path: "cdh-python-trim.txt", Size: 228,
			SHA256: "e01576174a099941eb76865682ee231b17aa2bd367c5a969284f338e89ba10df",
			Want:   []string{"profile=python314-build-v1", "version=3.14.8", "host=darwin-arm64"},
		}, {
			Path: "licenses/LICENSE.cpython.txt", Size: 36164,
			SHA256: "86e61415828a8b5b06ec8d024e6f086ce155a8b85fd0c419c0ba4dc004e74fdd",
		}},
		Probes: []Probe{{
			Path: "bin/python3.14", Args: []string{"-I", "--version"},
			Want: []string{"Python 3.14.8"}, Machine: MachARM64,
		}, {
			Path: "bin/python3.14",
			Args: []string{"-I", "-c", "import sys, ssl, sqlite3, bz2, lzma, zlib, ctypes, venv, ensurepip, pip, pyexpat, xml.etree.ElementTree; assert sys.version_info[:3] == (3, 14, 8); print('Python 3.14.8 runtime OK')"},
			Want: []string{"Python 3.14.8 runtime OK"}, Machine: MachARM64,
		}},
	}
}

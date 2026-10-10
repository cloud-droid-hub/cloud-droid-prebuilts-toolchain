// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package catalog

// python314由主仓python-trim从20261009完整上游归档生成。
func python314() Archive {
	return Archive{
		Name: "Python", Version: "3.14.8", Dir: "python314", Format: TarGzip, Prefix: "python",
		SHA256: "ce3f118977c3b8b0299b55ed24182c4f68d262825ce638eed7ffdb57ffa7115b",
		Parts: []Part{{
			Path: "payload/python314/cpython-3.14.8-cdh-linux-arm64.tar.gz.part-000", Size: 58800714,
			SHA256: "ce3f118977c3b8b0299b55ed24182c4f68d262825ce638eed7ffdb57ffa7115b",
		}},
		Files: []FileCheck{{
			Path: "cdh-python-trim.txt", Size: 227,
			SHA256: "1c66680f2ef58123a9296a45e31797ac369d4458082e59aff65089f1d7d4c3be",
			Want:   []string{"profile=python314-build-v1", "version=3.14.8", "host=linux-arm64"},
		}, {
			Path: "licenses/LICENSE.cpython.txt", Size: 36164,
			SHA256: "86e61415828a8b5b06ec8d024e6f086ce155a8b85fd0c419c0ba4dc004e74fdd",
		}},
		Probes: []Probe{{
			Path: "bin/python3.14", Args: []string{"-I", "--version"},
			Want: []string{"Python 3.14.8"}, Machine: ELFARM64,
		}, {
			Path: "bin/python3.14",
			Args: []string{"-I", "-c", "import sys, ssl, sqlite3, bz2, lzma, zlib, ctypes, venv, ensurepip, pip, pyexpat, xml.etree.ElementTree; assert sys.version_info[:3] == (3, 14, 8); print('Python 3.14.8 runtime OK')"},
			Want: []string{"Python 3.14.8 runtime OK"}, Machine: ELFARM64,
		}},
	}
}

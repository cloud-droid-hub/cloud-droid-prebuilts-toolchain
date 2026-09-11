// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package catalog

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
)

// Format identifies the compression wrapped around a tar stream.
type Format uint8

// Machine identifies the executable format required by an identity probe.
type Machine uint8

const (
	TarGzip Format = iota + 1
	TarXZ
	MaxPart int64 = 90 * 1024 * 1024
)

const (
	MachARM64 Machine = iota + 1
	ELFARM64
	ELFAMD64
	Script
)

// Part describes one ordinary Git object that forms an archive.
type Part struct {
	Path   string
	Size   int64
	SHA256 string
}

// Probe describes one command that proves an installed tool identity.
type Probe struct {
	Path    string
	Args    []string
	Want    []string
	Env     []string
	Machine Machine
}

// FileCheck describes an immutable identity file inside an archive.
type FileCheck struct {
	Path   string
	Size   int64
	SHA256 string
	Want   []string
}

// Archive describes one tool payload.
type Archive struct {
	Name     string
	Version  string
	Dir      string
	Format   Format
	SHA256   string
	Prefix   string
	Parts    []Part
	Probes   []Probe
	Files    []FileCheck
	DataOnly bool
}

// Spec is the complete static directory for one host branch.
type Spec struct {
	Host     string
	Ready    bool
	Archives []Archive
}

// Current returns the directory compiled into this host branch.
func Current() Spec {
	return Spec{
		Host:  "linux-arm64",
		Ready: true,
		Archives: []Archive{{
			Name:    "Go",
			Version: "1.26.8",
			Dir:     "go126",
			Format:  TarGzip,
			SHA256:  "211ffced9dcb9633a55eac6364816ec0ddd951389a740e88fa8b3337971bdda0",
			Prefix:  "go",
			Parts: []Part{{
				Path:   "payload/go126/go1.26.8.linux-arm64.tar.gz.part-000",
				Size:   63811405,
				SHA256: "211ffced9dcb9633a55eac6364816ec0ddd951389a740e88fa8b3337971bdda0",
			}},
			Probes: []Probe{{
				Path:    "bin/go",
				Args:    []string{"version"},
				Want:    []string{"go version go1.26.8 linux/arm64"},
				Machine: ELFARM64,
				Env:     []string{"GOTOOLCHAIN=local"},
			}},
		}, {
			Name:    "JDK",
			Version: "21.0.12.1+1",
			Dir:     "jdk21",
			Format:  TarGzip,
			SHA256:  "23e37e026f12f3e706f18938ff611db3032d075b09d0879a25d06718c773e223",
			Prefix:  "jdk-21.0.12.1+1",
			Parts: []Part{{
				Path:   "payload/jdk21/OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.12.1_1.tar.gz.part-000",
				Size:   94371840,
				SHA256: "ae1f1189c0deecc4109ae7366205b04c9c0de3adec8fc0ff97961b00d14838f4",
			}, {
				Path:   "payload/jdk21/OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.12.1_1.tar.gz.part-001",
				Size:   94371840,
				SHA256: "c84f17c6d592f9d2d77066de954aea115436f28765fb70730d614ad0f01ca6a4",
			}, {
				Path:   "payload/jdk21/OpenJDK21U-jdk_aarch64_linux_hotspot_21.0.12.1_1.tar.gz.part-002",
				Size:   16897495,
				SHA256: "d6202caa42ff5a0a6dc5021cdc3219d1c6defb681ca6b226db7a441bba7e8957",
			}},
			Probes: []Probe{{
				Path:    "bin/java",
				Args:    []string{"-version"},
				Want:    []string{"Temurin-21.0.12.1+1"},
				Env:     []string{"JAVA_TOOL_OPTIONS=", "_JAVA_OPTIONS=", "JDK_JAVA_OPTIONS="},
				Machine: ELFARM64,
			}},
		}, {
			Name:    "LLVM 19",
			Version: "19.1.7",
			Dir:     "llvm19",
			Format:  TarXZ,
			SHA256:  "11329b1ab6725583b6edfee2abbb9198c80686de16935e9cda4121e8581fbc02",
			Prefix:  "clang+llvm-19.1.7-aarch64-linux-gnu",
			Parts: []Part{{
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-000",
				Size:   94371840,
				SHA256: "35656233a7e77c2d025d7ae0fafb6a705e372f4e9d29e6156cdce6cdbdc43c70",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-001",
				Size:   94371840,
				SHA256: "1262d43d71dd6cae5a66575573b75b3287cdb36590f41d6cf6487eb833357e49",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-002",
				Size:   94371840,
				SHA256: "f2a3204c1c2d432b75dbbb4cac945436ebcb8944b963c85116a9e991587b3aee",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-003",
				Size:   94371840,
				SHA256: "62f47b4b694949096f14bfbfdcf1e4470fc2dba9c02f05f90998c39a7b3d5c67",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-004",
				Size:   94371840,
				SHA256: "e762a6207c3c62a361d3076d4177e0c1e203e141a78fff25001be3464ef8200a",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-005",
				Size:   94371840,
				SHA256: "77421c65b47250f271254938d7793b0c499034edab0ae56d3bc76b73e7728bb5",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-006",
				Size:   94371840,
				SHA256: "20f3bbed0da5782ec55fb33872a7086eec638966a67bde106ab2d2cb98b94da2",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-007",
				Size:   94371840,
				SHA256: "906fac88ce28201adf69ef6ef1c4d9127b3b69e00ff360f77a790ac8707abc33",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-008",
				Size:   94371840,
				SHA256: "afb2715c219b2a7190e6a1dbf50bb5b0f30981d9ae5d15314fdc073566055f29",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-009",
				Size:   94371840,
				SHA256: "4ec7bcde66fb88a4ef6e1a8d383066a917548383a275d28dcbc48ec4a6530d5f",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-010",
				Size:   94371840,
				SHA256: "4200ee1622e0c0cc933f08f9bf48ff699548fafa467b5201ea467be29b956b6e",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-011",
				Size:   94371840,
				SHA256: "5be8990ab96263149a8c05a683629265a207bceb52aa1a3eff14dd0971042f86",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-012",
				Size:   94371840,
				SHA256: "458b6dd723d01940cefa12fa8b1232cfcdcfe8a918f84125265babd612748d6b",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-013",
				Size:   94371840,
				SHA256: "726f9f8ec013724b99af7543393ca3163f03e219a9102cf3e25b682ddc86307b",
			}, {
				Path:   "payload/llvm19/clang+llvm-19.1.7-aarch64-linux-gnu-cdh.tar.xz.part-014",
				Size:   90859072,
				SHA256: "90076b2ca90501d20000bab929c71b808822d016b99cce3f7e1f5d84c179311b",
			}},
			Probes: []Probe{{
				Path:    "bin/clang",
				Args:    []string{"--version"},
				Want:    []string{"clang version 19.1.7"},
				Machine: ELFARM64,
			}, {
				Path:    "bin/ld.lld",
				Args:    []string{"--version"},
				Want:    []string{"LLD 19.1.7"},
				Machine: ELFARM64,
			}, {
				Path:    "bin/llvm-ar",
				Args:    []string{"--version"},
				Want:    []string{"LLVM version 19.1.7"},
				Machine: ELFARM64,
			}},
			Files: []FileCheck{{
				Path:   "lib/cdh/libxml2.so.2.9.14",
				Size:   2102064,
				SHA256: "84bc103b1c389d9730b20eac4f3b8bba20e52cb28c760261cf8caacdac1fc78d",
			}, {
				Path:   "lib/cdh/libicuuc.so.74.2",
				Size:   2230272,
				SHA256: "9d58137df08c36a03417445cc39576b0a95b538bdf0733f5455d2255deb9b799",
			}, {
				Path:   "lib/cdh/libicudata.so.74.2",
				Size:   30803584,
				SHA256: "96b1154b9ab94cfc908c0fe745edc128ee3f67e444c66e2d7a9cf06664df9255",
			}, {
				Path:   "lib/cdh/LICENSES/DEPENDENCIES.md",
				Size:   1177,
				SHA256: "e65df919c78a4a5c43d0b67320f9c90bb785ca881cd19ae668004a426a430222",
			}, {
				Path:   "lib/cdh/LICENSES/LLVM-LICENSE.TXT",
				Size:   15141,
				SHA256: "8d85c1057d742e597985c7d4e6320b015a9139385cff4cbae06ffc0ebe89afee",
			}, {
				Path:   "lib/cdh/LICENSES/icu-copyright",
				Size:   2214,
				SHA256: "5bbff5f646ce750ac0a347e7f91793279b7422a1119820b5f75d98879c201c68",
			}, {
				Path:   "lib/cdh/LICENSES/libxml2-copyright",
				Size:   3067,
				SHA256: "ee746b96cfa5be73c3ea3e4cfb1285e9b315d4c9267f99b2ee9c5d911d9fe3f4",
			}},
		}, {
			Name:    "LLVM 22",
			Version: "22.1.2",
			Dir:     "llvm22",
			Format:  TarXZ,
			SHA256:  "7badc7963ff970a476a4be9683d919f4e8f5e43c60a240b96e4f55df4481534d",
			Prefix:  "llvm-22.1.2-linux-arm64",
			Parts: []Part{{
				Path:   "payload/llvm22/llvm-22.1.2-linux-arm64.tar.xz.part-000",
				Size:   85458612,
				SHA256: "7badc7963ff970a476a4be9683d919f4e8f5e43c60a240b96e4f55df4481534d",
			}},
			Probes: []Probe{{
				Path:    "bin/clang",
				Args:    []string{"--version"},
				Want:    []string{"Ubuntu clang version 22.1.2", "Target: aarch64"},
				Machine: ELFARM64,
			}, {
				Path:    "bin/ld.lld",
				Args:    []string{"--version"},
				Want:    []string{"Ubuntu LLD 22.1.2"},
				Machine: ELFARM64,
			}, {
				Path:    "bin/llvm-ar",
				Args:    []string{"--version"},
				Want:    []string{"Ubuntu LLVM version 22.1.2"},
				Machine: ELFARM64,
			}},
			Files: []FileCheck{{
				Path:   "bin/clang",
				Size:   229241,
				SHA256: "57623c9c20f1ee5abefdef6cde8060984543cd44f30e5bd983dd8a82e0957e60",
			}, {
				Path:   "bin/lld",
				Size:   6346625,
				SHA256: "c14b701a6e99fcc6dcbaf5f94821042e5e1a2c90a126ee716978a0ed6fc09147",
			}, {
				Path:   "bin/llvm-ar",
				Size:   144777,
				SHA256: "7c02cfd7ed31f5b0b7d329be730eefa3e6e0c5ed72eb8d7008fcb7ec5923990d",
			}, {
				Path:   "lib/cdh/libclang-cpp.so.22.1",
				Size:   62459120,
				SHA256: "20a60fe73b4cb992d08369c731f8821e165274ad26387ae63e5ce839ba486b9e",
			}, {
				Path:   "lib/cdh/LICENSES/PACKAGES.md",
				Size:   4931,
				SHA256: "af216aa392fb2f4a739e983c546ac030cb9ef1f6104896f519effec289ade032",
			}, {
				Path:   "lib/cdh/LICENSES/llvm-copyright",
				Size:   5990,
				SHA256: "b2ed2fd344e8ddbf2e0c83113bd40fa1853438be61a0ff20af5acd187c07cbf6",
			}},
		}},
	}
}

// Valid checks structural requirements without reading payload files.
func Valid(spec Spec) error {
	switch spec.Host {
	case "darwin-arm64", "linux-arm64", "linux-amd64":
	default:
		return fmt.Errorf("unsupported host: %q", spec.Host)
	}
	if !spec.Ready {
		if len(spec.Archives) != 0 {
			return errors.New("unpublished directory has archives")
		}
		return nil
	}
	if len(spec.Archives) == 0 {
		return errors.New("published directory has no archives")
	}
	seen := make(map[string]struct{}, len(spec.Archives))
	partSeen := make(map[string]struct{})
	hostMachine := hostMach(spec.Host)
	for _, archive := range spec.Archives {
		if archive.Name == "" || archive.Version == "" || archive.Dir == "" {
			return errors.New("archive identity is incomplete")
		}
		if strings.Contains(archive.Dir, "/") || archive.Dir == "." || archive.Dir == ".." {
			return fmt.Errorf("invalid archive directory: %q", archive.Dir)
		}
		if _, ok := seen[archive.Dir]; ok {
			return fmt.Errorf("duplicate archive directory: %q", archive.Dir)
		}
		seen[archive.Dir] = struct{}{}
		if archive.Format != TarGzip && archive.Format != TarXZ {
			return fmt.Errorf("unsupported archive format for %s", archive.Name)
		}
		if !validSum(archive.SHA256) || len(archive.Parts) == 0 || !validPrefix(archive.Prefix) {
			return fmt.Errorf("archive data is incomplete for %s", archive.Name)
		}
		for _, part := range archive.Parts {
			if part.Path == "" || path.IsAbs(part.Path) || path.Clean(part.Path) != part.Path ||
				part.Path == "." || strings.HasPrefix(part.Path, "../") || strings.Contains(part.Path, "\\") {
				return fmt.Errorf("invalid part path for %s", archive.Name)
			}
			if part.Size <= 0 || part.Size > MaxPart || !validSum(part.SHA256) {
				return fmt.Errorf("invalid part data for %s", archive.Name)
			}
			if _, ok := partSeen[part.Path]; ok {
				return fmt.Errorf("duplicate part path: %s", part.Path)
			}
			partSeen[part.Path] = struct{}{}
		}
		if len(archive.Probes) == 0 && len(archive.Files) == 0 {
			return fmt.Errorf("archive has no identity checks: %s", archive.Name)
		}
		hasVersion := false
		hasNative := false
		fileSeen := make(map[string]struct{}, len(archive.Files))
		for _, file := range archive.Files {
			if file.Path == "" || path.IsAbs(file.Path) || path.Clean(file.Path) != file.Path ||
				file.Path == "." || strings.HasPrefix(file.Path, "../") || strings.Contains(file.Path, "\\") {
				return fmt.Errorf("invalid identity file path for %s", archive.Name)
			}
			if file.Size <= 0 || file.Size > MaxPart || !validSum(file.SHA256) {
				return fmt.Errorf("invalid identity file data for %s", archive.Name)
			}
			if _, exists := fileSeen[file.Path]; exists {
				return fmt.Errorf("duplicate identity file for %s", archive.Name)
			}
			fileSeen[file.Path] = struct{}{}
			for _, want := range file.Want {
				if want == "" || strings.ContainsRune(want, 0) {
					return fmt.Errorf("identity file has empty expected text for %s", archive.Name)
				}
				if strings.Contains(want, archive.Version) {
					hasVersion = true
				}
			}
		}
		for _, probe := range archive.Probes {
			if probe.Path == "" || path.IsAbs(probe.Path) || path.Clean(probe.Path) != probe.Path ||
				probe.Path == "." || strings.HasPrefix(probe.Path, "../") || strings.Contains(probe.Path, "\\") {
				return fmt.Errorf("invalid probe path for %s", archive.Name)
			}
			if len(probe.Want) == 0 {
				return fmt.Errorf("probe has no expected output for %s", archive.Name)
			}
			envSeen := make(map[string]struct{}, len(probe.Env))
			for _, setting := range probe.Env {
				name, _, ok := strings.Cut(setting, "=")
				if !ok || name == "" || strings.ContainsRune(setting, 0) {
					return fmt.Errorf("invalid probe environment for %s", archive.Name)
				}
				if _, exists := envSeen[name]; exists {
					return fmt.Errorf("duplicate probe environment for %s", archive.Name)
				}
				envSeen[name] = struct{}{}
			}
			switch probe.Machine {
			case hostMachine:
				hasNative = true
			case Script:
			default:
				return fmt.Errorf("probe machine does not match host for %s", archive.Name)
			}
			for _, want := range probe.Want {
				if want == "" {
					return fmt.Errorf("probe has empty expected output for %s", archive.Name)
				}
				if strings.Contains(want, archive.Version) {
					hasVersion = true
				}
			}
		}
		if !hasVersion {
			return fmt.Errorf("probe does not verify version for %s", archive.Name)
		}
		if archive.DataOnly && hasNative {
			return fmt.Errorf("data-only archive has a native probe: %s", archive.Name)
		}
		if !archive.DataOnly && !hasNative {
			return fmt.Errorf("archive has no native probe: %s", archive.Name)
		}
	}
	return nil
}

func validPrefix(prefix string) bool {
	return prefix != "" && prefix != "." && !path.IsAbs(prefix) &&
		path.Clean(prefix) == prefix && !strings.HasPrefix(prefix, "../") &&
		!strings.Contains(prefix, "\\")
}

func hostMach(host string) Machine {
	switch host {
	case "darwin-arm64":
		return MachARM64
	case "linux-arm64":
		return ELFARM64
	case "linux-amd64":
		return ELFAMD64
	default:
		return 0
	}
}

func validSum(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

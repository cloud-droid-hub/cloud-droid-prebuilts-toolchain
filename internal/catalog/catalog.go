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
		Host:  "linux-amd64",
		Ready: true,
		Archives: []Archive{{
			Name:    "Go",
			Version: "1.26.8",
			Dir:     "go126",
			Format:  TarGzip,
			SHA256:  "d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b",
			Prefix:  "go",
			Parts: []Part{{
				Path:   "payload/go126/go1.26.8.linux-amd64.tar.gz.part-000",
				Size:   66897291,
				SHA256: "d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b",
			}},
			Probes: []Probe{{
				Path:    "bin/go",
				Args:    []string{"version"},
				Want:    []string{"go version go1.26.8 linux/amd64"},
				Machine: ELFAMD64,
				Env:     []string{"GOTOOLCHAIN=local"},
			}},
		}, {
			Name:    "JDK",
			Version: "21.0.12.1+1",
			Dir:     "jdk21",
			Format:  TarGzip,
			SHA256:  "ce79869e1307ed8ee1e2baa86a412b1eb5b75d10a01006d788a6f968bcfaee94",
			Prefix:  "jdk-21.0.12.1+1",
			Parts: []Part{{
				Path:   "payload/jdk21/OpenJDK21U-jdk_x64_linux_hotspot_21.0.12.1_1.tar.gz.part-000",
				Size:   94371840,
				SHA256: "d8d656393d2dfa954d1b51c1adc06992188d86844938e1f8c4e0cb3f86651d02",
			}, {
				Path:   "payload/jdk21/OpenJDK21U-jdk_x64_linux_hotspot_21.0.12.1_1.tar.gz.part-001",
				Size:   94371840,
				SHA256: "bca36783436430adaf476951c076ff15c4a46bc64928602cb778326686ae5f26",
			}, {
				Path:   "payload/jdk21/OpenJDK21U-jdk_x64_linux_hotspot_21.0.12.1_1.tar.gz.part-002",
				Size:   18729667,
				SHA256: "d3b1a4a52a710a01dceb06b30248b241884e010e1ed95b7aa68542c49c0039ef",
			}},
			Probes: []Probe{{
				Path:    "bin/java",
				Args:    []string{"-version"},
				Want:    []string{"Temurin-21.0.12.1+1"},
				Env:     []string{"JAVA_TOOL_OPTIONS=", "_JAVA_OPTIONS=", "JDK_JAVA_OPTIONS="},
				Machine: ELFAMD64,
			}},
		}, {
			Name:    "LLVM 19",
			Version: "19.1.7",
			Dir:     "llvm19",
			Format:  TarXZ,
			SHA256:  "5872782d3c1dc3ed9fd0324fa6d008dda36dfa7cd53eb21be1059f739472c58c",
			Prefix:  "LLVM-19.1.7-Linux-X64",
			Parts: []Part{{
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-000",
				Size:   94371840,
				SHA256: "949e2f71086352a6384304da005180c75042bbf4fecaf6eb871a0f2ae2515bef",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-001",
				Size:   94371840,
				SHA256: "ec1c9b8e655d9656687849326904b33d3965a7064b1bacbdca0d9dbd5078245b",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-002",
				Size:   94371840,
				SHA256: "459800d856e70470470acef25660aceefd3101ad2eec7b0e18ce8528f0928358",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-003",
				Size:   94371840,
				SHA256: "bce0f2997bc65dc61399538bb39d3ab1dd506b035b4eed31a0c94cde896c2585",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-004",
				Size:   94371840,
				SHA256: "dbcc308f9cd1e5eb19a46794bf7bffafb9b88b0100d5975e82ea4de5fbb05194",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-005",
				Size:   94371840,
				SHA256: "1041305ea21102bba15447253f96720133cec5cc5971da028bfe314e975ed209",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-006",
				Size:   94371840,
				SHA256: "8635c686bd81d43219ce38b59c06f715ca0df11b5211d47f0096f5abeeb73548",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-007",
				Size:   94371840,
				SHA256: "2c3104589947bbf1f90a0dc10fbf31229ce5220386163b1d95af5b29d8f11f17",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-008",
				Size:   94371840,
				SHA256: "7414420e3db6ea4c4364b4cb652e955116641ec746310fad7999083e939f300b",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-009",
				Size:   94371840,
				SHA256: "675c1b4efed3b77da50080ddb48f69f94adc289b1073bc647cef47769ca5954e",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-010",
				Size:   94371840,
				SHA256: "c92a887611da4090c8e9c45f2646015354fe09f60a535b99a38d18475df1327d",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-011",
				Size:   94371840,
				SHA256: "3015391862af35e1d6b6afff321afee40408a90638c6c71f0077998753b8e899",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-012",
				Size:   94371840,
				SHA256: "eb1a82f8b71dbb525c0206538c5eb65860537c21a57b86ae9176b76ca1c6f7e7",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-013",
				Size:   94371840,
				SHA256: "8701496f5f936e96cd6c8eccf519489f78f612fbb8c8921a0a0fab8eb760eac6",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-014",
				Size:   94371840,
				SHA256: "c6a014b5197f95d1269a09819f2a45d0c5610ec0710b444b5c00a79685870f64",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-015",
				Size:   94371840,
				SHA256: "7c3f548cd7d6761b7344f8e3773c44be247eb94410daadfcdbd85d687b6a421a",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-016",
				Size:   94371840,
				SHA256: "f5b91300f60c0709fa994879c89c7258b60507a216d5989751e7175c6ba2b795",
			}, {
				Path:   "payload/llvm19/LLVM-19.1.7-Linux-X64-cdh.tar.xz.part-017",
				Size:   38552816,
				SHA256: "c5a18ea58944f2a135b01eb6f1ff4f4a21c979c9a2a971d72f91686339f28f17",
			}},
			Probes: []Probe{{
				Path:    "bin/clang",
				Args:    []string{"--version"},
				Want:    []string{"clang version 19.1.7", "Target: x86_64"},
				Machine: ELFAMD64,
			}, {
				Path:    "bin/ld.lld",
				Args:    []string{"--version"},
				Want:    []string{"LLD 19.1.7"},
				Machine: ELFAMD64,
			}, {
				Path:    "bin/llvm-ar",
				Args:    []string{"--version"},
				Want:    []string{"LLVM version 19.1.7"},
				Machine: ELFAMD64,
			}},
			Files: []FileCheck{{
				Path:   "lib/cdh/libxml2.so.2.9.14",
				Size:   1967424,
				SHA256: "8b6f68c242929aebe49eb28f800dc3e8ee7147298c94bdabc834f61eff1ac177",
			}, {
				Path:   "lib/cdh/libicuuc.so.74.2",
				Size:   2140336,
				SHA256: "7560aadde38e5f4237a47a1ddd5891f9b36768a77a60faae30beee003ac01901",
			}, {
				Path:   "lib/cdh/libicudata.so.74.2",
				Size:   30795392,
				SHA256: "ddbb3718b8bd9cbd780e5ab08b4503c30a6c4fa0706ebe5d074ed6b596c1714e",
			}, {
				Path:   "lib/cdh/LICENSES/DEPENDENCIES.md",
				Size:   1414,
				SHA256: "4859c6f0c3b66f11d33c4eec75fa498a9e1186360a7fb42089ca821f5638d501",
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
			SHA256:  "d7086a1d1cc27941f7cd5d9f732220316cb7cd5d056f52e15616abb204d25805",
			Prefix:  "llvm-22.1.2-linux-amd64",
			Parts: []Part{{
				Path:   "payload/llvm22/llvm-22.1.2-linux-amd64.tar.xz.part-000",
				Size:   83731000,
				SHA256: "d7086a1d1cc27941f7cd5d9f732220316cb7cd5d056f52e15616abb204d25805",
			}},
			Probes: []Probe{{
				Path:    "bin/clang",
				Args:    []string{"--version"},
				Want:    []string{"Ubuntu clang version 22.1.2", "Target: x86_64"},
				Machine: ELFAMD64,
			}, {
				Path:    "bin/ld.lld",
				Args:    []string{"--version"},
				Want:    []string{"Ubuntu LLD 22.1.2"},
				Machine: ELFAMD64,
			}, {
				Path:    "bin/llvm-ar",
				Args:    []string{"--version"},
				Want:    []string{"Ubuntu LLVM version 22.1.2"},
				Machine: ELFAMD64,
			}},
			Files: []FileCheck{{
				Path:   "bin/clang",
				Size:   171785,
				SHA256: "8c306ec371cbb1f70fbc375828bbc4e0ff346118521bbecbe712f97634ac9e25",
			}, {
				Path:   "bin/lld",
				Size:   6648025,
				SHA256: "bb018b22a28418e451b850b8965c2b4a276abf74359212ada6a273f645aa454f",
			}, {
				Path:   "bin/llvm-ar",
				Size:   95569,
				SHA256: "9228de1e34aa7ccd408441f71a56e406a5c30c991922807bae7b5bc95e955a4d",
			}, {
				Path:   "lib/cdh/libclang-cpp.so.22.1",
				Size:   62456224,
				SHA256: "62e6bdf5bf5b58a697ca643b547f24e1e988388474ff59580e9c58cd27def336",
			}, {
				Path:   "lib/cdh/LICENSES/PACKAGES.md",
				Size:   5018,
				SHA256: "5e84e0abd2d9ce4727ae44ddff2abbb3ea66945cf8ba787f40f1d518f368f342",
			}, {
				Path:   "lib/cdh/LICENSES/llvm-copyright",
				Size:   5990,
				SHA256: "b2ed2fd344e8ddbf2e0c83113bd40fa1853438be61a0ff20af5acd187c07cbf6",
			}},
		}, {
			Name:    "Android NDK",
			Version: "29.0.14206865",
			Dir:     "ndk29",
			Format:  TarXZ,
			SHA256:  "b8212777164944f1a4c4aa8501fee5d584f31dd7db6e68a2517cdd642d63447a",
			Prefix:  "android-ndk-r29",
			Parts: []Part{{
				Path:   "payload/ndk29/android-ndk-r29-linux-amd64.tar.xz.part-000",
				Size:   94211208,
				SHA256: "b8212777164944f1a4c4aa8501fee5d584f31dd7db6e68a2517cdd642d63447a",
			}},
			Probes: []Probe{{
				Path:    "toolchains/llvm/prebuilt/linux-x86_64/bin/clang",
				Args:    []string{"--version"},
				Want:    []string{"clang version 21.0.0", "Target: x86_64"},
				Machine: ELFAMD64,
			}, {
				Path:    "toolchains/llvm/prebuilt/linux-x86_64/bin/ld.lld",
				Args:    []string{"--version"},
				Want:    []string{"LLD 21.0.0"},
				Machine: ELFAMD64,
			}, {
				Path:    "toolchains/llvm/prebuilt/linux-x86_64/bin/llvm-ar",
				Args:    []string{"--version"},
				Want:    []string{"LLVM version 21.0.0"},
				Machine: ELFAMD64,
			}, {
				Path:    "ndk-build",
				Args:    []string{"--version"},
				Want:    []string{"GNU Make 4.3"},
				Machine: Script,
			}},
			Files: []FileCheck{{
				Path:   "source.properties",
				Size:   107,
				SHA256: "716f3518a923198cfab037abb32dc3f1b1f7e9a9dcdcda6b66be5906215d2658",
				Want:   []string{"Pkg.Revision = 29.0.14206865"},
			}, {
				Path:   "prebuilt/linux-x86_64/bin/make",
				Size:   210728,
				SHA256: "ee92765a5dc7556669fd3e7f9fc58c7ba960523d2131052c9b760f5d1a2fc29d",
			}, {
				Path:   "toolchains/llvm/prebuilt/linux-x86_64/lib/libxml2.so.2",
				Size:   1502296,
				SHA256: "b9858e922ae6ad101745947775c37c39b3d57b46256a5668ffbab780e627cf70",
			}, {
				Path:   "toolchains/llvm/prebuilt/linux-x86_64/sysroot/usr/include/aarch64-linux-android/asm/types.h",
				Size:   204,
				SHA256: "4b8d02254d1d5ee1b7eb6ee962a63a935330799c14eaf07f93698c2cf8e59e2a",
			}, {
				Path:   "toolchains/llvm/prebuilt/linux-x86_64/sysroot/usr/lib/aarch64-linux-android/35/crtbegin_dynamic.o",
				Size:   3896,
				SHA256: "7ace808410683717e8d85edfcb0d3cd1e1175e338cacc8e19c555a80060d5326",
			}, {
				Path:   "toolchains/llvm/prebuilt/linux-x86_64/lib/clang/21/lib/linux/libclang_rt.builtins-aarch64-android.a",
				Size:   515368,
				SHA256: "670aa9076a57337dc8b12f4249464bb6b4e19784ae9462cd431710850d3e5ee2",
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

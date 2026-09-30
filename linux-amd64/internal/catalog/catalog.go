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
			Name:    "LLVM 22",
			Version: "22.1.2",
			Dir:     "llvm22",
			Format:  TarXZ,
			SHA256:  "55f4bf7ce313c0f4af2ad36152086686ad3e076b7bbfbc27bb0381118b031517",
			Prefix:  "llvm-22.1.2-linux-amd64",
			Parts: []Part{{
				Path:   "payload/llvm22/llvm-22.1.2-linux-amd64-cdh.tar.xz.part-000",
				Size:   94371840,
				SHA256: "effab4b4666e595bc07531a8ae7a831497fa39f3ddadc2bf88f8fa81e7eea667",
			}, {
				Path:   "payload/llvm22/llvm-22.1.2-linux-amd64-cdh.tar.xz.part-001",
				Size:   46685220,
				SHA256: "5a3f31c8c8fda3c09c2d809a4054f5efdeda8626e94fe784e0eafcf056307945",
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
				Path:   "lib/libffi.so.8",
				Size:   67897,
				SHA256: "5587385396c4eaa7050198a7cc0d9e666f1dadaec86db75b801fe6031be13a9f",
			}, {
				Path:   "lib/libedit.so.2",
				Size:   228425,
				SHA256: "1c778cf484d2328f22e284203c88f0954157d1c89385be256a213d0af036caec",
			}, {
				Path:   "lib/libz3.so.4",
				Size:   19059105,
				SHA256: "1b33bd6945da51409498d19f1bb0c352abaf3e5f9d512bf00fdde96423ba9a5c",
			}, {
				Path:   "lib/libxml2.so.16",
				Size:   1491753,
				SHA256: "afb0114cdbca3cf15c11d5219dc3b1da3a6b25d2bae95123c71021f665b749a8",
			}, {
				Path:   "lib/cdh/libedit.so.2.0.72",
				Size:   228425,
				SHA256: "1c778cf484d2328f22e284203c88f0954157d1c89385be256a213d0af036caec",
			}, {
				Path:   "lib/cdh/libgnutls.so.30.37.1",
				Size:   2130457,
				SHA256: "419c7a6646e6524b109302c9c0361f2c6396a3560f0ec6aee7c7dfcca8d4249e",
			}, {
				Path:   "lib/cdh/libssl.so.3",
				Size:   1140753,
				SHA256: "fc5596cf5f8c9b760e2cc85956331a862fc1235a8809beec4169dacc349087db",
			}, {
				Path:   "lib/cdh/libcrypto.so.3",
				Size:   6560369,
				SHA256: "bf6f4a415f2dd7484a37da10822f850afed45621478dc23ab7fdbc35cf590347",
			}, {
				Path:   "lib/cdh/LICENSES/PACKAGES.md",
				Size:   5269,
				SHA256: "0b4f8fb97a0f9b013112f7ba2fee9577deea830306a0cb5598b2c3583686a3d5",
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
		}, {
			Name:    "Bazel",
			Version: "8.5.1",
			Dir:     "bazel851",
			Format:  TarXZ,
			SHA256:  "56a08dd08f3dbad54acae51e8526c514b9f60613f86bff44aaf003cdc6036d2c",
			Prefix:  "bazel-8.5.1",
			Parts: []Part{{
				Path:   "payload/bazel851/bazel-8.5.1-linux-amd64.tar.xz.part-000",
				Size:   56753736,
				SHA256: "56a08dd08f3dbad54acae51e8526c514b9f60613f86bff44aaf003cdc6036d2c",
			}},
			Files: []FileCheck{{
				Path:   "LICENSE",
				Size:   11358,
				SHA256: "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30",
				Want:   []string{"Apache License"},
			}},
			Probes: []Probe{{
				Path:    "bin/bazel",
				Args:    []string{"--version"},
				Want:    []string{"bazel 8.5.1"},
				Machine: ELFAMD64,
			}},
		}, {
			Name:    "Rust",
			Version: "1.97.1",
			Dir:     "rust197",
			Format:  TarXZ,
			SHA256:  "5724241cdefc8e87dafba1aa3152090147b580ca513f8414ec7ed9cadd3372a0",
			Prefix:  "rust-1.97.1",
			Parts: []Part{{
				Path:   "payload/rust197/rust-1.97.1-linux-amd64.tar.xz.part-000",
				Size:   94371840,
				SHA256: "1fa91ec91090d51b7c86d653683a9b5613a61f786ae43c796f71674490bf333a",
			}, {
				Path:   "payload/rust197/rust-1.97.1-linux-amd64.tar.xz.part-001",
				Size:   73423448,
				SHA256: "6c3e1855b1279081edcf65db8d0b97f88270c1dd32232e49ad11995392699663",
			}},
			Files: []FileCheck{{
				Path:   "lib/rustlib/aarch64-linux-android/lib/libcore-61c2b492ae6a25f6.rlib",
				Size:   3155262,
				SHA256: "d9e177b04f5afc9874a5ea8359c1912fcbc21723c82486dfee6770062eb7296b",
			}},
			Probes: []Probe{{
				Path:    "bin/rustc",
				Args:    []string{"-vV"},
				Want:    []string{"release: 1.97.1", "LLVM version: 22.1.6"},
				Machine: ELFAMD64,
			}, {
				Path:    "bin/cargo",
				Args:    []string{"-V"},
				Want:    []string{"cargo 1.97.1"},
				Machine: ELFAMD64,
			}, {
				Path:    "bin/rustdoc",
				Args:    []string{"--version"},
				Want:    []string{"rustdoc 1.97.1"},
				Machine: ELFAMD64,
			}},
		}, aosp15(), androidSDK(), appSDK()},
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

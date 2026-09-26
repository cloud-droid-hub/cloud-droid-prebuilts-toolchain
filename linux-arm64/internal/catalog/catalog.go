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
			Name:    "LLVM 22",
			Version: "22.1.2",
			Dir:     "llvm22",
			Format:  TarXZ,
			SHA256:  "18374afd0db2a4a868d51f21f7aa1cc2646de41fec4e94b4e8e955575cc17027",
			Prefix:  "llvm-22.1.2-linux-arm64",
			Parts: []Part{{
				Path:   "payload/llvm22/llvm-22.1.2-linux-arm64-cdh.tar.xz.part-000",
				Size:   94371840,
				SHA256: "76d9a3970a7d8e89afba5ca7249fd807d1c7f2d1782819e600b7ca193ba294f0",
			}, {
				Path:   "payload/llvm22/llvm-22.1.2-linux-arm64-cdh.tar.xz.part-001",
				Size:   60079336,
				SHA256: "d28ee5e6c082622c44e2730c3816ab92aa82b37a9ef730d2c9725f8bbfe273a0",
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
				Path:   "lib/libffi.so.8",
				Size:   199009,
				SHA256: "1bc65683239c49763b470f89f6968286e088e4216b97e1188bce19a6e6d4f783",
			}, {
				Path:   "lib/libedit.so.2",
				Size:   334905,
				SHA256: "4e78810fe1ccaad87e20377644549c9c9c2fc0095bf63b1e437fc625849ef20c",
			}, {
				Path:   "lib/libz3.so.4",
				Size:   17396073,
				SHA256: "8de3409f0691cd121f097d2886f773e4357b5097360ee2c5d4e412bcd39303cf",
			}, {
				Path:   "lib/libxml2.so.16",
				Size:   1614633,
				SHA256: "4a4dea438b1f028117bd3b9c4f06788f1b466088c740d5c04d1973db68f877fe",
			}, {
				Path:   "lib/cdh/LICENSES/PACKAGES.md",
				Size:   4931,
				SHA256: "af216aa392fb2f4a739e983c546ac030cb9ef1f6104896f519effec289ade032",
			}, {
				Path:   "lib/cdh/LICENSES/llvm-copyright",
				Size:   5990,
				SHA256: "b2ed2fd344e8ddbf2e0c83113bd40fa1853438be61a0ff20af5acd187c07cbf6",
			}},
		}, {
			Name:     "Android NDK",
			Version:  "29.0.14206865",
			Dir:      "ndk29",
			Format:   TarXZ,
			SHA256:   "1b9f8b4b469832bb10b72749690c660f14fcc877950fda504e1402c2c843d77a",
			Prefix:   "android-ndk-r29",
			DataOnly: true,
			Parts: []Part{{
				Path:   "payload/ndk29/android-ndk-r29-linux-arm64.tar.xz.part-000",
				Size:   17130064,
				SHA256: "1b9f8b4b469832bb10b72749690c660f14fcc877950fda504e1402c2c843d77a",
			}},
			Files: []FileCheck{{
				Path:   "source.properties",
				Size:   107,
				SHA256: "716f3518a923198cfab037abb32dc3f1b1f7e9a9dcdcda6b66be5906215d2658",
				Want:   []string{"Pkg.Revision = 29.0.14206865"},
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
			SHA256:  "bfc534d25c91a274c36aad38bd0f612c59ca1ea0cc23ec04e7464eb48b3baedb",
			Prefix:  "bazel-8.5.1",
			Parts: []Part{{
				Path:   "payload/bazel851/bazel-8.5.1-linux-arm64.tar.xz.part-000",
				Size:   54908956,
				SHA256: "bfc534d25c91a274c36aad38bd0f612c59ca1ea0cc23ec04e7464eb48b3baedb",
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
				Machine: ELFARM64,
			}},
		}, {
			Name:    "Rust",
			Version: "1.97.1",
			Dir:     "rust197",
			Format:  TarXZ,
			SHA256:  "bab434183ef3e5d5151bafe99e204caee4b28084362ac4e6922d95dc943b23d3",
			Prefix:  "rust-1.97.1",
			Parts: []Part{{
				Path:   "payload/rust197/rust-1.97.1-linux-arm64.tar.xz.part-000",
				Size:   94371840,
				SHA256: "fc8e519b876bb680c1f7655dea8f8645007230da6ef0a5c977dc43a4a1e18e4e",
			}, {
				Path:   "payload/rust197/rust-1.97.1-linux-arm64.tar.xz.part-001",
				Size:   45117112,
				SHA256: "f08ce2311eb195faaef59e89cd7726410ca76b4dfd5ba6d0c6ea965d01ad6b3e",
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
				Machine: ELFARM64,
			}, {
				Path:    "bin/cargo",
				Args:    []string{"-V"},
				Want:    []string{"cargo 1.97.1"},
				Machine: ELFARM64,
			}, {
				Path:    "bin/rustdoc",
				Args:    []string{"--version"},
				Want:    []string{"rustdoc 1.97.1"},
				Machine: ELFARM64,
			}},
		}, aosp15(), androidSDK()},
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

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
		Host:  "darwin-arm64",
		Ready: true,
		Archives: []Archive{{
			Name:    "Go",
			Version: "1.26.8",
			Dir:     "go126",
			Format:  TarGzip,
			SHA256:  "a012b25b571bd0138a03dcd25375ceba866fe5ca822f426d2c66a4de56fd3f4b",
			Prefix:  "go",
			Parts: []Part{{
				Path:   "payload/go126/go1.26.8.darwin-arm64.tar.gz.part-000",
				Size:   64626620,
				SHA256: "a012b25b571bd0138a03dcd25375ceba866fe5ca822f426d2c66a4de56fd3f4b",
			}},
			Probes: []Probe{{
				Path:    "bin/go",
				Args:    []string{"version"},
				Want:    []string{"go version go1.26.8 darwin/arm64"},
				Machine: MachARM64,
				Env:     []string{"GOTOOLCHAIN=local"},
			}},
		}, {
			Name:    "JDK",
			Version: "21.0.12.1+1",
			Dir:     "jdk21",
			Format:  TarGzip,
			SHA256:  "3623232f33a9c3baadf304480b2535f9a3cba8a58d42ecbb438ba267315d9998",
			Prefix:  "jdk-21.0.12.1+1/Contents/Home",
			Parts: []Part{{
				Path:   "payload/jdk21/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz.part-000",
				Size:   94371840,
				SHA256: "66364b6204d07ef043b808d21a00cc254bd0328432bf73a88d243352c7461516",
			}, {
				Path:   "payload/jdk21/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz.part-001",
				Size:   94371840,
				SHA256: "b9eb2b87d5a50d78e332631543cda54df617a120e2c954cf26ebc3dabad0a97b",
			}, {
				Path:   "payload/jdk21/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.12.1_1.tar.gz.part-002",
				Size:   11329724,
				SHA256: "0f7ecd01cdd8e4eb6e82f169242688f064306f953c81d197c4862529a866145f",
			}},
			Probes: []Probe{{
				Path:    "bin/java",
				Args:    []string{"-version"},
				Want:    []string{"Temurin-21.0.12.1+1"},
				Env:     []string{"JAVA_TOOL_OPTIONS=", "_JAVA_OPTIONS=", "JDK_JAVA_OPTIONS="},
				Machine: MachARM64,
			}},
		}, {
			Name:    "LLVM 22",
			Version: "22.1.8",
			Dir:     "llvm22",
			Format:  TarXZ,
			SHA256:  "76c2be3c57ca741a095a70ccb739148ea57e13033f6c1d39e9b342aac64c2db2",
			Prefix:  "LLVM-22.1.8-macOS-ARM64",
			Parts: []Part{{
				Path:   "payload/llvm22/LLVM-22.1.8-macOS-ARM64-cdh.tar.xz.part-000",
				Size:   94371840,
				SHA256: "06237a8525b95ee82661ff565c6cb0489d3e9c9c6681c3439aa9aa0a62938e01",
			}, {
				Path:   "payload/llvm22/LLVM-22.1.8-macOS-ARM64-cdh.tar.xz.part-001",
				Size:   94371840,
				SHA256: "e4987320de882dbd1209f3aa10ec597008db3785609ff30f8989643cbe9cdafe",
			}, {
				Path:   "payload/llvm22/LLVM-22.1.8-macOS-ARM64-cdh.tar.xz.part-002",
				Size:   94371840,
				SHA256: "bb8f3ad02ef648cc7a4eb70966d4ef2667c7b19e6f81be789c4abc9a972bfcba",
			}, {
				Path:   "payload/llvm22/LLVM-22.1.8-macOS-ARM64-cdh.tar.xz.part-003",
				Size:   68591448,
				SHA256: "7cbf6938bb7a7df2227eea074690becb41535f128344cf1bb7776aa3496a87bc",
			}},
			Probes: []Probe{{
				Path:    "bin/clang",
				Args:    []string{"--version"},
				Want:    []string{"clang version 22.1.8"},
				Machine: MachARM64,
			}, {
				Path:    "bin/ld64.lld",
				Args:    []string{"--version"},
				Want:    []string{"LLD 22.1.8"},
				Machine: MachARM64,
			}, {
				Path:    "bin/llvm-ar",
				Args:    []string{"--version"},
				Want:    []string{"LLVM version 22.1.8"},
				Machine: MachARM64,
			}},
		}, {
			Name:    "Android NDK",
			Version: "29.0.14206865",
			Dir:     "ndk29",
			Format:  TarXZ,
			SHA256:  "d75ac4144a1abcc23e6152fa1389f00a30347b5eefe1dfcd4504d41c8f869cad",
			Prefix:  "android-ndk-r29",
			Parts: []Part{{
				Path:   "payload/ndk29/android-ndk-r29-darwin-arm64.tar.xz.part-000",
				Size:   65733068,
				SHA256: "d75ac4144a1abcc23e6152fa1389f00a30347b5eefe1dfcd4504d41c8f869cad",
			}},
			Files: []FileCheck{{
				Path:   "source.properties",
				Size:   107,
				SHA256: "716f3518a923198cfab037abb32dc3f1b1f7e9a9dcdcda6b66be5906215d2658",
				Want:   []string{"Pkg.Revision = 29.0.14206865"},
			}},
			Probes: []Probe{{
				Path:    "toolchains/llvm/prebuilt/darwin-x86_64/bin/clang",
				Args:    []string{"--version"},
				Want:    []string{"clang version 21.0.0"},
				Machine: MachARM64,
			}, {
				Path:    "toolchains/llvm/prebuilt/darwin-x86_64/bin/ld.lld",
				Args:    []string{"--version"},
				Want:    []string{"LLD 21.0.0"},
				Machine: MachARM64,
			}, {
				Path:    "toolchains/llvm/prebuilt/darwin-x86_64/bin/llvm-ar",
				Args:    []string{"--version"},
				Want:    []string{"LLVM version 21.0.0"},
				Machine: MachARM64,
			}, {
				Path:    "ndk-build",
				Args:    []string{"--version"},
				Want:    []string{"GNU Make 4.3"},
				Machine: Script,
			}},
		}, {
			Name:    "Bazel",
			Version: "8.5.1",
			Dir:     "bazel851",
			Format:  TarXZ,
			SHA256:  "d58912c25ec81549ec4d1768ffe1134578dc0878f84190d3743a86aed2d2cf6b",
			Prefix:  "bazel-8.5.1",
			Parts: []Part{{
				Path:   "payload/bazel851/bazel-8.5.1-darwin-arm64.tar.xz.part-000",
				Size:   52001564,
				SHA256: "d58912c25ec81549ec4d1768ffe1134578dc0878f84190d3743a86aed2d2cf6b",
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
				Machine: MachARM64,
			}},
		}, {
			Name:    "Rust",
			Version: "1.97.1",
			Dir:     "rust197",
			Format:  TarXZ,
			SHA256:  "4b6291ed634a36fc6d008a141261a9e2d1a6e10428d0e9c36ce10d961b9fb74d",
			Prefix:  "rust-1.97.1",
			Parts: []Part{{
				Path:   "payload/rust197/rust-1.97.1-darwin-arm64.tar.xz.part-000",
				Size:   94371840,
				SHA256: "c706047175dbde463db3f434c519935de860cb8e599d1dda3c4ae2fb75e9e63b",
			}, {
				Path:   "payload/rust197/rust-1.97.1-darwin-arm64.tar.xz.part-001",
				Size:   45801704,
				SHA256: "c4cdf6be1d624c2befa4ecc85121762c1739fb06d0f1ab3244cf9bc9636a169b",
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
				Machine: MachARM64,
			}, {
				Path:    "bin/cargo",
				Args:    []string{"-V"},
				Want:    []string{"cargo 1.97.1"},
				Machine: MachARM64,
			}, {
				Path:    "bin/rustdoc",
				Args:    []string{"--version"},
				Want:    []string{"rustdoc 1.97.1"},
				Machine: MachARM64,
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

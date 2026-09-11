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
		if len(archive.Probes) == 0 {
			return fmt.Errorf("archive has no identity probes: %s", archive.Name)
		}
		hasVersion := false
		hasNative := false
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

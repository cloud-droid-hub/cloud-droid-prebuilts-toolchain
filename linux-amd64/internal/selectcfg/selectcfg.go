// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

// Package selectcfg resolves installed tool locations without changing process state.
package selectcfg

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
)

// Resolve returns one explicit path from an installed host directory.
func Resolve(root, name string, spec catalog.Spec) (string, error) {
	if root == "" || !spec.Ready {
		return "", errors.New("toolchain directory is unavailable")
	}
	dir := ""
	rel := ""
	switch name {
	case "go":
		dir = "go126"
	case "jdk":
		dir = "jdk21"
	case "llvm19":
		dir = "llvm19"
	case "llvm22":
		dir = "llvm22"
	case "llvm22-clang":
		dir, rel = "llvm22", filepath.Join("bin", "clang")
	case "llvm22-clang++":
		dir, rel = "llvm22", filepath.Join("bin", "clang++")
	case "llvm22-lld":
		dir, rel = "llvm22", filepath.Join("bin", "ld.lld")
	case "llvm22-ar":
		dir, rel = "llvm22", filepath.Join("bin", "llvm-ar")
	case "ndk":
		dir = "ndk29"
	case "ndk-sysroot":
		tag, err := hostTag(spec.Host)
		if err != nil {
			return "", err
		}
		dir, rel = "ndk29", filepath.Join("toolchains", "llvm", "prebuilt", tag, "sysroot")
	case "ndk-resource":
		tag, err := hostTag(spec.Host)
		if err != nil {
			return "", err
		}
		dir, rel = "ndk29", filepath.Join("toolchains", "llvm", "prebuilt", tag, "lib", "clang", "21")
	default:
		return "", fmt.Errorf("unknown tool selection: %s", name)
	}
	actual, ok := findDir(spec, dir)
	if !ok {
		return "", fmt.Errorf("tool is not published: %s", name)
	}
	return filepath.Join(root, actual, rel), nil
}

func findDir(spec catalog.Spec, want string) (string, bool) {
	for _, archive := range spec.Archives {
		if archive.Dir == want {
			return archive.Dir, true
		}
	}
	return "", false
}

func hostTag(host string) (string, error) {
	switch host {
	case "darwin-arm64":
		return "darwin-x86_64", nil
	case "linux-arm64", "linux-amd64":
		return "linux-x86_64", nil
	default:
		return "", fmt.Errorf("unsupported host: %s", host)
	}
}

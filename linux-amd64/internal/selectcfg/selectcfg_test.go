// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package selectcfg

import (
	"path/filepath"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
)

func TestResolve(t *testing.T) {
	spec := catalog.Spec{
		Host:  "linux-amd64",
		Ready: true,
		Archives: []catalog.Archive{
			{Name: "Go", Dir: "go126"},
			{Name: "JDK", Dir: "jdk21trim1"},
			{Name: "LLVM 22", Dir: "llvm22"},
			{Name: "Android NDK", Dir: "ndk29"},
		},
	}
	root := filepath.Join("work", "tools")
	for _, item := range []struct {
		name string
		want string
	}{
		{name: "go", want: filepath.Join(root, "go126")},
		{name: "jdk", want: filepath.Join(root, "jdk21trim1")},
		{name: "llvm22", want: filepath.Join(root, "llvm22")},
		{name: "llvm22-clang", want: filepath.Join(root, "llvm22", "bin", "clang")},
		{name: "llvm22-clang++", want: filepath.Join(root, "llvm22", "bin", "clang++")},
		{name: "llvm22-lld", want: filepath.Join(root, "llvm22", "bin", "ld.lld")},
		{name: "llvm22-ar", want: filepath.Join(root, "llvm22", "bin", "llvm-ar")},
		{name: "ndk", want: filepath.Join(root, "ndk29")},
		{name: "ndk-sysroot", want: filepath.Join(root, "ndk29", "toolchains", "llvm", "prebuilt", "linux-x86_64", "sysroot")},
		{name: "ndk-resource", want: filepath.Join(root, "ndk29", "toolchains", "llvm", "prebuilt", "linux-x86_64", "lib", "clang", "21")},
	} {
		got, err := Resolve(root, item.name, spec)
		if err != nil || got != item.want {
			t.Fatalf("%s=%q err=%v want=%q", item.name, got, err, item.want)
		}
	}
}

func TestBadName(t *testing.T) {
	spec := catalog.Spec{Host: "linux-arm64", Ready: true}
	if _, err := Resolve("tools", "missing", spec); err == nil {
		t.Fatal("unknown selection accepted")
	}
}

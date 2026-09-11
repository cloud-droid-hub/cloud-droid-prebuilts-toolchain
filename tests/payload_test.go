// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

func TestPayloads(t *testing.T) {
	t.Setenv("GOTOOLCHAIN", "go1.27.0")
	t.Setenv("JAVA_TOOL_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("_JAVA_OPTIONS", "-XX:InvalidCDHOption")
	t.Setenv("JDK_JAVA_OPTIONS", "-XX:InvalidCDHOption")
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	dest := filepath.Join(t.TempDir(), "tools")
	if err := install.Run(context.Background(), repo, dest, catalog.Current()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOTOOLCHAIN", "local")
	program := filepath.Join(dest, "go126", "bin", "go")
	output, err := exec.Command(program, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("go version: %v: %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "go version go1.26.8 darwin/arm64" {
		t.Fatalf("go version=%q", output)
	}
	info, err := os.Stat(program)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("go mode=%#o", info.Mode().Perm())
	}
	t.Setenv("JAVA_TOOL_OPTIONS", "")
	t.Setenv("_JAVA_OPTIONS", "")
	t.Setenv("JDK_JAVA_OPTIONS", "")
	java := filepath.Join(dest, "jdk21", "bin", "java")
	output, err = exec.Command(java, "-version").CombinedOutput()
	if err != nil {
		t.Fatalf("java version: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "Temurin-21.0.12.1+1") {
		t.Fatalf("java version=%q", output)
	}
	info, err = os.Stat(java)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("java mode=%#o", info.Mode().Perm())
	}
	for _, name := range []string{"CodeResources", "libjli.dylib", "Info.plist"} {
		if _, err := os.Lstat(filepath.Join(dest, "jdk21", name)); !os.IsNotExist(err) {
			t.Fatalf("JDK根含包装文件%s: %v", name, err)
		}
	}
}

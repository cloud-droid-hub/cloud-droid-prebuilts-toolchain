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

func TestCurrentGo(t *testing.T) {
	t.Setenv("GOTOOLCHAIN", "go1.27.0")
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
}

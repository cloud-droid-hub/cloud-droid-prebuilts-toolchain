// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

type testInfo struct {
	mode  os.FileMode
	links uint16
}

func (i testInfo) Name() string       { return "sample" }
func (i testInfo) Size() int64        { return 1 }
func (i testInfo) Mode() os.FileMode  { return i.mode }
func (i testInfo) ModTime() time.Time { return time.Time{} }
func (i testInfo) IsDir() bool        { return i.mode.IsDir() }
func (i testInfo) Sys() any {
	stat := &syscall.Stat_t{}
	for n := uint16(0); n < i.links; n++ {
		stat.Nlink++
	}
	return stat
}

func TestFileKinds(t *testing.T) {
	if err := plain(testInfo{mode: 0o644, links: 1}); err != nil {
		t.Fatal(err)
	}
	for _, info := range []testInfo{{0o644, 2}, {os.ModeSymlink, 1}, {os.ModeDir, 1}} {
		if err := plain(info); err == nil {
			t.Fatal("invalid injected metadata accepted")
		}
	}
	ids := make(idSet)
	if err := ids.add("first", testInfo{mode: 0o644, links: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ids.add("second", testInfo{mode: 0o644, links: 1}); err == nil {
		t.Fatal("duplicate injected file identity accepted")
	}
}

func TestSpace(t *testing.T) {
	if err := room(minFree+10, 10); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]uint64{{minFree - 1, 0}, {minFree, 1}, {minFree + 10, 11}, {minFree, ^uint64(0)}} {
		if err := room(pair[0], pair[1]); err == nil {
			t.Fatalf("space reservation accepted: %v", pair)
		}
	}
}

func TestNewFiles(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "source.c")
	if err := writeNew(name, "original"); err != nil {
		t.Fatal(err)
	}
	if err := writeNew(name, "changed"); err == nil {
		t.Fatal("existing output was overwritten")
	}
	f, info, err := openPlain(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.WriteFile(name, []byte("different size"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stableFile(f, info); err == nil {
		t.Fatal("input changed without rejection")
	}
}

func TestHost(t *testing.T) {
	for _, host := range []string{"darwin-arm64", "linux-arm64", "linux-amd64"} {
		cfg, err := hostConfig(host)
		if err != nil || native(cfg, cfg.goos, cfg.arch) != nil {
			t.Fatalf("supported host failed: %s: %v", host, err)
		}
		if err := native(cfg, cfg.goos, "other"); err == nil {
			t.Fatal("foreign compiler architecture accepted")
		}
	}
	if _, err := hostConfig("linux-riscv64"); err == nil {
		t.Fatal("unknown host accepted")
	}
	cfg, err := hostConfig(runtime.GOOS + "-" + runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := binaryArch(program, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestKeepDirs(t *testing.T) {
	dir := t.TempDir()
	mark := filepath.Join(dir, "keep")
	if err := os.WriteFile(mark, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := options{host: runtime.GOOS + "-" + runtime.GOARCH, parts: filepath.Join(filepath.Dir(dir), "missing"), prefix: "llvm", dest: dir, work: filepath.Join(filepath.Dir(dir), "unused")}
	if err := check(context.Background(), opts, io.Discard); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing installation directory accepted: %v", err)
	}
	data, err := os.ReadFile(mark)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing directory changed: %q: %v", data, err)
	}
}

func TestProbeEnv(t *testing.T) {
	t.Setenv("LIBCLANG_PATH", "outside")
	t.Setenv("CPATH", "outside")
	t.Setenv("LD_PRELOAD", "outside")
	env := probeEnv("/project/build/probe", "/project/output/llvm22/lib/cdh")
	got := make(map[string]string)
	for _, item := range env {
		key, value, _ := strings.Cut(item, "=")
		if _, exists := got[key]; exists {
			t.Fatalf("duplicate environment name: %s", key)
		}
		got[key] = value
	}
	if got["LIBCLANG_PATH"] != "/project/output/llvm22/lib/cdh" || got["TMPDIR"] != "/project/build/probe" || got["CPATH"] != "" || got["LD_PRELOAD"] != "" {
		t.Fatalf("probe environment was not isolated: %v", got)
	}
}

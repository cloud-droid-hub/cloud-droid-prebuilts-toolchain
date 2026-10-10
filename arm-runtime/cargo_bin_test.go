package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// twinInfo injects uplift link metadata at the stat boundary; no real link is made.
type twinInfo struct {
	os.FileInfo
}

func (info twinInfo) Sys() any { return &syscall.Stat_t{Nlink: 2, Dev: 7, Ino: 99} }

func TestCargoBin(t *testing.T) {
	root := t.TempDir()
	release := filepath.Join(root, "target", "release")
	alias := filepath.Join(release, "vhost-device-media")
	twin := filepath.Join(release, "deps", "vhost_device_media-abc")
	depInfo := filepath.Join(release, "deps", "vhost_device_media-abc.d")
	for name, data := range map[string]string{alias: "product", twin: "product", depInfo: "dep-info"} {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(data), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Each path reports two links while its uplift alias still exists.
	linked := map[string]string{alias: alias, twin: alias}
	lstatFile = func(name string) (os.FileInfo, error) {
		info, err := os.Lstat(name)
		if err != nil || linked[name] == "" {
			return info, err
		}
		if _, err := os.Lstat(linked[name]); err != nil {
			return info, nil
		}
		return twinInfo{info}, nil
	}
	defer func() { lstatFile = os.Lstat }()

	dest := filepath.Join(root, "out", "vhost-device-media")
	if err := cargoBin(alias, dest, root); err != nil {
		t.Fatalf("multi-link cargo product was not resolved: %v", err)
	}
	if _, err := os.Lstat(alias); !os.IsNotExist(err) {
		t.Fatalf("uplift alias must be removed: %v", err)
	}
	if got, err := os.ReadFile(twin); err != nil || string(got) != "product" {
		t.Fatalf("deps product changed: %q %v", got, err)
	}
	if got, err := os.ReadFile(depInfo); err != nil || string(got) != "dep-info" {
		t.Fatalf("unrelated deps file changed: %q %v", got, err)
	}
	info, err := os.Lstat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := regular(info); err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("published product: mode=%v error=%v", info.Mode(), err)
	}
	if got, err := os.ReadFile(dest); err != nil || string(got) != "product" {
		t.Fatalf("published product content: %q %v", got, err)
	}

	single := filepath.Join(release, "single")
	if err := os.WriteFile(single, []byte("single"), 0o755); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(root, "out", "single")
	if err := cargoBin(single, plain, root); err != nil {
		t.Fatalf("single-link cargo product was rejected: %v", err)
	}
	if _, err := os.Lstat(single); err != nil {
		t.Fatalf("single-link product must be kept: %v", err)
	}
	if got, err := os.ReadFile(plain); err != nil || string(got) != "single" {
		t.Fatalf("published single product: %q %v", got, err)
	}

	lone := filepath.Join(root, "lone", "release", "vhost-device-media")
	if err := os.MkdirAll(filepath.Dir(lone), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lone, []byte("lone"), 0o755); err != nil {
		t.Fatal(err)
	}
	linked[lone] = lone
	missing := filepath.Join(root, "out", "lone")
	if err := cargoBin(lone, missing, root); err == nil {
		t.Fatal("multi-link product without a deps twin was accepted")
	}
	if _, err := os.Lstat(lone); err != nil {
		t.Fatalf("rejected product must be kept: %v", err)
	}
	if _, err := os.Lstat(missing); !os.IsNotExist(err) {
		t.Fatalf("rejected product must not be published: %v", err)
	}
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
)

func TestMain(m *testing.M) { os.Exit(workpath.Run(m)) }

func testDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Logf("警告：设置测试目录权限：%v", err)
	}
	return dir
}

func makeSource(t *testing.T, dir string) input {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, item := range []struct{ name, body string }{
		{"up/bin/tool", "original program"}, {"up/LICENSE", "original license"},
		{"up/doc/unused", "unused manual"},
	} {
		wr, err := zw.Create(item.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(wr, item.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source.zip"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return input{Name: "source.zip", SHA256: fmt.Sprintf("%x", sha256.Sum256(buf.Bytes())), Kind: "zip", Prefix: "up",
		Rules: []rule{{From: "bin", To: "bin", Tree: true, Exec: true}, {From: "LICENSE", To: "LICENSE"}}}
}

func readFiles(t *testing.T, dest string) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join(dest, "payload.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	files := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			files[h.Name] = string(data)
		}
	}
	return files
}

func TestReplay(t *testing.T) {
	dir := testDir(t)
	in := makeSource(t, dir)
	plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
	for _, name := range []string{"one", "two"} {
		if err := generate(context.Background(), plan, dir, filepath.Join(dir, name), io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]string{"sample/bin/tool": "original program", "sample/LICENSE": "original license"}
	if got := readFiles(t, filepath.Join(dir, "one")); !reflect.DeepEqual(got, want) {
		t.Fatalf("生成文件=%v want=%v", got, want)
	}
	for _, name := range []string{"payload.tar.gz", "payload.tar.gz.part-000", "report.json"} {
		one, err := os.ReadFile(filepath.Join(dir, "one", name))
		if err != nil {
			t.Fatal(err)
		}
		two, err := os.ReadFile(filepath.Join(dir, "two", name))
		if err != nil || !bytes.Equal(one, two) {
			t.Fatalf("重放%s不同：%v", name, err)
		}
	}
}

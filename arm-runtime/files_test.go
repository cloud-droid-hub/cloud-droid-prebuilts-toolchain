package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeFree 注入固定可用空间并截获警告输出，测试结束后恢复。
func fakeFree(t *testing.T, free uint64) *bytes.Buffer {
	t.Helper()
	oldFree, oldOut := freeOf, warnOut
	var buf bytes.Buffer
	freeOf = func(string) (uint64, error) { return free, nil }
	warnOut = &buf
	t.Cleanup(func() { freeOf, warnOut = oldFree, oldOut })
	return &buf
}

func TestSpaceWarn(t *testing.T) {
	out := fakeFree(t, 37196750848)
	if err := spaceNeed(t.TempDir(), 0); err != nil {
		t.Fatalf("可用空间不足被当作错误返回: %v", err)
	}
	want := "警告：可用空间 37196750848 字节低于50GB下限，继续复制\n"
	if out.String() != want {
		t.Fatalf("警告输出 %q，期望 %q", out.String(), want)
	}
}

func TestSpaceWrite(t *testing.T) {
	out := fakeFree(t, 60<<30)
	if err := spaceNeed(t.TempDir(), 20<<30); err != nil {
		t.Fatalf("写入后不足50GB被当作错误返回: %v", err)
	}
	want := "警告：可用空间 64424509440 字节写入 21474836480 字节后低于50GB下限，继续复制\n"
	if out.String() != want {
		t.Fatalf("警告输出 %q，期望 %q", out.String(), want)
	}
}

func TestSpaceEnough(t *testing.T) {
	out := fakeFree(t, 80<<30)
	if err := spaceNeed(t.TempDir(), 1<<30); err != nil {
		t.Fatalf("空间充足仍返回错误: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("空间充足不应输出警告: %q", out.String())
	}
}

func TestSpaceStat(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := spaceNeed(missing, 0); err == nil {
		t.Fatal("目录不存在时未返回读取失败")
	}
}

func TestSourceLow(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	zip := gzip.NewWriter(&buf)
	data := sampleTar(t, []sample{{"source/COPYING", tar.TypeReg, "", "license", 0o644}})
	if _, err := zip.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zip.Close(); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "source.tar.gz")
	if err := os.WriteFile(src, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	out := fakeFree(t, 1<<30)
	dest := filepath.Join(root, "low")
	if err := sourceData(src, dest, root, sum); err != nil {
		t.Fatalf("可用空间不足时复制被中止: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "source", "COPYING"))
	if err != nil || string(got) != "license" {
		t.Fatalf("空间不足时复制结果: got %q error=%v", got, err)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if out.Len() == 0 {
		t.Fatal("空间不足时未输出警告")
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "警告：可用空间 1073741824 字节") {
			t.Fatalf("警告行格式不符: %q", line)
		}
	}
}

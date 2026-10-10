// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
	"golang.org/x/sys/unix"
)

func TestMain(m *testing.M) { os.Exit(workpath.Run(m)) }

func testDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Logf("警告：设置0755失败：%s：%v", dir, err)
	}
	return dir
}

type testNode struct {
	name string
	data string
	link string
	kind byte
}

func nodes() []testNode {
	var out []testNode
	for _, name := range mustFiles {
		out = append(out, testNode{name: name, data: "fixture:" + name})
	}
	return append(out,
		testNode{name: "jmods/java.base.jmod", data: "module-copy"},
		testNode{name: "man/man1/java.1", data: "manual"},
		testNode{name: "lib/src.zip", data: "sources"},
		testNode{name: "legal/java.compiler/LICENSE", link: "../java.base/LICENSE", kind: tar.TypeSymlink},
	)
}

func source(t *testing.T, list []testNode) (string, plan) {
	t.Helper()
	file := filepath.Join(testDir(t), "source.tar.gz")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, item := range list {
		kind := item.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		head := tar.Header{Name: "jdk/" + item.name, Mode: 0o755, Typeflag: kind,
			Linkname: item.link, ModTime: time.Unix(55555, 0)}
		if kind == tar.TypeReg {
			head.Size = int64(len(item.data))
		}
		if err := tw.WriteHeader(&head); err != nil {
			t.Fatal(err)
		}
		if kind == tar.TypeReg {
			if _, err := io.WriteString(tw, item.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return file, plan{Host: "fixture", Prefix: "jdk", SrcSHA: hex.EncodeToString(sum[:]), Stem: "fixture"}
}

func output(t *testing.T, dir string) map[string]string {
	t.Helper()
	file, err := os.Open(filepath.Join(dir, "fixture.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gr, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	out := make(map[string]string)
	for {
		head, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !head.ModTime.Equal(time.Unix(0, 0)) || head.Uid != 0 || head.Gid != 0 {
			t.Fatalf("归档元数据未规范化：%+v", head)
		}
		if head.Typeflag == tar.TypeDir {
			continue
		}
		if head.Typeflag != tar.TypeReg {
			t.Fatalf("输出链接没有隔离为普通文件：%+v", head)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		out[head.Name] = string(data)
	}
	return out
}

func TestReplay(t *testing.T) {
	in, spec := source(t, nodes())
	dir := testDir(t)
	first, next := filepath.Join(dir, "first"), filepath.Join(dir, "next")
	for _, dest := range []string{first, next} {
		if _, err := pack(context.Background(), spec, in, dest); err != nil {
			t.Fatal(err)
		}
		if err := verify(context.Background(), spec, in, dest); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"fixture.tar.gz", "fixture.tar.gz.part-000", "report.json"} {
		a, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(next, name))
		if err != nil || !bytes.Equal(a, b) {
			t.Fatalf("重放字节不同：%s err=%v", name, err)
		}
	}
	out := output(t, first)
	for _, name := range mustFiles {
		if out[outPrefix+"/"+name] != "fixture:"+name {
			t.Fatalf("保留文件字节改变：%s", name)
		}
	}
	for _, name := range []string{"jmods/java.base.jmod", "man/man1/java.1", "lib/src.zip"} {
		if _, ok := out[outPrefix+"/"+name]; ok {
			t.Fatalf("裁剪文件仍存在：%s", name)
		}
	}
	if out[outPrefix+"/legal/java.compiler/LICENSE"] != "fixture:legal/java.base/LICENSE" {
		t.Fatal("许可证链接字节丢失")
	}
}

func TestRejectArc(t *testing.T) {
	cases := []struct {
		name string
		edit func([]testNode) []testNode
	}{
		{"escape", func(list []testNode) []testNode { return append(list, testNode{name: "../escape", data: "bad"}) }},
		{"duplicate", func(list []testNode) []testNode { return append(list, list[0]) }},
		{"missing", func(list []testNode) []testNode { return list[1:] }},
		{"abs-link", func(list []testNode) []testNode { list[len(list)-1].link = "/outside"; return list }},
		{"escape-link", func(list []testNode) []testNode { list[len(list)-1].link = "../../../outside"; return list }},
		{"cycle", func(list []testNode) []testNode { list[len(list)-1].link = "LICENSE"; return list }},
		{"drop-link", func(list []testNode) []testNode { list[len(list)-1].link = "../../lib/src.zip"; return list }},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			in, spec := source(t, item.edit(nodes()))
			dest := filepath.Join(testDir(t), "dest")
			if _, err := pack(context.Background(), spec, in, dest); err == nil {
				t.Fatal("无效归档被发布")
			}
			if _, err := os.Lstat(dest); !os.IsNotExist(err) {
				t.Fatalf("失败后存在目标：%v", err)
			}
		})
	}
}

func TestKeepDest(t *testing.T) {
	in, spec := source(t, nodes())
	dir := testDir(t)
	dest := filepath.Join(dir, "dest")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	mark := filepath.Join(dest, "keep")
	if err := os.WriteFile(mark, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := pack(context.Background(), spec, in, dest); err == nil {
		t.Fatal("已有目标未拒绝")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(dest, link); err != nil {
		t.Fatal(err)
	}
	if _, err := pack(context.Background(), spec, in, link); err == nil {
		t.Fatal("目标符号链接未拒绝")
	}
	data, err := os.ReadFile(mark)
	if err != nil || string(data) != "existing" {
		t.Fatalf("已有文件被改动：data=%q err=%v", data, err)
	}
}

func TestBadSource(t *testing.T) {
	in, spec := source(t, nodes())
	spec.SrcSHA = "0000000000000000000000000000000000000000000000000000000000000000"
	dest := filepath.Join(testDir(t), "dest")
	if _, err := pack(context.Background(), spec, in, dest); err == nil {
		t.Fatal("输入摘要变化没有拒绝")
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Fatalf("摘要失败仍发布：%v", err)
	}
}

func TestCorrupt(t *testing.T) {
	in, spec := source(t, nodes())
	dest := filepath.Join(testDir(t), "dest")
	if _, err := pack(context.Background(), spec, in, dest); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dest, "fixture.tar.gz.part-000")
	if err := os.WriteFile(name, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verify(context.Background(), spec, in, dest); err == nil {
		t.Fatal("损坏分片没有拒绝")
	}
}

func TestCancel(t *testing.T) {
	in, spec := source(t, nodes())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dest := filepath.Join(testDir(t), "dest")
	if _, err := pack(ctx, spec, in, dest); err == nil {
		t.Fatal("已取消操作未拒绝")
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Fatalf("取消后仍发布：%v", err)
	}
}

func TestParts(t *testing.T) {
	in, spec := source(t, nodes())
	data, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	dir := testDir(t)
	mid := len(data) / 2
	spec.Source, spec.Sizes = "source.tar.gz", []int64{int64(mid), int64(len(data) - mid)}
	for index, body := range [][]byte{data[:mid], data[mid:]} {
		name := filepath.Join(dir, []string{"source.tar.gz.part-000", "source.tar.gz.part-001"}[index])
		if err := os.WriteFile(name, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dest := filepath.Join(testDir(t), "dest")
	if _, err := pack(context.Background(), spec, dir, dest); err != nil {
		t.Fatal(err)
	}
	if err := verify(context.Background(), spec, dir, dest); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "source.tar.gz.part-001")); err != nil {
		t.Fatal(err)
	}
	if err := verify(context.Background(), spec, dir, dest); err == nil {
		t.Fatal("缺失输入分片未拒绝")
	}
}

func TestPublish(t *testing.T) {
	dir := testDir(t)
	for _, name := range []string{"stage", "dest"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, "mark"), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fd, err := unix.Open(dir, unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if err := publish(fd, "stage", "dest"); err == nil {
		t.Fatal("并发发布覆盖已有目录")
	}
	for _, name := range []string{"stage", "dest"} {
		data, err := os.ReadFile(filepath.Join(dir, name, "mark"))
		if err != nil || string(data) != name {
			t.Fatalf("发布拒绝改动原件：name=%s data=%q err=%v", name, data, err)
		}
	}
}

func TestCLI(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"pack", "--host", "other", "--source", "x", "--dest", "y"},
		{"pack", "--host", "darwin-arm64"}, {"verify", "--source", "x", "--dest", "y", "extra"}} {
		if err := run(context.Background(), args); err == nil {
			t.Fatalf("无效命令被接受：%v", args)
		}
	}
	if err := run(context.Background(), []string{"--help"}); err != nil {
		t.Fatal(err)
	}
}

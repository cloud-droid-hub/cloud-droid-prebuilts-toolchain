// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestReject(t *testing.T) {
	dir := testDir(t)
	in := makeSource(t, dir)
	plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
	dest := filepath.Join(dir, "payload")
	if err := generate(context.Background(), plan, dir, dest, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := verify(context.Background(), plan, dir, dest, "", io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "payload.tar.gz.part-000"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verify(context.Background(), plan, dir, dest, "", io.Discard); err == nil {
		t.Fatal("损坏分片被接受")
	}
	plan.Inputs[0].SHA256 = "wrong"
	if err := generate(context.Background(), plan, dir, filepath.Join(dir, "bad"), io.Discard); err == nil {
		t.Fatal("不匹配的原始源被接受")
	}
	if _, err := os.Lstat(filepath.Join(dir, "bad")); !os.IsNotExist(err) {
		t.Fatalf("坏输入产生了发布目录：%v", err)
	}
}

func TestProtect(t *testing.T) {
	dir := testDir(t)
	in := makeSource(t, dir)
	plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
	dest := filepath.Join(dir, "existing")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	mark := filepath.Join(dest, "keep")
	if err := os.WriteFile(mark, []byte("existing data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generate(context.Background(), plan, dir, dest, io.Discard); err == nil {
		t.Fatal("已有目标被覆盖")
	}
	if got, err := os.ReadFile(mark); err != nil || string(got) != "existing data" {
		t.Fatalf("已有目标被改写：%q %v", got, err)
	}
	link := filepath.Join(dir, "parent-link")
	if err := os.Symlink(dest, link); err != nil {
		t.Fatal(err)
	}
	if err := generate(context.Background(), plan, dir, filepath.Join(link, "new"), io.Discard); err == nil {
		t.Fatal("符号链接输出父目录被接受")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := generate(ctx, plan, dir, filepath.Join(dir, "cancelled"), io.Discard); err == nil {
		t.Fatal("取消后仍成功发布")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if len(entry.Name()) >= 10 && entry.Name()[:10] == "trim-pack-" {
			t.Fatalf("取消留下候选：%s", entry.Name())
		}
	}
}

func zipLink(t *testing.T, name, link string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := zip.FileHeader{Name: name, Method: zip.Store}
	h.SetMode(os.ModeSymlink | 0o777)
	wr, err := zw.CreateHeader(&h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(wr, link); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestBadLinks(t *testing.T) {
	for _, target := range []string{"../../outside", "/outside", "alias"} {
		t.Run(target, func(t *testing.T) {
			dir := testDir(t)
			body := zipLink(t, "up/alias", target)
			if err := os.WriteFile(filepath.Join(dir, "link.zip"), body, 0o644); err != nil {
				t.Fatal(err)
			}
			in := input{Name: "link.zip", Kind: "zip", Prefix: "up", SHA256: fmt.Sprintf("%x", sha256.Sum256(body)), Rules: []rule{{Tree: true}}}
			plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in}}
			if err := generate(context.Background(), plan, dir, filepath.Join(dir, "bad"), io.Discard); err == nil {
				t.Fatal("越界或循环链接被接受")
			}
		})
	}
}

func TestAliases(t *testing.T) {
	dir := testDir(t)
	in := makeSource(t, dir)
	plan := spec{Host: "darwin-arm64", Comp: "sample", Prefix: "sample", Inputs: []input{in},
		Aliases: map[string]string{"bin/alias": "bin/tool"}}
	dest := filepath.Join(dir, "out")
	if err := generate(context.Background(), plan, dir, dest, io.Discard); err != nil {
		t.Fatal(err)
	}
	files := readFiles(t, dest)
	if files["sample/bin/alias"] != "original program" || files["sample/bin/tool"] != "original program" {
		t.Fatalf("普通文件别名字节不符：%v", files)
	}
	plan.Aliases = map[string]string{"LICENSE": "bin/tool"}
	if err := generate(context.Background(), plan, dir, filepath.Join(dir, "conflict"), io.Discard); err == nil {
		t.Fatal("别名覆盖原文件")
	}
}

func TestThin(t *testing.T) {
	dir := testDir(t)
	var buf bytes.Buffer
	for _, value := range []uint32{0xcafebabe, 2, 0x01000007, 3, 48, 32, 0, 0x0100000c, 0, 80, 32, 0} {
		if err := binary.Write(&buf, binary.BigEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, cpu := range []uint32{0x01000007, 0x0100000c} {
		for _, value := range []uint32{0xfeedfacf, cpu, 0, 2, 0, 0, 0, 0} {
			if err := binary.Write(&buf, binary.LittleEndian, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	name := filepath.Join(dir, "fat.bin")
	if err := os.WriteFile(name, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	item := node{Kind: tar.TypeReg, Size: int64(buf.Len())}
	if err := thin(&item, file); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(io.NewSectionReader(file, item.Off, item.Size))
	if err != nil || !bytes.Equal(got, buf.Bytes()[80:112]) {
		t.Fatalf("arm64切片错误：offset=%d size=%d err=%v", item.Off, item.Size, err)
	}
}

func TestBadMachO(t *testing.T) {
	dir := testDir(t)
	name := filepath.Join(dir, "broken.bin")
	if err := os.WriteFile(name, []byte{0xcf, 0xfa, 0xed, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	item := node{Kind: tar.TypeReg, Size: 4}
	if err := thin(&item, file); err == nil {
		t.Fatal("损坏Mach-O被当作普通数据保留")
	}
}

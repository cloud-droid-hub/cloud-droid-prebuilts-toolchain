// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHardAlias(t *testing.T) {
	tree := testTree(t)
	base := "usr/lib/aarch64-linux-gnu/"
	data := sampleTar(t, []sample{
		{"usr/bin/perl5", tar.TypeLink, "usr/bin/perl", "", 0755},
		{base + "other.so", tar.TypeLink, base + "alias.so", "", 0644},
		{base + "real.so", tar.TypeReg, "", "controlled ELF data", 0644},
		{base + "alias.so", tar.TypeLink, base + "real.so", "", 0644},
	})
	if err := tree.readTar(bytes.NewReader(data), debKeep); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "root")
	if err := tree.emit(dest, ""); err != nil {
		t.Fatal(err)
	}
	var prior []os.FileInfo
	for _, name := range []string{"real.so", "alias.so", "other.so"} {
		path := filepath.Join(dest, base, name)
		info, err := os.Lstat(path)
		if err != nil || regular(info) != nil {
			t.Fatal("归档链接没有生成独立普通副本", name, err)
		}
		for _, old := range prior {
			if os.SameFile(info, old) {
				t.Fatal("输出别名共享inode", name)
			}
		}
		prior = append(prior, info)
		got, err := os.ReadFile(path)
		if err != nil || string(got) != "controlled ELF data" {
			t.Fatal("归档链接内容不同", name, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(dest, "usr/bin")); !os.IsNotExist(err) {
		t.Fatal("未选程序进入目标数据", err)
	}
}

func TestHardBad(t *testing.T) {
	for _, link := range []string{"../outside", "/outside", "", "usr/lib/aarch64-linux-gnu/missing", "usr/lib/aarch64-linux-gnu/loop.so", "usr/include"} {
		t.Run(link, func(t *testing.T) {
			tree := testTree(t)
			data := sampleTar(t, []sample{
				{"usr/include/a.h", tar.TypeReg, "", "header", 0644},
				{"usr/lib/aarch64-linux-gnu/loop.so", tar.TypeLink, link, "", 0644},
			})
			err := tree.readTar(bytes.NewReader(data), debKeep)
			if err == nil {
				err = tree.emit(filepath.Join(t.TempDir(), "root"), "")
			}
			if err == nil {
				t.Fatal("越界、空目标、缺项、循环或目录硬链接被接受", link)
			}
		})
	}
}

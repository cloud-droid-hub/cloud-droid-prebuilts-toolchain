// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestArmModes(t *testing.T) {
	for _, kind := range []string{"source", "git"} {
		for _, mask := range []int{0o022, 0o077} {
			t.Run(fmt.Sprintf("%s/%03o", kind, mask), func(t *testing.T) {
				root := t.TempDir()
				items := []sample{
					{"sub/data.txt", tar.TypeReg, "", "content", 0o600},
					{"bin/tool", tar.TypeReg, "", "#!/bin/sh\n", 0o500},
				}
				var args []string
				var source string
				var digest [sha256.Size]byte
				inputs := make(map[string]os.FileInfo)
				if kind == "source" {
					var buf bytes.Buffer
					writer := gzip.NewWriter(&buf)
					if _, err := writer.Write(sampleTar(t, items)); err != nil {
						t.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					source = filepath.Join(root, "input.tar.gz")
					if err := os.WriteFile(source, buf.Bytes(), 0o644); err != nil {
						t.Fatal(err)
					}
					digest = sha256.Sum256(buf.Bytes())
					args = []string{kind, "--src", source, "--sha256", fmt.Sprintf("%x", digest)}
				} else {
					source = filepath.Join(root, "input")
					for _, item := range items {
						name := filepath.Join(source, item.name)
						if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(name, []byte(item.data), os.FileMode(item.mode)); err != nil {
							t.Fatal(err)
						}
						info, err := os.Lstat(name)
						if err != nil {
							t.Fatal(err)
						}
						inputs[item.name] = info
					}
					args = []string{kind, "--src", source}
				}
				before, err := os.Lstat(source)
				if err != nil {
					t.Fatal(err)
				}
				output := filepath.Join(root, "output")
				args = append(args, "--dest", output)
				old := syscall.Umask(mask)
				defer syscall.Umask(old)
				if err := run(args); err != nil {
					t.Fatal(err)
				}
				for _, part := range []string{".", "sub", "bin"} {
					info, err := os.Lstat(filepath.Join(output, part))
					if err != nil || !info.IsDir() || info.Mode().Perm() != 0o755 {
						t.Errorf("展开目录模式错误：%s info=%v err=%v", part, info, err)
					}
				}
				for _, item := range items {
					name := filepath.Join(output, item.name)
					info, err := os.Lstat(name)
					if err != nil {
						t.Fatal(err)
					}
					mode := os.FileMode(0o644)
					if item.mode&0o111 != 0 {
						mode = 0o755
					}
					if err := regular(info); err != nil || info.Mode().Perm() != mode {
						t.Errorf("展开文件模式或身份错误：%s mode=%v err=%v", item.name, info.Mode(), err)
					}
					data, err := os.ReadFile(name)
					if err != nil || string(data) != item.data {
						t.Errorf("展开内容错误：%s %v", item.name, err)
					}
					if kind == "git" {
						input := filepath.Join(source, item.name)
						src, err := os.Lstat(input)
						if err != nil || src.Mode().Perm() != os.FileMode(item.mode) || os.SameFile(src, info) ||
							!os.SameFile(inputs[item.name], src) || !src.ModTime().Equal(inputs[item.name].ModTime()) {
							t.Errorf("原输入模式变化或输出复用inode：%s %v", item.name, err)
						}
						data, err := os.ReadFile(input)
						if err != nil || string(data) != item.data {
							t.Errorf("原输入内容变化：%s %v", item.name, err)
						}
					}
				}
				after, err := os.Lstat(source)
				if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
					t.Errorf("原输入身份或模式变化：%v", err)
				}
				if kind == "source" {
					data, err := os.ReadFile(source)
					if err != nil || sha256.Sum256(data) != digest {
						t.Errorf("原归档内容变化：%v", err)
					}
				} else if err := run([]string{"git-check", "--src", source, "--dest", output}); err != nil {
					t.Errorf("读取规范权限的展开副本失败：%v", err)
				}
				tree, err := newTree("")
				if err != nil {
					t.Fatal(err)
				}
				info, statErr := tree.data.Stat()
				if statErr != nil || regular(info) != nil || info.Mode().Perm() != 0o644 {
					t.Errorf("归档工作文件模式错误：info=%v err=%v", info, statErr)
				}
				if err := tree.close(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

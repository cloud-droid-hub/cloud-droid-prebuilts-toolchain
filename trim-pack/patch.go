// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func patcher(ctx context.Context, raw *os.File, work string) (string, error) {
	gr, err := gzip.NewReader(raw)
	if err != nil {
		return "", err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	seen := make(map[string]bool)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		for _, name := range []string{"patchelf.cc", "patchelf.h", "elf.h"} {
			if h.Name != "patchelf-0.18.0/src/"+name {
				continue
			}
			if seen[name] || h.Typeflag != tar.TypeReg {
				return "", errors.New("patchelf固定源码类型或集合不符")
			}
			seen[name] = true
			file, err := newFile(filepath.Join(work, name))
			if err != nil {
				return "", err
			}
			_, err = io.Copy(file, ctxRead{ctx: ctx, in: tr})
			if err = errors.Join(err, file.Close()); err != nil {
				return "", err
			}
		}
	}
	if len(seen) != 3 {
		return "", errors.New("patchelf固定源码缺少必需文件")
	}
	root := os.Getenv("CDH_TOOLCHAIN_ROOT")
	if root == "" || !filepath.IsAbs(root) {
		return "", errors.New("LLVM Linux生成需要绝对路径CDH_TOOLCHAIN_ROOT登记原生LLVM22")
	}
	compiler := filepath.Join(root, "llvm22", "bin", "clang++")
	cmd := exec.CommandContext(ctx, compiler, "--version")
	version, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(version), "clang version 22.") {
		return "", fmt.Errorf("登记宿主C++编译器不是LLVM22：%s：%v", version, err)
	}
	linker := filepath.Join(root, "llvm22", "bin", "ld.lld")
	if runtime.GOOS == "darwin" {
		linker = filepath.Join(root, "llvm22", "bin", "ld64.lld")
	}
	version, err = exec.CommandContext(ctx, linker, "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(version), "LLD 22.") {
		return "", fmt.Errorf("登记宿主链接器不是配对LLD22：%s：%v", version, err)
	}
	name := filepath.Join(work, "patchelf")
	args := []string{"-std=c++17", "-O2", "--ld-path=" + linker, "-DPACKAGE_STRING=\"patchelf 0.18.0\"", "-o", name, filepath.Join(work, "patchelf.cc")}
	if runtime.GOOS == "darwin" {
		sdk, err := exec.CommandContext(ctx, "/usr/bin/xcrun", "--show-sdk-path").Output()
		if err != nil {
			return "", err
		}
		lib := filepath.Join(root, "llvm22", "lib")
		args = append(args, "-isysroot", strings.TrimSpace(string(sdk)), "-L"+lib, "-Wl,-rpath,"+lib, "-lc++abi", "-lunwind")
	}
	cmd = exec.CommandContext(ctx, compiler, args...)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "TMPDIR="+work, "TMP="+work, "TEMP="+work)
	if body, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("构建固定patchelf源码：%s：%w", body, err)
	}
	if err := os.Chmod(name, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "警告：新patchelf设置0755失败：%v\n", err)
	}
	body, err := exec.CommandContext(ctx, name, "--version").CombinedOutput()
	if err != nil || string(body) != "patchelf 0.18.0\n" {
		return "", fmt.Errorf("固定patchelf身份不符：%s：%v", body, err)
	}
	return name, nil
}

func patchFiles(ctx context.Context, b *bundle, data *os.File, tool, work string, plan spec) error {
	var names []string
	for name := range b.Items {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		item := b.Items[name]
		if item.Kind != tar.TypeReg {
			continue
		}
		file, err := elf.NewFile(io.NewSectionReader(data, item.Off, item.Size))
		if err != nil || file.Section(".dynamic") == nil {
			continue
		}
		candidate := filepath.Join(work, "patch.elf")
		out, err := newFile(candidate)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, io.NewSectionReader(data, item.Off, item.Size))
		if err = errors.Join(err, out.Close()); err != nil {
			return err
		}
		rel, err := filepath.Rel(filepath.FromSlash(path.Dir(name)), filepath.FromSlash("lib/cdh"))
		if err != nil {
			return err
		}
		rpath := "$ORIGIN"
		if rel != "." {
			rpath += "/" + filepath.ToSlash(rel)
		}
		steps := []string{rpath}
		if first, ok := plan.PreRPath[name]; ok {
			steps = append([]string{first}, steps...)
		}
		for _, value := range steps {
			args := []string{"--set-rpath", value, candidate}
			if plan.ForceR && !plan.RunPaths[name] {
				args = append([]string{"--force-rpath"}, args...)
			}
			cmd := exec.CommandContext(ctx, tool, args...)
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "TMPDIR="+work, "TMP="+work, "TEMP="+work)
			if body, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("修改%s RPATH=%s：%s：%w", name, value, body, err)
			}
		}
		patched, err := os.Open(candidate)
		if err != nil {
			return err
		}
		item.Off, err = data.Seek(0, io.SeekEnd)
		if err == nil {
			item.Size, err = io.Copy(data, ctxRead{ctx: ctx, in: patched})
		}
		if err = errors.Join(err, patched.Close(), os.Remove(candidate)); err != nil {
			return err
		}
	}
	return nil
}

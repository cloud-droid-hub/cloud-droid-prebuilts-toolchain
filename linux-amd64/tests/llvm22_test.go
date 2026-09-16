// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"crypto/sha256"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func llvm22Root(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires native linux/amd64")
	}
	root := os.Getenv("CDH_LLVM22_TEST_ROOT")
	if root == "" {
		t.Skip("set CDH_LLVM22_TEST_ROOT to an existing LLVM 22 installation")
	}
	if filepath.Base(root) != "llvm22" {
		t.Fatalf("invalid LLVM 22 installation root: %q", root)
	}
	llvm22Dir(t, root)
	return root
}

func llvm22File(t *testing.T, name string) {
	t.Helper()
	info, err := os.Lstat(name)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Sys().(*syscall.Stat_t).Nlink != 1 {
		t.Fatalf("not an independent regular file: %s", name)
	}
	f, err := elf.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	machine := f.Machine
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if machine != elf.EM_X86_64 {
		t.Fatalf("wrong machine %s: %s", machine, name)
	}
}

func llvm22Exec(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), name, args...)
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "LD_LIBRARY_PATH=") && !strings.HasPrefix(item, "LD_PRELOAD=") {
			cmd.Env = append(cmd.Env, item)
		}
	}
	out, err := cmd.CombinedOutput()
	t.Logf("command=%q exit=%v\n%s", cmd.Args, err, out)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLLVM22Tools(t *testing.T) {
	root := llvm22Root(t)
	for _, name := range []string{
		"clang", "clang++", "clang-22", "clang-cpp", "clang-scan-deps", "dsymutil",
		"ld.lld", "ld64.lld", "lld", "lld-link", "wasm-ld", "llvm-addr2line", "llvm-ar",
		"llvm-as", "llvm-bitcode-strip", "llvm-cov", "llvm-cxxfilt", "llvm-dis", "llvm-dwp",
		"llvm-install-name-tool", "llvm-libtool-darwin", "llvm-link", "llvm-lipo", "llvm-nm",
		"llvm-objcopy", "llvm-objdump", "llvm-otool", "llvm-profdata", "llvm-ranlib",
		"llvm-readelf", "llvm-readobj", "llvm-size", "llvm-strings", "llvm-strip", "llvm-symbolizer",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "bin", name)
			if name == "ld.lld" {
				target, err := os.Readlink(path)
				if err != nil || target != "lld" {
					t.Fatalf("registered link target=%q error=%v", target, err)
				}
				llvm22File(t, filepath.Join(root, "bin", "lld"))
			} else {
				llvm22File(t, path)
			}
			args := []string{"--version"}
			if name == "lld" {
				args = []string{"-flavor", "gnu", "--version"}
			}
			llvm22Exec(t, path, args...)
		})
	}
}

func TestLLVM22Load(t *testing.T) {
	root := llvm22Root(t)
	work := t.TempDir()
	source := filepath.Join(work, "load.c")
	program := filepath.Join(work, "load")
	body := `#include <dlfcn.h>
#include <stdio.h>
int main(int argc, char **argv) {
  if (argc != 2) return 1;
  void *lib = dlopen(argv[1], RTLD_NOW | RTLD_LOCAL);
  if (!lib) { fprintf(stderr, "%s\n", dlerror()); return 2; }
  if (puts(argv[1]) < 0) return 3;
  return dlclose(lib) ? 4 : 0;
}
`
	if err := os.WriteFile(source, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	clang := filepath.Join(root, "bin", "clang")
	llvm22File(t, clang)
	llvm22Exec(t, clang, "--ld-path="+filepath.Join(root, "bin", "ld.lld"), source, "-ldl", "-o", program)
	llvm22File(t, program)
	count, skipped := 0, 0
	err := filepath.WalkDir(filepath.Join(root, "lib"), func(path string, item os.DirEntry, err error) error {
		if err != nil || item.IsDir() {
			return err
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Sys().(*syscall.Stat_t).Nlink != 1 {
			t.Fatalf("library input is not an independent ordinary file: %s", path)
		}
		f, err := elf.Open(path)
		if err != nil {
			return nil
		}
		kind := f.Type
		if err := f.Close(); err != nil {
			return err
		}
		if kind != elf.ET_DYN {
			return nil
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		t.Run(name, func(t *testing.T) {
			count++
			llvm22File(t, path)
			if name == "lib/LLVMPolly.so" {
				skipped++
				t.Skip("LLVM host plugin; independent dlopen is not a registered consumer")
			}
			llvm22Exec(t, program, path)
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("no shared libraries tested")
	}
	t.Logf("shared_objects=%d independent_loads=%d plugins_skipped=%d separate_processes=true", count, count-skipped, skipped)
}

func TestLLVM22Copies(t *testing.T) {
	root := llvm22Root(t)
	for _, pair := range [][2]string{
		{"lib/cdh/libLLVM.so.22.1", "lib/libLLVM.so.22.1"},
		{"lib/cdh/libffi.so.8.2.0", "lib/libffi.so.8"},
		{"lib/cdh/libedit.so.2.0.72", "lib/libedit.so.2"},
		{"lib/cdh/libz3.so.4", "lib/libz3.so.4"},
		{"lib/cdh/libxml2.so.16.1.2", "lib/libxml2.so.16"},
	} {
		var infos [2]os.FileInfo
		var sums [2][32]byte
		for i, name := range pair {
			path := filepath.Join(root, filepath.FromSlash(name))
			llvm22File(t, path)
			info, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil || int64(len(data)) != info.Size() {
				t.Fatalf("read %s: size=%d error=%v", name, len(data), err)
			}
			infos[i] = info
			sums[i] = sha256.Sum256(data)
		}
		if os.SameFile(infos[0], infos[1]) || infos[0].Mode() != infos[1].Mode() || infos[0].Size() != infos[1].Size() || sums[0] != sums[1] {
			t.Fatalf("copy differs or shares file identity: %v", pair)
		}
		t.Logf("source=%s target=%s bytes=%d sha256=%x independent=true", pair[0], pair[1], infos[0].Size(), sums[0])
	}
}

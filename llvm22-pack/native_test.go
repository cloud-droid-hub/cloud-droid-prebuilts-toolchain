// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"context"
	"debug/elf"
	"debug/macho"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var runNative = func(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	return cmd.CombinedOutput()
}

func nativeCmd(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	if err := archNative(name); err != nil {
		return nil, err
	}
	return runNative(ctx, env, name, args...)
}

func archNative(name string) error {
	info, err := pathInfo(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(name)
		base := filepath.Base(name)
		allowed := filepath.Base(filepath.Dir(name)) == "bin" &&
			(base == "clang" && target == "clang-22" || base == "ld.lld" && target == "lld")
		if err != nil || !allowed {
			return fmt.Errorf("unregistered executable link: %s", name)
		}
		name = filepath.Join(filepath.Dir(name), target)
	}
	f, err := openInput(name)
	if err != nil {
		return err
	}
	defer f.Close()
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH != "arm64" {
			return fmt.Errorf("unsupported native host: %s/%s", runtime.GOOS, runtime.GOARCH)
		}
		bin, err := macho.NewFile(f)
		if err != nil {
			return err
		}
		if bin.Magic != macho.Magic64 || bin.Cpu != macho.CpuArm64 {
			return fmt.Errorf("executable is not native Darwin arm64: %s", name)
		}
	case "linux":
		machine := elf.EM_AARCH64
		if runtime.GOARCH == "amd64" {
			machine = elf.EM_X86_64
		} else if runtime.GOARCH != "arm64" {
			return fmt.Errorf("unsupported native host: %s/%s", runtime.GOOS, runtime.GOARCH)
		}
		bin, err := elf.NewFile(f)
		if err != nil {
			return err
		}
		if bin.Class != elf.ELFCLASS64 || bin.Data != elf.ELFDATA2LSB || bin.Machine != machine {
			return fmt.Errorf("executable is not native Linux %s: %s", runtime.GOARCH, name)
		}
	default:
		return fmt.Errorf("unsupported native host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return nil
}

func nativeRun(t *testing.T, env []string, name string, args ...string) {
	t.Helper()
	out, err := nativeCmd(t.Context(), env, name, args...)
	t.Logf("command=%q exit=%v\n%s", append([]string{name}, args...), err, out)
	if err != nil {
		t.Fatal(err)
	}
}

// 多别名加载是额外诊断，不代替登记消费者的原生验收。
func TestNativeAlias(t *testing.T) {
	root := os.Getenv("CDH_PACK_ROOT")
	if root == "" {
		t.Skip("set CDH_PACK_ROOT for native alias verification")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("candidate path must be absolute")
	}
	var libs []string
	var env []string
	link := "ld.lld"
	args := []string{}
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH != "arm64" {
			t.Fatal("unsupported Darwin architecture")
		}
		link = "ld64.lld"
		out, err := exec.CommandContext(t.Context(), "xcrun", "--sdk", "macosx", "--show-sdk-path").Output()
		if err != nil {
			t.Fatal(err)
		}
		args = append(args, "-isysroot", strings.TrimSpace(string(out)), "--ld-path="+filepath.Join(root, "bin", "ld64.lld"))
		libs = []string{"libclang.dylib", "libc++.1.0.dylib", "libc++.1.dylib", "libc++.dylib", "libc++abi.1.0.dylib", "libc++abi.1.dylib", "libc++abi.dylib", "libunwind.1.0.dylib", "libunwind.1.dylib", "libunwind.dylib"}
		env = []string{"DYLD_LIBRARY_PATH=" + filepath.Join(root, "lib")}
	case "linux":
		if runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
			t.Fatal("unsupported Linux architecture")
		}
		args = append(args, "--ld-path="+filepath.Join(root, "bin", "ld.lld"), "-ldl")
		libs = []string{"cdh/libclang-22.so.22", "libclang-22.so.1", "libclang-22.so.22", "cdh/libLLVM.so.22.1", "libLLVM-22.so", "cdh/libclang-cpp.so.22.1", "libclang-cpp.so.22.1"}
		if runtime.GOARCH == "arm64" {
			libs = append(libs, "cdh/libclang-22.so.1", "cdh/libLLVM-22.so", "libLLVM.so.22.1")
		}
	default:
		t.Fatal("unsupported native host")
	}
	if err := archNative(filepath.Join(root, "bin", link)); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	source := filepath.Join(work, "load.c")
	program := filepath.Join(work, "load")
	body := `#include <dlfcn.h>
#include <stdio.h>
#include <string.h>
typedef struct { const void *data; unsigned flags; } CXString;
int main(int argc, char **argv) {
  if (argc < 2 || argc > 64) return 1;
  void *libs[64] = {0};
  for (int n = 1; n < argc; n++) {
    libs[n] = dlopen(argv[n], RTLD_NOW | RTLD_LOCAL);
    if (!libs[n]) { fprintf(stderr, "%s: %s\n", argv[n], dlerror()); return 2; }
    printf("loaded %s\n", argv[n]);
    fflush(stdout);
    CXString (*version)(void) = dlsym(libs[n], "clang_getClangVersion");
    const char *(*text)(CXString) = dlsym(libs[n], "clang_getCString");
    void (*dispose)(CXString) = dlsym(libs[n], "clang_disposeString");
    if (version && text && dispose) {
      CXString value = version();
      const char *actual = text(value);
      printf("%s\n", actual);
      if (!strstr(actual, "clang version 22.1.")) return 3;
      dispose(value);
    }
  }
  for (int n = argc - 1; n > 0; n--) {
    if (dlclose(libs[n])) return 4;
  }
  return 0;
}
`
	if err := os.WriteFile(source, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	args = append(args, source, "-o", program)
	nativeRun(t, nil, filepath.Join(root, "bin", "clang"), args...)
	var paths []string
	for _, lib := range libs {
		name := filepath.Join(root, "lib", lib)
		info, err := os.Lstat(name)
		if err != nil || plain(info) != nil {
			t.Fatalf("library is not an independent regular file: %s", name)
		}
		paths = append(paths, name)
	}
	for _, name := range paths {
		t.Run(filepath.Base(name), func(t *testing.T) {
			nativeRun(t, env, program, name)
		})
	}
	t.Run("together", func(t *testing.T) {
		nativeRun(t, env, program, paths...)
	})
}

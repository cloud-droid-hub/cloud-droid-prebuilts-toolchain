// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"context"
	"debug/elf"
	"debug/macho"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type probeCmd struct {
	ctx  context.Context
	log  io.Writer
	env  []string
	work string
}

func probeEnv(work, lib string) []string {
	blocked := map[string]bool{
		"TMPDIR": true, "TMP": true, "TEMP": true, "LIBCLANG_PATH": true,
		"LD_PRELOAD": true, "LD_LIBRARY_PATH": true, "DYLD_LIBRARY_PATH": true,
		"DYLD_INSERT_LIBRARIES": true, "CPATH": true, "C_INCLUDE_PATH": true,
		"CPLUS_INCLUDE_PATH": true, "LIBRARY_PATH": true, "COMPILER_PATH": true,
		"GCC_EXEC_PREFIX": true, "SDKROOT": true,
	}
	var env []string
	for _, item := range os.Environ() {
		name, _, _ := strings.Cut(item, "=")
		if !blocked[name] {
			env = append(env, item)
		}
	}
	return append(env, "TMPDIR="+work, "TMP="+work, "TEMP="+work, "LIBCLANG_PATH="+lib)
}

func (p probeCmd) run(name string, args ...string) (string, error) {
	if err := freeSpace(p.work, probeSpace); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(p.ctx, 2*time.Minute)
	defer cancel()
	if err := writeLog(p.log, "exec=%q\n", append([]string{name}, args...)); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env = p.work, p.env
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		code = -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
	}
	logErr := writeLog(p.log, "%sexec_exit=%d error=%v\n", out, code, err)
	if err != nil {
		return string(out), errors.Join(fmt.Errorf("native command failed: %s: %w", name, err), logErr)
	}
	return string(out), logErr
}

func binaryArch(name string, cfg hostCfg) error {
	info, err := os.Stat(name)
	if err != nil || plain(info) != nil {
		return fmt.Errorf("binary is not an independent regular file: %s", name)
	}
	if cfg.goos == "darwin" {
		f, err := macho.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		if f.Cpu != macho.CpuArm64 {
			return fmt.Errorf("Mach-O CPU %v is not arm64: %s", f.Cpu, name)
		}
		return nil
	}
	f, err := elf.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	want := elf.EM_AARCH64
	if cfg.arch == "amd64" {
		want = elf.EM_X86_64
	}
	if f.Class != elf.ELFCLASS64 || f.Machine != want {
		return fmt.Errorf("ELF machine %v is not %v: %s", f.Machine, want, name)
	}
	return nil
}

func runProbe(ctx context.Context, cfg hostCfg, root, work string, log io.Writer) error {
	libdir := filepath.Join(root, "lib", "cdh")
	libname := "libclang-22.so.22"
	if cfg.goos == "darwin" {
		libdir, libname = filepath.Join(root, "lib"), "libclang.dylib"
	}
	if err := realDir(libdir); err != nil {
		return err
	}
	lib := filepath.Join(libdir, libname)
	info, err := os.Lstat(lib)
	if err != nil || plain(info) != nil {
		return fmt.Errorf("libclang is not a regular one-link file: %s", lib)
	}
	p := probeCmd{ctx, log, probeEnv(work, libdir), work}
	if err := writeLog(log, "LIBCLANG_PATH=%q\n", libdir); err != nil {
		return err
	}
	bin := filepath.Join(root, "bin")
	tools := []string{"clang", "clang++", cfg.lld, "llvm-ar", "llvm-ranlib", "llvm-nm", "llvm-objdump", "llvm-objcopy", "llvm-strip"}
	if cfg.goos == "linux" {
		tools = append(tools, "llvm-readelf")
	}
	for _, name := range tools {
		tool := filepath.Join(bin, name)
		if err := binaryArch(tool, cfg); err != nil {
			return err
		}
		out, err := p.run(tool, "--version")
		if err != nil {
			return err
		}
		if !strings.Contains(out, cfg.version) {
			return fmt.Errorf("tool version differs from LLVM %s: %s", cfg.version, tool)
		}
	}
	var common []string
	if cfg.goos == "darwin" {
		sdk, err := p.run("/usr/bin/xcrun", "--sdk", "macosx", "--show-sdk-path")
		if err != nil {
			return err
		}
		sdk = strings.TrimSpace(sdk)
		if !filepath.IsAbs(sdk) {
			return fmt.Errorf("macOS SDK is not an absolute path: %q", sdk)
		}
		common = append(common, "-isysroot", sdk)
	}
	link := "--ld-path=" + filepath.Join(bin, cfg.lld)
	files := []struct{ name, body string }{
		{"c.c", "#include <stddef.h>\n#include <stdio.h>\nint main(void) { puts(\"native C passed\"); return sizeof(size_t) == 8 ? 0 : 1; }\n"},
		{"cpp.cc", "#include <string>\n#include <vector>\n#include <cstdio>\nint main() { std::vector<std::string> v{\"llvm22\"}; try { throw v.at(0); } catch (const std::string& s) { puts(\"native C++ passed\"); return s == \"llvm22\" ? 0 : 1; } }\n"},
		{"answer.c", "int answer(void) { return 42; }\n"},
		{"archive.c", "int answer(void);\nint main(void) { return answer() == 42 ? 0 : 1; }\n"},
		{"load.c", loadSource(cfg.version)},
	}
	for _, file := range files {
		if err := writeNew(filepath.Join(work, file.name), file.body); err != nil {
			return err
		}
	}
	for _, item := range [][2]string{{"c.c", "clang"}, {"cpp.cc", "clang++"}} {
		source := filepath.Join(work, item[0])
		output := source + ".out"
		args := append(append([]string{}, common...), link, source, "-o", output)
		if _, err := p.run(filepath.Join(bin, item[1]), args...); err != nil {
			return err
		}
		if err := binaryArch(output, cfg); err != nil {
			return err
		}
		if _, err := p.run(output); err != nil {
			return err
		}
		stripped := output + ".stripped"
		if _, err := p.run(filepath.Join(bin, "llvm-objcopy"), "--strip-debug", output, stripped); err != nil {
			return err
		}
		if _, err := p.run(filepath.Join(bin, "llvm-strip"), stripped); err != nil {
			return err
		}
		if err := binaryArch(stripped, cfg); err != nil {
			return err
		}
		if _, err := p.run(stripped); err != nil {
			return err
		}
	}
	object, archive := filepath.Join(work, "answer.o"), filepath.Join(work, "libanswer.a")
	args := append(append([]string{}, common...), "-c", filepath.Join(work, "answer.c"), "-o", object)
	if _, err := p.run(filepath.Join(bin, "clang"), args...); err != nil {
		return err
	}
	if err := binaryArch(object, cfg); err != nil {
		return err
	}
	for _, args := range [][]string{{"llvm-ar", "rcs", archive, object}, {"llvm-ranlib", archive}, {"llvm-objdump", "--syms", object}} {
		if _, err := p.run(filepath.Join(bin, args[0]), args[1:]...); err != nil {
			return err
		}
	}
	if cfg.goos == "linux" {
		if _, err := p.run(filepath.Join(bin, "llvm-readelf"), "-h", object); err != nil {
			return err
		}
	}
	members, err := p.run(filepath.Join(bin, "llvm-ar"), "t", archive)
	if err != nil || strings.TrimSpace(members) != "answer.o" {
		return fmt.Errorf("archive member check failed: %q: %v", members, err)
	}
	symbols, err := p.run(filepath.Join(bin, "llvm-nm"), archive)
	if err != nil || !strings.Contains(symbols, "answer") {
		return fmt.Errorf("archive symbol check failed: %v", err)
	}
	for _, item := range []string{"archive", "load"} {
		program := filepath.Join(work, item)
		args := append(append([]string{}, common...), link, filepath.Join(work, item+".c"))
		if item == "archive" {
			args = append(args, archive)
		} else if cfg.goos == "linux" {
			args = append(args, "-ldl")
		}
		args = append(args, "-o", program)
		if _, err := p.run(filepath.Join(bin, "clang"), args...); err != nil {
			return err
		}
		if err := binaryArch(program, cfg); err != nil {
			return err
		}
		var runArgs []string
		if item == "load" {
			runArgs = append(runArgs, lib)
		}
		if _, err := p.run(program, runArgs...); err != nil {
			return err
		}
	}
	return nil
}

func loadSource(version string) string {
	return `#include <dlfcn.h>
#include <stdio.h>
#include <string.h>
typedef struct { const void *data; unsigned flags; } CXString;
int main(int argc, char **argv) {
  if (argc != 2) return 1;
  void *lib = dlopen(argv[1], RTLD_NOW | RTLD_LOCAL);
  if (!lib) { fprintf(stderr, "%s\n", dlerror()); return 2; }
  CXString (*version)(void) = dlsym(lib, "clang_getClangVersion");
  const char *(*text)(CXString) = dlsym(lib, "clang_getCString");
  void (*dispose)(CXString) = dlsym(lib, "clang_disposeString");
  if (!version || !text || !dispose) return 3;
  CXString value = version();
  const char *actual = text(value);
  if (!actual) return 4;
  puts(actual);
  int result = strstr(actual, "clang version ` + version + `") ? 0 : 5;
  dispose(value);
  if (dlclose(lib)) return 6;
  return result;
}
`
}

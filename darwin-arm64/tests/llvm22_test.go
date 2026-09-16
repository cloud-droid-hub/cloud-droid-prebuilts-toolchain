// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"context"
	"crypto/sha256"
	"debug/macho"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/install"
)

func TestLLVM22Size(t *testing.T) {
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var archive catalog.Archive
	for _, item := range catalog.Current().Archives {
		if item.Dir == "llvm22" {
			archive = item
			break
		}
	}
	if len(archive.Parts) == 0 {
		t.Fatal("LLVM 22 archive is not registered")
	}
	total := sha256.New()
	var size int64
	for _, part := range archive.Parts {
		path := filepath.Join(filepath.Dir(work), filepath.FromSlash(part.Path))
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 || info.Sys().(*syscall.Stat_t).Nlink != 1 {
			t.Fatalf("archive part is not an independent regular file with mode 0644: %s", part.Path)
		}
		if info.Size() != part.Size || info.Size() > catalog.MaxPart {
			t.Fatalf("part %s size=%d registered=%d", part.Path, info.Size(), part.Size)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		n, readErr := io.Copy(io.MultiWriter(total, digest), file)
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read part %s: %v; close: %v", part.Path, readErr, closeErr)
		}
		if got := hex.EncodeToString(digest.Sum(nil)); got != part.SHA256 {
			t.Fatalf("part %s SHA-256=%s", part.Path, got)
		}
		size += n
	}
	if got := hex.EncodeToString(total.Sum(nil)); got != archive.SHA256 {
		t.Fatalf("LLVM 22 archive SHA-256=%s", got)
	}
	if len(archive.Parts) > 4 || size > 4*catalog.MaxPart {
		t.Fatalf("LLVM 22 archive too large: %d parts, %d bytes", len(archive.Parts), size)
	}
	t.Logf("LLVM 22: %d parts, %d bytes", len(archive.Parts), size)
}

func TestLLVM22Build(t *testing.T) {
	spec := catalog.Current()
	for _, archive := range spec.Archives {
		if archive.Dir == "llvm22" {
			spec.Archives = []catalog.Archive{archive}
			break
		}
	}
	if len(spec.Archives) != 1 {
		t.Fatal("LLVM 22 archive is not registered")
	}
	root := os.Getenv("CDH_LLVM22_TEST_ROOT")
	if root == "" {
		work, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(t.TempDir(), "tools")
		if err := install.Run(context.Background(), filepath.Dir(work), dest, spec); err != nil {
			t.Fatal(err)
		}
		root = filepath.Join(dest, "llvm22")
	}
	if !filepath.IsAbs(root) || filepath.Base(root) != "llvm22" {
		t.Fatalf("LLVM 22 root is not an absolute installation path: %q", root)
	}
	if err := install.Check(context.Background(), filepath.Dir(root), spec); err != nil {
		t.Fatal(err)
	}
	sdk := strings.TrimSpace(llvm22Run(t, "xcrun", "--sdk", "macosx", "--show-sdk-path"))
	if !filepath.IsAbs(sdk) {
		t.Fatalf("macOS SDK path=%q", sdk)
	}
	bin := filepath.Join(root, "bin")
	for _, name := range []string{"clang", "clang++", "ld64.lld", "llvm-ar", "llvm-ranlib", "llvm-nm", "llvm-objdump", "llvm-objcopy", "llvm-strip"} {
		llvm22File(t, root, filepath.Join(bin, name))
	}
	work := t.TempDir()
	for _, probe := range []struct {
		name string
		tool string
		body string
	}{
		{"c.c", "clang", "#include <stddef.h>\nint main(void) { return sizeof(size_t) == 8 ? 0 : 1; }\n"},
		{"cpp.cc", "clang++", "#include <string>\n#include <vector>\nint main() { std::vector<std::string> v{\"llvm22\"}; try { throw v.at(0); } catch (const std::string& s) { return s == \"llvm22\" ? 0 : 1; } }\n"},
	} {
		source := filepath.Join(work, probe.name)
		program := source + ".out"
		if err := os.WriteFile(source, []byte(probe.body), 0o644); err != nil {
			t.Fatal(err)
		}
		llvm22Run(t, filepath.Join(bin, probe.tool), "-isysroot", sdk, "--ld-path="+filepath.Join(bin, "ld64.lld"), source, "-o", program)
		llvm22Run(t, program)
		llvm22Mach(t, program)
		stripped := program + ".stripped"
		llvm22Run(t, filepath.Join(bin, "llvm-objcopy"), "--strip-debug", program, stripped)
		llvm22Run(t, filepath.Join(bin, "llvm-strip"), stripped)
		llvm22Run(t, stripped)
	}
	source := filepath.Join(work, "answer.c")
	object := filepath.Join(work, "answer.o")
	archive := filepath.Join(work, "libanswer.a")
	if err := os.WriteFile(source, []byte("int answer(void) { return 42; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	llvm22Run(t, filepath.Join(bin, "clang"), "-isysroot", sdk, "-c", source, "-o", object)
	llvm22Run(t, filepath.Join(bin, "llvm-ar"), "rcs", archive, object)
	llvm22Run(t, filepath.Join(bin, "llvm-ranlib"), archive)
	if got := strings.TrimSpace(llvm22Run(t, filepath.Join(bin, "llvm-ar"), "t", archive)); got != "answer.o" {
		t.Fatalf("archive members=%q", got)
	}
	if got := llvm22Run(t, filepath.Join(bin, "llvm-nm"), archive); !strings.Contains(got, "_answer") {
		t.Fatalf("archive symbols=%q", got)
	}
	llvm22Run(t, filepath.Join(bin, "llvm-objdump"), "--syms", object)
	source = filepath.Join(work, "load.c")
	program := filepath.Join(work, "load")
	body := `#include <dlfcn.h>
#include <stdio.h>
#include <string.h>
#include <clang-c/Index.h>
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
  printf("%s\n", actual);
  int result = strstr(actual, "clang version 22.1.8") ? 0 : 4;
  dispose(value);
  if (dlclose(lib)) return 5;
  return result;
}
`
	if err := os.WriteFile(source, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	libclang := filepath.Join(root, "lib", "libclang.dylib")
	llvm22File(t, root, libclang)
	llvm22Run(t, filepath.Join(bin, "clang"), "-isysroot", sdk, "-I"+filepath.Join(root, "include"), "--ld-path="+filepath.Join(bin, "ld64.lld"), source, "-o", program)
	t.Log(strings.TrimSpace(llvm22Run(t, program, libclang)))
}

func llvm22Run(t *testing.T, program string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), program, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", program, args, err, output)
	}
	return string(output)
}

func llvm22File(t *testing.T, root, path string) {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(real, root+string(filepath.Separator)) {
		t.Fatalf("tool path leaves LLVM 22: %s", path)
	}
	info, err := os.Stat(real)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Sys().(*syscall.Stat_t).Nlink != 1 {
		t.Fatalf("tool is not an independent regular file: %s", path)
	}
}

func llvm22Mach(t *testing.T, path string) {
	t.Helper()
	binary, err := macho.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	cpu := binary.Cpu
	if err := binary.Close(); err != nil {
		t.Fatal(err)
	}
	if cpu != macho.CpuArm64 {
		t.Fatalf("program CPU=%v", cpu)
	}
}

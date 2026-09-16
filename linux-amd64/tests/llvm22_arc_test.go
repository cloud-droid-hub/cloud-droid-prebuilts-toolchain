// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package tests

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/ulikunitz/xz"
)

func llvm22Input(t *testing.T, repo string, archive catalog.Archive) io.Reader {
	t.Helper()
	if len(archive.Parts) == 0 {
		t.Fatalf("LLVM 22 parts=%d", len(archive.Parts))
	}
	total := sha256.New()
	var readers []io.Reader
	var infos []os.FileInfo
	for _, part := range archive.Parts {
		if filepath.IsAbs(part.Path) || filepath.Clean(part.Path) != part.Path || strings.HasPrefix(part.Path, "../") {
			t.Fatalf("invalid part path: %s", part.Path)
		}
		name := filepath.Join(repo, filepath.FromSlash(part.Path))
		llvm22Dir(t, filepath.Dir(name))
		info, err := os.Lstat(name)
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 || info.Sys().(*syscall.Stat_t).Nlink != 1 {
			t.Fatalf("part is not an independent ordinary file with mode 0644: %s", name)
		}
		for _, prior := range infos {
			if os.SameFile(prior, info) {
				t.Fatalf("parts share identity: %s", name)
			}
		}
		infos = append(infos, info)
		if info.Size() != part.Size || info.Size() > catalog.MaxPart {
			t.Fatalf("part size=%d registered=%d: %s", info.Size(), part.Size, name)
		}
		f, err := os.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := f.Close(); err != nil {
				t.Error(err)
			}
		})
		now, err := f.Stat()
		if err != nil || !os.SameFile(info, now) {
			t.Fatalf("part identity changed: %s", name)
		}
		digest := sha256.New()
		n, err := io.Copy(io.MultiWriter(total, digest), f)
		if err != nil || n != part.Size || fmt.Sprintf("%x", digest.Sum(nil)) != part.SHA256 {
			t.Fatalf("part bytes differ: %s: size=%d error=%v", name, n, err)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		readers = append(readers, f)
	}
	if fmt.Sprintf("%x", total.Sum(nil)) != archive.SHA256 {
		t.Fatal("LLVM 22 archive digest differs")
	}
	stream, err := xz.NewReader(io.MultiReader(readers...))
	if err != nil {
		t.Fatal(err)
	}
	return stream
}

func llvm22Dir(t *testing.T, root string) {
	t.Helper()
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		t.Fatalf("directory path is not absolute and clean: %s", root)
	}
	for name := root; ; name = filepath.Dir(name) {
		info, err := os.Lstat(name)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("path is not a real directory: %s: %v", name, err)
		}
		if name == filepath.Dir(name) {
			return
		}
	}
}

func llvm22Archive(t *testing.T) (string, catalog.Archive) {
	t.Helper()
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(work)
	var archive catalog.Archive
	for _, item := range catalog.Current().Archives {
		if item.Dir == "llvm22" {
			archive = item
			break
		}
	}
	if root := os.Getenv("CDH_LLVM22_PARTS"); root != "" {
		llvm22Dir(t, root)
		repo = root
		for i := range archive.Parts {
			archive.Parts[i].Path = filepath.Base(archive.Parts[i].Path)
		}
	}
	return repo, archive
}

func TestLLVM22Split(t *testing.T) {
	var packed bytes.Buffer
	w, err := xz.NewWriter(&packed)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("two ordinary parts preserve the compressed stream")
	if _, err := w.Write(want); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data := packed.Bytes()
	archive := catalog.Archive{SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	repo := t.TempDir()
	for i, body := range [][]byte{data[:len(data)/2], data[len(data)/2:]} {
		name := fmt.Sprintf("sample.tar.xz.part-%03d", i)
		if err := os.WriteFile(filepath.Join(repo, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
		archive.Parts = append(archive.Parts, catalog.Part{
			Path: name, Size: int64(len(body)), SHA256: fmt.Sprintf("%x", sha256.Sum256(body)),
		})
	}
	got, err := io.ReadAll(llvm22Input(t, repo, archive))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("decoded bytes=%q error=%v", got, err)
	}
}

type llvm22Entry struct {
	size int64
	mode int64
	sum  string
}

func TestLLVM22Arc(t *testing.T) {
	repo, archive := llvm22Archive(t)
	if len(archive.Parts) > 2 {
		t.Fatalf("LLVM 22 parts=%d", len(archive.Parts))
	}
	xr := llvm22Input(t, repo, archive)
	tr := tar.NewReader(xr)
	items := make(map[string]llvm22Entry)
	seen := make(map[string]bool)
	links := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(h.Name, "/")
		if path.Clean(name) != name || (name != archive.Prefix && !strings.HasPrefix(name, archive.Prefix+"/")) || seen[name] {
			t.Fatalf("invalid or duplicate archive path: %s", name)
		}
		seen[name] = true
		switch h.Typeflag {
		case tar.TypeDir:
			if h.Mode != 0o755 {
				t.Fatalf("directory mode=%#o: %s", h.Mode, name)
			}
		case tar.TypeReg, tar.TypeRegA:
			if h.Mode != 0o644 && h.Mode != 0o755 {
				t.Fatalf("file mode=%#o: %s", h.Mode, name)
			}
			digest := sha256.New()
			n, err := io.Copy(digest, tr)
			if err != nil || n != h.Size {
				t.Fatalf("archive file read: %s: size=%d error=%v", name, n, err)
			}
			items[strings.TrimPrefix(name, archive.Prefix+"/")] = llvm22Entry{n, h.Mode, fmt.Sprintf("%x", digest.Sum(nil))}
		case tar.TypeSymlink:
			allowed := name == archive.Prefix+"/bin/clang" && h.Linkname == "clang-22" ||
				name == archive.Prefix+"/bin/ld.lld" && h.Linkname == "lld"
			if !allowed {
				t.Fatalf("unregistered archive link: %s -> %s", name, h.Linkname)
			}
			links++
		default:
			t.Fatalf("unsupported archive type=%d: %s", h.Typeflag, name)
		}
	}
	if _, err := io.Copy(io.Discard, xr); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{
		{"bin/clang++", "bin/clang"}, {"bin/clang-22", "bin/clang"}, {"bin/clang-cpp", "bin/clang"},
		{"bin/llvm-ranlib", "bin/llvm-ar"}, {"bin/llvm-strip", "bin/llvm-objcopy"},
		{"lib/libLLVM-22.so", "lib/cdh/libLLVM.so.22.1"}, {"lib/libLLVM.so.22.1", "lib/cdh/libLLVM.so.22.1"},
		{"lib/libclang-22.so.1", "lib/cdh/libclang-22.so.22"}, {"lib/libclang-22.so.22", "lib/cdh/libclang-22.so.22"},
		{"lib/libclang-cpp.so.22.1", "lib/cdh/libclang-cpp.so.22.1"},
		{"lib/libffi.so.8", "lib/cdh/libffi.so.8.2.0"}, {"lib/libedit.so.2", "lib/cdh/libedit.so.2.0.72"},
		{"lib/libz3.so.4", "lib/cdh/libz3.so.4"}, {"lib/libxml2.so.16", "lib/cdh/libxml2.so.16.1.2"},
	} {
		left, ok := items[pair[0]]
		right, found := items[pair[1]]
		if !ok || !found || left != right {
			t.Fatalf("ordinary alias bytes or mode differ: %v", pair)
		}
	}
	work, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	licenses, err := os.ReadDir(filepath.Join(filepath.Dir(work), "licenses", "llvm22"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range licenses {
		name := filepath.Join(filepath.Dir(work), "licenses", "llvm22", item.Name())
		info, err := os.Lstat(name)
		if err != nil || !info.Mode().IsRegular() || info.Sys().(*syscall.Stat_t).Nlink != 1 {
			t.Fatalf("license is not an independent ordinary file: %s: %v", name, err)
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		want := llvm22Entry{int64(len(data)), 0o644, fmt.Sprintf("%x", sha256.Sum256(data))}
		if items["lib/cdh/LICENSES/"+item.Name()] != want {
			t.Fatalf("license bytes or mode differ: %s", item.Name())
		}
	}
	if len(items) != 419 || links != 1 || len(licenses) != 28 {
		t.Fatalf("regular=%d links=%d licenses=%d", len(items), links, len(licenses))
	}
	var size int64
	for _, part := range archive.Parts {
		size += part.Size
	}
	t.Logf("parts=%d bytes=%d regular=%d links=%d licenses=%d", len(archive.Parts), size, len(items), links, len(licenses))
}

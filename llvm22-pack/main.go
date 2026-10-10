// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ulikunitz/xz"
)

type entry struct {
	head tar.Header
	off  int64
	sum  [32]byte
	from string
}

type bundle struct {
	items  map[string]*entry
	names  []string
	prefix string
	bytes  int64
	info   os.FileInfo
}

var pathInfo = os.Lstat

func main() {
	start := time.Now()
	err := logf("command=%q started=%s\n", os.Args, start.UTC().Format(time.RFC3339))
	if err == nil {
		if len(os.Args) < 5 {
			err = errors.New("usage: llvm22-pack pack input.tar.xz prefix new-directory [dest=source ...] | verify input.tar.xz output.tar.xz prefix [dest=source ...]")
		} else {
			switch os.Args[1] {
			case "pack":
				err = pack(os.Args[2], os.Args[3], os.Args[4], os.Args[5:]...)
			case "verify":
				err = verify(os.Args[2], os.Args[3], os.Args[4], os.Args[5:]...)
			default:
				err = errors.New("unknown operation")
			}
		}
	}
	if err == nil {
		err = logf("exit=0 elapsed=%s\n", time.Since(start))
	}
	if err != nil {
		if err := writeLog(os.Stderr, "exit=1 error=%v elapsed=%s\n", err, time.Since(start)); err != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func logf(format string, args ...any) error {
	return writeLog(os.Stdout, format, args...)
}

func writeLog(out io.Writer, format string, args ...any) error {
	text := fmt.Sprintf(format, args...)
	n, err := io.WriteString(out, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}

func inPrefix(name, prefix string) bool {
	return name == prefix || strings.HasPrefix(name, prefix+"/")
}

func arcName(name, prefix string) (string, error) {
	clean := strings.TrimSuffix(name, "/")
	if clean == "" || path.IsAbs(clean) || path.Clean(clean) != clean || !inPrefix(clean, prefix) || strings.ContainsAny(clean, "\\\t\r\n") {
		return "", fmt.Errorf("invalid archive path: %q", name)
	}
	return clean, nil
}

func scan(r io.Reader, prefix string, data io.Writer) (*bundle, error) {
	if prefix == "" || path.Base(prefix) != prefix || prefix == "." || prefix == ".." {
		return nil, errors.New("prefix must be one directory name")
	}
	b := &bundle{items: make(map[string]*entry), prefix: prefix}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name, err := arcName(h.Name, prefix)
		if err != nil {
			return nil, err
		}
		if _, ok := b.items[name]; ok {
			return nil, fmt.Errorf("duplicate archive path: %s", name)
		}
		if h.Mode&^int64(0o777) != 0 {
			return nil, fmt.Errorf("unsupported file mode: %s", name)
		}
		e := &entry{head: *h, off: b.bytes}
		e.head.Name = name
		switch h.Typeflag {
		case tar.TypeDir, tar.TypeSymlink, tar.TypeLink:
			if h.Size != 0 {
				return nil, fmt.Errorf("non-file data: %s", name)
			}
		case tar.TypeReg, tar.TypeRegA:
			e.head.Typeflag = tar.TypeReg
			hash := sha256.New()
			var out io.Writer = hash
			if data != nil {
				out = io.MultiWriter(data, hash)
			}
			n, err := io.Copy(out, tr)
			if err != nil || n != h.Size {
				return nil, fmt.Errorf("read file %s: bytes=%d expected=%d error=%v", name, n, h.Size, err)
			}
			copy(e.sum[:], hash.Sum(nil))
			b.bytes += n
		default:
			return nil, fmt.Errorf("unsupported archive type %d: %s", h.Typeflag, name)
		}
		b.items[name] = e
		b.names = append(b.names, name)
	}
	if len(b.names) == 0 {
		return nil, errors.New("empty archive")
	}
	sort.Strings(b.names)
	for _, name := range b.names {
		for dir := path.Dir(name); inPrefix(dir, prefix); dir = path.Dir(dir) {
			if e, ok := b.items[dir]; ok && e.head.Typeflag != tar.TypeDir {
				return nil, fmt.Errorf("non-directory ancestor: %s", dir)
			}
		}
		e := b.items[name]
		if e.head.Typeflag == tar.TypeSymlink || e.head.Typeflag == tar.TypeLink {
			if _, err := b.resolve(name); err != nil {
				return nil, err
			}
		}
	}
	return b, nil
}

func (b *bundle) resolve(name string) (*entry, error) {
	seen := make(map[string]bool)
	for {
		if seen[name] {
			return nil, fmt.Errorf("archive link cycle: %s", name)
		}
		seen[name] = true
		e, ok := b.items[name]
		if !ok {
			return nil, fmt.Errorf("missing archive target: %s", name)
		}
		switch e.head.Typeflag {
		case tar.TypeReg:
			return e, nil
		case tar.TypeSymlink, tar.TypeLink:
			target := e.head.Linkname
			if target == "" || path.IsAbs(target) || strings.ContainsAny(target, "\\\t\r\n") {
				return nil, fmt.Errorf("invalid archive target: %s -> %q", name, target)
			}
			if e.head.Typeflag == tar.TypeSymlink {
				target = path.Join(path.Dir(name), target)
			} else {
				target = path.Clean(target)
			}
			if !inPrefix(target, b.prefix) {
				return nil, fmt.Errorf("archive target escapes prefix: %s", name)
			}
			name = target
		default:
			return nil, fmt.Errorf("archive target is not a regular file: %s", name)
		}
	}
}

func (b *bundle) allowed(e *entry) bool {
	if e.head.Typeflag != tar.TypeSymlink {
		return false
	}
	return e.head.Name == b.prefix+"/bin/clang" && e.head.Linkname == "clang-22" ||
		e.head.Name == b.prefix+"/bin/ld.lld" && e.head.Linkname == "lld"
}

func (b *bundle) result(e *entry) (tar.Header, *entry, error) {
	h := e.head
	if e.from != "" {
		target, err := b.resolve(e.from)
		return h, target, err
	}
	if h.Typeflag != tar.TypeSymlink && h.Typeflag != tar.TypeLink {
		return h, e, nil
	}
	target, err := b.resolve(h.Name)
	if err != nil {
		return h, nil, err
	}
	if !b.allowed(e) {
		h.Typeflag, h.Linkname = tar.TypeReg, ""
		h.Size, h.Mode = target.head.Size, target.head.Mode
	}
	return h, target, nil
}

func writeTar(b *bundle, data io.ReaderAt, out, report io.Writer) error {
	tw := tar.NewWriter(out)
	files, links, made, licenses := 0, 0, 0, 0
	var total int64
	for _, name := range b.names {
		e := b.items[name]
		h, target, err := b.result(e)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&h); err != nil {
			return err
		}
		if h.Typeflag == tar.TypeReg {
			files++
			total += h.Size
			if e != target {
				made++
			}
			hash := sha256.New()
			n, err := io.Copy(io.MultiWriter(tw, hash), io.NewSectionReader(data, target.off, h.Size))
			if err != nil || n != h.Size || fmt.Sprintf("%x", hash.Sum(nil)) != fmt.Sprintf("%x", target.sum) {
				return fmt.Errorf("staged data mismatch: %s bytes=%d error=%v", name, n, err)
			}
			upper := strings.ToUpper(name)
			if strings.Contains(upper, "LICENSE") || strings.Contains(upper, "COPYING") || strings.Contains(upper, "NOTICE") || strings.HasSuffix(upper, "/COPYRIGHT") {
				licenses++
			}
		} else if h.Typeflag == tar.TypeSymlink {
			links++
		}
		if err := writeLog(report, "%c\t%04o\t%d\t%x\t%s\t%s\n", h.Typeflag, h.Mode, h.Size, target.sum, name, target.head.Name); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return logf("regular=%d symlinks=%d converted=%d regular_bytes=%d added_bytes=%d license_files=%d\n", files, links, made, total, total-b.bytes, licenses)
}

func plain(info os.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || st.Nlink != 1 {
		return errors.New("file must be regular with one link")
	}
	return nil
}

func safePath(name string) error {
	abs, err := filepath.Abs(name)
	if err != nil {
		return err
	}
	for p := abs; ; p = filepath.Dir(p) {
		info, err := pathInfo(p)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("filesystem link is not permitted: %s", p)
		}
		if p == filepath.Dir(p) {
			return nil
		}
	}
}

func openInput(name string) (*os.File, error) {
	if err := safePath(name); err != nil {
		return nil, err
	}
	info, err := pathInfo(name)
	if err != nil {
		return nil, err
	}
	if err := plain(info); err != nil {
		return nil, fmt.Errorf("input %s: %w", name, err)
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	now, err := f.Stat()
	if err != nil || plain(now) != nil || !os.SameFile(info, now) {
		f.Close()
		return nil, errors.New("input identity changed")
	}
	return f, nil
}

func readArc(name, prefix string, data io.Writer) (*bundle, error) {
	f, err := openInput(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	hash := sha256.New()
	xr, err := xz.NewReader(io.TeeReader(f, hash))
	if err != nil {
		return nil, err
	}
	b, err := scan(xr, prefix, data)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(io.Discard, xr); err != nil {
		return nil, err
	}
	b.info, err = f.Stat()
	if err != nil || plain(b.info) != nil {
		return nil, errors.New("input identity changed while reading")
	}
	if err := logf("input=%s sha256=%x regular_bytes=%d entries=%d\n", name, hash.Sum(nil), b.bytes, len(b.names)); err != nil {
		return nil, err
	}
	return b, nil
}

func freeSpace(dir string, need uint64) error {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return err
	}
	free := uint64(st.Bavail) * uint64(st.Bsize)
	if free < (50<<30)+need {
		return fmt.Errorf("free bytes %d cannot preserve 50 GiB with %d new bytes", free, need)
	}
	return nil
}

type stageFile struct {
	f    *os.File
	dir  string
	left int64
}

func (s *stageFile) Write(p []byte) (int, error) {
	if s.left < int64(len(p)) {
		if err := freeSpace(s.dir, 64<<20); err != nil {
			return 0, err
		}
		s.left = 64 << 20
	}
	n, err := s.f.Write(p)
	s.left -= int64(n)
	return n, err
}

func newFile(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
}

func pack(input, prefix, dir string, copies ...string) error {
	parent := filepath.Dir(dir)
	if err := safePath(parent); err != nil {
		return err
	}
	if err := freeSpace(parent, 64<<20); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return err
	}
	data, err := newFile(filepath.Join(dir, "content.bin"))
	if err != nil {
		return err
	}
	defer data.Close()
	b, err := readArc(input, prefix, &stageFile{f: data, dir: dir})
	if err != nil {
		return err
	}
	if err := b.addCopies(copies); err != nil {
		return err
	}
	if err := data.Sync(); err != nil {
		return err
	}
	var size uint64
	for _, name := range b.names {
		h, _, err := b.result(b.items[name])
		if err != nil {
			return err
		}
		size += uint64(h.Size) + 4096
	}
	if err := freeSpace(dir, size); err != nil {
		return err
	}
	out, err := newFile(filepath.Join(dir, prefix+"-cdh.tar"))
	if err != nil {
		return err
	}
	defer out.Close()
	report, err := newFile(filepath.Join(dir, "entries.tsv"))
	if err != nil {
		return err
	}
	defer report.Close()
	hash := sha256.New()
	if err := writeTar(b, data, io.MultiWriter(out, hash), report); err != nil {
		return err
	}
	var seen []os.FileInfo
	for _, f := range []*os.File{data, out, report} {
		if err := f.Sync(); err != nil {
			return err
		}
		info, err := f.Stat()
		if err != nil || plain(info) != nil {
			return fmt.Errorf("output identity check failed: %s", f.Name())
		}
		for _, prior := range seen {
			if os.SameFile(prior, info) {
				return errors.New("outputs share a file identity")
			}
		}
		seen = append(seen, info)
	}
	return logf("tar=%s sha256=%x independent=true\n", out.Name(), hash.Sum(nil))
}

func verify(input, output, prefix string, copies ...string) error {
	before, err := readArc(input, prefix, nil)
	if err != nil {
		return err
	}
	if err := before.addCopies(copies); err != nil {
		return err
	}
	after, err := readArc(output, prefix, nil)
	if err != nil {
		return err
	}
	if os.SameFile(before.info, after.info) {
		return errors.New("input and output share a file identity")
	}
	if len(before.items) != len(after.items) {
		return errors.New("archive entry count changed")
	}
	files, links := 0, 0
	for _, name := range before.names {
		h, target, err := before.result(before.items[name])
		if err != nil {
			return err
		}
		got, ok := after.items[name]
		if !ok || got.head.Typeflag != h.Typeflag || got.head.Mode != h.Mode || got.head.Size != h.Size || got.head.Linkname != h.Linkname {
			return fmt.Errorf("archive metadata differs: %s", name)
		}
		if h.Typeflag == tar.TypeReg {
			files++
			if got.sum != target.sum {
				return fmt.Errorf("archive target bytes differ: %s", name)
			}
		} else if h.Typeflag == tar.TypeSymlink {
			links++
		}
	}
	return logf("verified_regular=%d verified_symlinks=%d all_paths_modes_bytes_and_licenses_preserved=true\n", files, links)
}

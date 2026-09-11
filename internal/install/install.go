// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/ulikunitz/xz"
)

// Run installs every archive and publishes the complete directory atomically.
func Run(ctx context.Context, repo, dest string, spec catalog.Spec) error {
	if err := catalog.Valid(spec); err != nil {
		return err
	}
	if !spec.Ready {
		return errors.New("toolchain payload is not published")
	}
	if err := checkHost(spec.Host); err != nil {
		return err
	}
	repo, err := filepath.Abs(repo)
	if err != nil {
		return fmt.Errorf("resolve repository: %w", err)
	}
	repo, err = cleanRoot(repo)
	if err != nil {
		return err
	}
	dest, err = cleanDest(dest)
	if err != nil {
		return err
	}
	holds, err := holdsRepo(dest, repo)
	if err != nil {
		return err
	}
	if holds {
		return fmt.Errorf("destination contains repository: %s", dest)
	}
	parent := filepath.Dir(dest)
	work, err := os.MkdirTemp(parent, ".cdh-toolchain-")
	if err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}
	defer os.RemoveAll(work)
	if err := os.Chmod(work, 0o700); err != nil {
		return fmt.Errorf("secure install directory: %w", err)
	}
	next := filepath.Join(work, "next")
	if err := os.Mkdir(next, 0o755); err != nil {
		return fmt.Errorf("create new toolchain root: %w", err)
	}
	if err := os.Chmod(next, 0o755); err != nil {
		return fmt.Errorf("set new toolchain root mode: %w", err)
	}
	for _, archive := range spec.Archives {
		if err := unpack(ctx, repo, next, archive); err != nil {
			return fmt.Errorf("install %s: %w", archive.Name, err)
		}
	}
	fence := filepath.Join(next, "go.mod")
	if err := os.WriteFile(fence, []byte("module cdh.invalid/toolchains\n\ngo 1.26.0\n"), 0o644); err != nil {
		return fmt.Errorf("write module boundary: %w", err)
	}
	if err := os.Chmod(fence, 0o644); err != nil {
		return fmt.Errorf("set module boundary mode: %w", err)
	}
	if err := os.Chmod(next, 0o755); err != nil {
		return fmt.Errorf("set new toolchain root mode: %w", err)
	}
	if err := checkAt(ctx, next, spec); err != nil {
		return err
	}
	if info, statErr := os.Lstat(dest); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination is not a regular directory: %s", dest)
		}
		if err := swapDirs(next, dest); err != nil {
			return fmt.Errorf("replace toolchains: %w", err)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect destination: %w", statErr)
	} else {
		if err := os.Rename(next, dest); err != nil {
			return fmt.Errorf("publish toolchains: %w", err)
		}
	}
	return nil
}

// Check runs the branch identity and installed tool probes.
func Check(ctx context.Context, root string, spec catalog.Spec) error {
	if err := catalog.Valid(spec); err != nil {
		return err
	}
	if !spec.Ready {
		return errors.New("toolchain payload is not published")
	}
	if err := checkHost(spec.Host); err != nil {
		return err
	}
	return checkAt(ctx, root, spec)
}

func checkAt(ctx context.Context, root string, spec catalog.Spec) error {
	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect toolchain root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("toolchain root is not a regular directory: %s", root)
	}
	for _, archive := range spec.Archives {
		archiveRoot := filepath.Join(root, archive.Dir)
		info, err := os.Lstat(archiveRoot)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("toolchain directory is invalid: %s", archiveRoot)
		}
		for _, probe := range archive.Probes {
			program, err := safeExec(archiveRoot, probe.Path)
			if err != nil {
				return fmt.Errorf("invalid probe for %s: %w", archive.Name, err)
			}
			if err := checkMachine(program, probe.Machine); err != nil {
				return fmt.Errorf("probe machine for %s: %w", archive.Name, err)
			}
			cmd := exec.CommandContext(ctx, program, probe.Args...)
			cmd.Env = mergeEnv(os.Environ(), probe.Env)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("probe %s: %w: %s", archive.Name, err, strings.TrimSpace(string(output)))
			}
			for _, want := range probe.Want {
				if !strings.Contains(string(output), want) {
					return fmt.Errorf("probe %s output misses %q", archive.Name, want)
				}
			}
		}
	}
	return nil
}

func mergeEnv(base, extra []string) []string {
	keys := make(map[string]struct{}, len(extra))
	for _, setting := range extra {
		name, _, _ := strings.Cut(setting, "=")
		keys[name] = struct{}{}
	}
	result := make([]string, 0, len(base)+len(extra))
	for _, setting := range base {
		name, _, _ := strings.Cut(setting, "=")
		if _, replaced := keys[name]; !replaced {
			result = append(result, setting)
		}
	}
	return append(result, extra...)
}

func unpack(ctx context.Context, repo, root string, archive catalog.Archive) error {
	stream, err := openParts(repo, archive.Parts)
	if err != nil {
		return err
	}
	defer stream.Close()
	var reader io.Reader
	var closeFn func() error
	switch archive.Format {
	case catalog.TarGzip:
		gz, err := gzip.NewReader(stream)
		if err != nil {
			return fmt.Errorf("open gzip stream: %w", err)
		}
		reader = gz
		closeFn = gz.Close
	case catalog.TarXZ:
		xzReader, err := xz.NewReader(stream)
		if err != nil {
			return fmt.Errorf("open xz stream: %w", err)
		}
		reader = xzReader
		closeFn = func() error { return nil }
	default:
		return fmt.Errorf("unsupported archive format: %d", archive.Format)
	}
	target := filepath.Join(root, archive.Dir)
	if err := os.Mkdir(target, 0o755); err != nil {
		return fmt.Errorf("create archive root: %w", err)
	}
	if err := os.Chmod(target, 0o755); err != nil {
		return fmt.Errorf("set archive root mode: %w", err)
	}
	if err := untar(ctx, reader, target, archive.Prefix); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("finish archive stream: %w", err)
	}
	if err := fixModes(target); err != nil {
		return err
	}
	if err := closeFn(); err != nil {
		return fmt.Errorf("close archive stream: %w", err)
	}
	if err := stream.finish(archive.SHA256); err != nil {
		return err
	}
	return nil
}

func untar(ctx context.Context, reader io.Reader, root, prefix string) error {
	tarReader := tar.NewReader(reader)
	seen := make(map[string]struct{})
	nodes := make(map[fileID]string)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		rel, skip, err := stripName(header.Name, prefix)
		if err != nil {
			return err
		}
		if skip {
			continue
		}
		if _, exists := seen[rel]; exists {
			return fmt.Errorf("duplicate archive path: %s", rel)
		}
		seen[rel] = struct{}{}
		dest, err := safeJoin(root, rel)
		if err != nil {
			return err
		}
		if err := makeParent(root, rel); err != nil {
			return err
		}
		if err := trackParents(root, rel, nodes); err != nil {
			return err
		}
		if _, err := os.Lstat(dest); err == nil {
			if err := checkNode(nodes, rel, dest); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect archive path: %w", err)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(dest, 0o755); err != nil {
				if !errors.Is(err, os.ErrExist) {
					return fmt.Errorf("create archive directory: %w", err)
				}
				info, statErr := os.Lstat(dest)
				if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					return fmt.Errorf("archive directory path is not a regular directory: %s", rel)
				}
			}
			if err := os.Chmod(dest, 0o755); err != nil {
				return fmt.Errorf("set archive directory mode: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			mode := os.FileMode(0o644)
			if header.Mode&0o111 != 0 {
				mode = 0o755
			}
			file, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return fmt.Errorf("create archive file: %w", err)
			}
			written, copyErr := io.Copy(file, tarReader)
			closeErr := file.Close()
			if copyErr != nil {
				return fmt.Errorf("write archive file: %w", copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close archive file: %w", closeErr)
			}
			if err := os.Chmod(dest, mode); err != nil {
				return fmt.Errorf("set archive file mode: %w", err)
			}
			if written != header.Size {
				return fmt.Errorf("archive file size mismatch: %s", rel)
			}
		case tar.TypeSymlink:
			if path.IsAbs(header.Linkname) {
				return fmt.Errorf("absolute archive link: %s", rel)
			}
			linkRel := path.Clean(path.Join(path.Dir(rel), header.Linkname))
			if linkRel == ".." || strings.HasPrefix(linkRel, "../") {
				return fmt.Errorf("archive link escapes root: %s", rel)
			}
			if err := os.Symlink(header.Linkname, dest); err != nil {
				return fmt.Errorf("create archive link: %w", err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type %d: %s", header.Typeflag, rel)
		}
		if err := checkNode(nodes, rel, dest); err != nil {
			return err
		}
	}
}

type fileID struct {
	device uint64
	inode  uint64
}

func nodeID(info os.FileInfo) (fileID, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileID{}, errors.New("file identity is unavailable")
	}
	return fileID{device: uint64(stat.Dev), inode: uint64(stat.Ino)}, nil
}

func checkNode(nodes map[fileID]string, rel, name string) error {
	info, err := os.Lstat(name)
	if err != nil {
		return fmt.Errorf("inspect archive node: %w", err)
	}
	id, err := nodeID(info)
	if err != nil {
		return err
	}
	if prior, exists := nodes[id]; exists && prior != rel {
		return fmt.Errorf("archive paths share one filesystem node: %s and %s", prior, rel)
	}
	nodes[id] = rel
	return nil
}

func trackParents(root, rel string, nodes map[fileID]string) error {
	parent := path.Dir(rel)
	if parent == "." {
		return nil
	}
	current := root
	logical := ""
	for _, part := range strings.Split(parent, "/") {
		current = filepath.Join(current, part)
		logical = path.Join(logical, part)
		if err := checkNode(nodes, logical, current); err != nil {
			return err
		}
	}
	return nil
}

func stripName(name, prefix string) (string, bool, error) {
	if path.IsAbs(name) {
		return "", false, fmt.Errorf("absolute archive path: %s", name)
	}
	clean := path.Clean(strings.TrimPrefix(name, "./"))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false, fmt.Errorf("archive path escapes root: %s", name)
	}
	if clean == prefix {
		return "", true, nil
	}
	if !strings.HasPrefix(clean, prefix+"/") {
		return "", true, nil
	}
	rel := strings.TrimPrefix(clean, prefix+"/")
	if rel == "." || rel == "" {
		return "", true, nil
	}
	return rel, false, nil
}

func safeJoin(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative: %s", rel)
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root: %s", rel)
	}
	return filepath.Join(root, clean), nil
}

func safeExec(root, rel string) (string, error) {
	program, err := safeJoin(root, rel)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(program)
	if err != nil {
		return "", fmt.Errorf("resolve probe program: %w", err)
	}
	inside, err := inTreeReal(root, real)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", fmt.Errorf("probe program escapes toolchain: %s", rel)
	}
	return real, nil
}

func makeParent(root, rel string) error {
	parent := path.Dir(rel)
	if parent == "." {
		return nil
	}
	current := root
	for _, name := range strings.Split(parent, "/") {
		current = filepath.Join(current, name)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o755); err != nil {
				return fmt.Errorf("create archive parent: %w", err)
			}
			if err := os.Chmod(current, 0o755); err != nil {
				return fmt.Errorf("set archive parent mode: %w", err)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect archive parent: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive parent is not a regular directory: %s", current)
		}
	}
	return nil
}

func cleanRoot(root string) (string, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("inspect repository root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("repository root is not a regular directory: %s", root)
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	if real != filepath.Clean(root) {
		return "", fmt.Errorf("repository root contains a linked path: %s", root)
	}
	return real, nil
}

func cleanDest(dest string) (string, error) {
	abs, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("resolve destination: %w", err)
	}
	if filepath.Dir(abs) == abs {
		return "", fmt.Errorf("destination cannot be a filesystem root: %s", abs)
	}
	parent := filepath.Dir(abs)
	if err := makeDirs(parent, 0o755); err != nil {
		return "", fmt.Errorf("create destination parent: %w", err)
	}
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", fmt.Errorf("resolve destination parent: %w", err)
	}
	return filepath.Join(realParent, filepath.Base(abs)), nil
}

func makeDirs(name string, mode os.FileMode) error {
	info, err := os.Lstat(name)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path is not a regular directory: %s", name)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(name)
	if parent != name {
		if err := makeDirs(parent, mode); err != nil {
			return err
		}
	}
	if err := os.Mkdir(name, mode); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		info, statErr := os.Lstat(name)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path changed while creating directory: %s", name)
		}
		return nil
	}
	return os.Chmod(name, mode)
}

func inTree(root, item string) bool {
	rel, err := filepath.Rel(root, item)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func inTreeReal(root, item string) (bool, error) {
	rootInfo, err := os.Stat(root)
	if err != nil {
		return false, fmt.Errorf("inspect containment root: %w", err)
	}
	current := item
	for {
		info, err := os.Stat(current)
		if err != nil {
			return false, fmt.Errorf("inspect contained path: %w", err)
		}
		if os.SameFile(rootInfo, info) {
			return true, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false, nil
		}
		current = parent
	}
}

func holdsRepo(dest, repo string) (bool, error) {
	if _, err := os.Lstat(dest); err == nil {
		return inTreeReal(dest, repo)
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect destination: %w", err)
	}
	return inTree(dest, repo), nil
}

func fixModes(root string) error {
	return filepath.WalkDir(root, func(pathName string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		mode := os.FileMode(0o644)
		if info.IsDir() || info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		if err := os.Chmod(pathName, mode); err != nil {
			return fmt.Errorf("set installed mode: %w", err)
		}
		return nil
	})
}

func openParts(repo string, parts []catalog.Part) (*partRead, error) {
	reader := &partRead{all: sha256.New()}
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		pathName, err := safeJoin(repo, part.Path)
		if err != nil {
			reader.Close()
			return nil, err
		}
		if _, exists := seen[pathName]; exists {
			reader.Close()
			return nil, fmt.Errorf("duplicate payload part: %s", part.Path)
		}
		seen[pathName] = struct{}{}
		real, err := filepath.EvalSymlinks(pathName)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("resolve payload part: %w", err)
		}
		if real != filepath.Clean(pathName) {
			reader.Close()
			return nil, fmt.Errorf("payload part contains a linked path: %s", part.Path)
		}
		info, err := os.Lstat(pathName)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("inspect payload part: %w", err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != part.Size {
			reader.Close()
			return nil, fmt.Errorf("invalid payload part: %s", part.Path)
		}
		file, err := os.Open(pathName)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("open payload part: %w", err)
		}
		openInfo, err := file.Stat()
		if err != nil || !openInfo.Mode().IsRegular() || openInfo.Size() != part.Size || !os.SameFile(info, openInfo) {
			file.Close()
			reader.Close()
			return nil, fmt.Errorf("payload part changed while opening: %s", part.Path)
		}
		reader.parts = append(reader.parts, partFile{file: file, want: part, sum: sha256.New()})
	}
	return reader, nil
}

func checkHost(want string) error {
	got := runtime.GOOS + "-" + runtime.GOARCH
	if got != want {
		return fmt.Errorf("host mismatch: got %s want %s", got, want)
	}
	return nil
}

func checkMachine(program string, want catalog.Machine) error {
	switch want {
	case catalog.MachARM64:
		return checkMach(program)
	case catalog.ELFARM64:
		return checkELF(program, elf.EM_AARCH64)
	case catalog.ELFAMD64:
		return checkELF(program, elf.EM_X86_64)
	case catalog.Script:
		return checkScript(program)
	default:
		return fmt.Errorf("unsupported probe machine: %d", want)
	}
}

func checkMach(program string) error {
	fat, fatErr := macho.OpenFat(program)
	if fatErr == nil {
		defer fat.Close()
		return errors.New("fat Mach-O is not allowed")
	}
	thin, err := macho.Open(program)
	if err != nil {
		return fmt.Errorf("open Mach-O: %w", err)
	}
	defer thin.Close()
	if thin.Cpu != macho.CpuArm64 {
		return fmt.Errorf("Mach-O CPU is %s", thin.Cpu)
	}
	return nil
}

func checkELF(program string, want elf.Machine) error {
	file, err := elf.Open(program)
	if err != nil {
		return fmt.Errorf("open ELF: %w", err)
	}
	defer file.Close()
	if file.Machine != want {
		return fmt.Errorf("ELF machine is %s", file.Machine)
	}
	return nil
}

func checkScript(program string) error {
	file, err := os.Open(program)
	if err != nil {
		return err
	}
	defer file.Close()
	magic := make([]byte, 2)
	if _, err := io.ReadFull(file, magic); err != nil {
		return fmt.Errorf("read script header: %w", err)
	}
	if string(magic) != "#!" {
		return errors.New("script has no interpreter header")
	}
	return nil
}

type partRead struct {
	parts []partFile
	index int
	all   hash.Hash
}

type partFile struct {
	file *os.File
	want catalog.Part
	sum  hash.Hash
	size int64
}

func (reader *partRead) Read(buffer []byte) (int, error) {
	for {
		if reader.index >= len(reader.parts) {
			return 0, io.EOF
		}
		part := &reader.parts[reader.index]
		count, err := part.file.Read(buffer)
		if count > 0 {
			_, _ = part.sum.Write(buffer[:count])
			_, _ = reader.all.Write(buffer[:count])
			part.size += int64(count)
		}
		if errors.Is(err, io.EOF) {
			if part.size != part.want.Size || hex.EncodeToString(part.sum.Sum(nil)) != part.want.SHA256 {
				return count, fmt.Errorf("payload part checksum mismatch: %s", part.want.Path)
			}
			if closeErr := part.file.Close(); closeErr != nil {
				return count, closeErr
			}
			part.file = nil
			reader.index++
			if count != 0 {
				return count, nil
			}
			continue
		}
		return count, err
	}
}

func (reader *partRead) finish(want string) error {
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	got := hex.EncodeToString(reader.all.Sum(nil))
	if got != want {
		return fmt.Errorf("archive checksum mismatch: got %s", got)
	}
	return nil
}

func (reader *partRead) Close() error {
	var first error
	for index := reader.index; index < len(reader.parts); index++ {
		part := &reader.parts[index]
		if part.file == nil {
			continue
		}
		if err := part.file.Close(); err != nil && first == nil {
			first = err
		}
		part.file = nil
	}
	return first
}

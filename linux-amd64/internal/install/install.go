// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/ulikunitz/xz"
	"golang.org/x/sys/unix"
)

const fenceData = "module cdh.invalid/toolchains\n\ngo 1.26.0\n"

// Run installs missing archives and preserves verified components.
func Run(ctx context.Context, repo, dest string, spec catalog.Spec) error {
	return RunTo(ctx, repo, dest, spec, io.Discard)
}

// RunTo installs missing archives and reports component progress.
func RunTo(ctx context.Context, repo, dest string, spec catalog.Spec, output io.Writer) error {
	if err := catalog.Valid(spec); err != nil {
		return err
	}
	if !spec.Ready {
		return errors.New("toolchain payload is not published")
	}
	if err := checkHost(spec.Host); err != nil {
		return err
	}
	progress, err := newProgress(output)
	if err != nil {
		return err
	}
	repo, err = filepath.Abs(repo)
	if err != nil {
		return fmt.Errorf("resolve repository: %w", err)
	}
	repo, err = cleanRoot(repo)
	if err != nil {
		return err
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	if filepath.Dir(dest) == dest {
		return fmt.Errorf("destination cannot be a filesystem root: %s", dest)
	}
	holds, err := holdsRepo(dest, repo)
	if err != nil {
		return err
	}
	if holds {
		return fmt.Errorf("destination contains repository: %s", dest)
	}
	root, err := openTree(dest, true)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := checkFence(ctx, root); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	total := len(spec.Archives)
	for offset, archive := range spec.Archives {
		item := progItem{
			component: archive.Name,
			index:     offset + 1,
			total:     total,
			remaining: total - offset - 1,
		}
		started := time.Now()
		progress.line(item, "checking", started)
		if err := progress.result(); err != nil {
			return err
		}
		if err := matchRoot(root, dest); err != nil {
			return err
		}
		renamed, renameErr := renameSDK(ctx, root, archive)
		if renameErr != nil {
			return fmt.Errorf("rename SDK: %w", renameErr)
		}
		if renamed {
			if _, err := fmt.Fprintln(output, "renamed verified SDK to "+filepath.Join(dest, archive.Dir)); err != nil {
				return err
			}
		}
		component, openErr := openDir(root, archive.Dir)
		if openErr == nil {
			checkErr := checkComp(ctx, component, archive)
			migrated := false
			if checkErr != nil {
				var fixErr error
				migrated, fixErr = migrateJDK(ctx, repo, root, component, archive)
				if fixErr != nil {
					checkErr = errors.Join(checkErr, fixErr)
				} else if migrated {
					checkErr = nil
				}
			}
			closeErr := component.Close()
			if checkErr != nil {
				return fmt.Errorf("existing component %s is invalid: %w", archive.Name, checkErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close component %s: %w", archive.Name, closeErr)
			}
			if err := matchRoot(root, dest); err != nil {
				return err
			}
			state := "skipped"
			if migrated {
				state = "migrated"
				if _, err := fmt.Fprintln(output, "preserved legacy JDK at "+filepath.Join(dest, "jdk21-original")); err != nil {
					return fmt.Errorf("write JDK backup path: %w", err)
				}
			}
			progress.line(item, state, started)
			if err := progress.result(); err != nil {
				return err
			}
			continue
		} else if !errors.Is(openErr, os.ErrNotExist) {
			return fmt.Errorf("inspect component %s: %w", archive.Name, openErr)
		}
		stop := progress.startBeat(item, started)
		if err := matchRoot(root, dest); err != nil {
			stop()
			return err
		}
		installErr := installArc(ctx, repo, root, archive)
		stop()
		if err := progress.result(); err != nil {
			return err
		}
		if installErr != nil {
			return fmt.Errorf("install %s: %w", archive.Name, installErr)
		}
		progress.line(item, "installed", started)
		if err := progress.result(); err != nil {
			return fmt.Errorf("install %s: %w", archive.Name, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := matchRoot(root, dest); err != nil {
		return err
	}
	if err := putFence(ctx, root); err != nil {
		return err
	}
	if err := matchRoot(root, dest); err != nil {
		return err
	}
	if err := checkAt(ctx, root, spec); err != nil {
		return err
	}
	return matchRoot(root, dest)
}

type progItem struct {
	component string
	index     int
	total     int
	remaining int
}

type progressOut struct {
	output   io.Writer
	interval time.Duration
	mu       sync.Mutex
	err      error
}

func newProgress(output io.Writer) (*progressOut, error) {
	if output == nil {
		return nil, errors.New("progress output is nil")
	}
	interval := 5 * time.Second
	if value := os.Getenv("CDH_TOOLCHAIN_PROGRESS_INTERVAL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("invalid CDH_TOOLCHAIN_PROGRESS_INTERVAL: %q", value)
		}
		interval = parsed
	}
	return &progressOut{output: output, interval: interval}, nil
}

func (progress *progressOut) line(item progItem, state string, started time.Time) {
	progress.mu.Lock()
	defer progress.mu.Unlock()
	if progress.err != nil {
		return
	}
	elapsed := int64(time.Since(started) / time.Second)
	_, progress.err = fmt.Fprintf(progress.output,
		"toolchain_progress component=%q index=%d total=%d remaining=%d elapsed_seconds=%d state=%s\n",
		item.component, item.index, item.total, item.remaining, elapsed, state)
}

func (progress *progressOut) startBeat(item progItem, started time.Time) func() {
	progress.line(item, "installing", started)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(progress.interval)
		defer ticker.Stop()
		defer close(done)
		for {
			select {
			case <-ticker.C:
				progress.line(item, "installing", started)
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

func (progress *progressOut) result() error {
	progress.mu.Lock()
	defer progress.mu.Unlock()
	if progress.err != nil {
		return fmt.Errorf("write install progress: %w", progress.err)
	}
	return nil
}

func openRoot(dest string) (*os.File, error) {
	pathInfo, err := os.Lstat(dest)
	if err != nil {
		return nil, fmt.Errorf("inspect toolchain root: %w", err)
	}
	root, err := os.Open(dest)
	if err != nil {
		return nil, fmt.Errorf("open toolchain root: %w", err)
	}
	openInfo, err := root.Stat()
	if err != nil || !openInfo.IsDir() || !os.SameFile(pathInfo, openInfo) {
		root.Close()
		return nil, fmt.Errorf("toolchain root changed while opening: %s", dest)
	}
	return root, nil
}

func matchRoot(root *os.File, dest string) error {
	current, err := openTree(dest, false)
	if err != nil {
		return fmt.Errorf("reopen toolchain root: %w", err)
	}
	defer current.Close()
	pathInfo, err := current.Stat()
	if err != nil {
		return fmt.Errorf("inspect current toolchain root: %w", err)
	}
	openInfo, err := root.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened toolchain root: %w", err)
	}
	if !os.SameFile(pathInfo, openInfo) {
		return fmt.Errorf("toolchain root changed during install: %s", dest)
	}
	return nil
}

func installArc(ctx context.Context, repo string, root *os.File, archive catalog.Archive) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	workRoot, work, err := makeTemp(root, ".cdh-toolchain-")
	if err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}
	defer workRoot.Close()
	defer clearTemp(root, workRoot, work)
	if err := unpack(ctx, repo, workRoot, archive); err != nil {
		return err
	}
	if !sameChild(root, workRoot, work) {
		return errors.New("install directory changed during unpack")
	}
	candidate, err := openDir(workRoot, archive.Dir)
	if err != nil {
		return fmt.Errorf("open install candidate: %w", err)
	}
	defer candidate.Close()
	if err := checkComp(ctx, candidate, archive); err != nil {
		return err
	}
	if !sameChild(workRoot, candidate, archive.Dir) {
		return errors.New("install candidate changed after check")
	}
	if !sameChild(root, workRoot, work) {
		return errors.New("install directory changed before publish")
	}
	_ = ctx.Done()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !sameChild(workRoot, candidate, archive.Dir) {
		return errors.New("install candidate changed before publish")
	}
	if err := moveNew(workRoot, archive.Dir, root); err != nil {
		return fmt.Errorf("publish component without replacement: %w", err)
	}
	if !sameChild(root, candidate, archive.Dir) {
		return errors.New("published component identity changed")
	}
	return nil
}

// migrateJDK先验证登记候选，再原子换位，保留没有裁剪标记的旧JDK。
func migrateJDK(ctx context.Context, repo string, root, prior *os.File, archive catalog.Archive) (bool, error) {
	if archive.Name != "JDK" || archive.Dir != "jdk21" {
		return false, nil
	}
	var marker unix.Stat_t
	if err := unix.Fstatat(int(prior.Fd()), "cdh-jdk-trim.txt", &marker, unix.AT_SYMLINK_NOFOLLOW); !errors.Is(err, unix.ENOENT) {
		return false, nil
	}
	saved := archive
	saved.Dir = "jdk21-original"
	if err := installArc(ctx, repo, root, saved); err != nil {
		return false, fmt.Errorf("prepare JDK migration without replacement: %w", err)
	}
	fresh, err := openDir(root, saved.Dir)
	if err != nil {
		return false, err
	}
	defer fresh.Close()
	swapped := false
	defer func() {
		if !swapped && sameChild(root, fresh, saved.Dir) {
			clearTemp(root, fresh, saved.Dir)
		}
	}()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !sameChild(root, prior, archive.Dir) || !sameChild(root, fresh, saved.Dir) {
		return false, errors.New("JDK directory changed before migration")
	}
	if err := swapAt(root, archive.Dir, saved.Dir); err != nil {
		return false, fmt.Errorf("exchange verified JDK with legacy directory: %w", err)
	}
	swapped = true
	if !sameChild(root, fresh, archive.Dir) || !sameChild(root, prior, saved.Dir) {
		return false, errors.New("JDK directory identity changed during migration")
	}
	return true, nil
}

func checkFence(ctx context.Context, root *os.File) error {
	file, err := openRel(root, "go.mod", false)
	if err != nil {
		return fmt.Errorf("inspect module boundary: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("inspect module boundary: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o644 {
		file.Close()
		return errors.New("module boundary is invalid")
	}
	data, readErr := io.ReadAll(io.LimitReader(&ctxRead{ctx: ctx, reader: file}, int64(len(fenceData)+1)))
	closeErr := file.Close()
	if readErr != nil {
		return fmt.Errorf("read module boundary: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close module boundary: %w", closeErr)
	}
	if string(data) != fenceData {
		return errors.New("module boundary content is invalid")
	}
	return nil
}

func putFence(ctx context.Context, root *os.File) error {
	if err := checkFence(ctx, root); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	workRoot, work, err := makeTemp(root, ".cdh-toolchain-fence-")
	if err != nil {
		return fmt.Errorf("create module boundary directory: %w", err)
	}
	defer workRoot.Close()
	defer clearTemp(root, workRoot, work)
	file, err := newFile(workRoot, "go.mod", 0o644)
	if err != nil {
		return fmt.Errorf("write module boundary: %w", err)
	}
	_, writeErr := file.WriteString(fenceData)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write module boundary: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close module boundary: %w", closeErr)
	}
	if err := checkFence(ctx, workRoot); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := moveNew(workRoot, "go.mod", root); err != nil {
		return fmt.Errorf("publish module boundary without replacement: %w", err)
	}
	return nil
}

func clearTemp(parent, root *os.File, name string) {
	if clearDir(root) != nil {
		return
	}
	if sameChild(parent, root, name) {
		_ = unix.Unlinkat(int(parent.Fd()), name, unix.AT_REMOVEDIR)
	}
}

func sameChild(parent, child *os.File, name string) bool {
	var pathStat unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &pathStat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return false
	}
	info, err := child.Stat()
	if err != nil {
		return false
	}
	openStat, ok := info.Sys().(*syscall.Stat_t)
	return ok && uint64(pathStat.Dev) == uint64(openStat.Dev) && uint64(pathStat.Ino) == uint64(openStat.Ino)
}

func makeTemp(root *os.File, prefix string) (*os.File, string, error) {
	for attempt := 0; attempt < 100; attempt++ {
		var token [8]byte
		if _, err := rand.Read(token[:]); err != nil {
			return nil, "", err
		}
		name := prefix + hex.EncodeToString(token[:])
		if err := unix.Mkdirat(int(root.Fd()), name, 0o700); err != nil {
			if errors.Is(err, unix.EEXIST) {
				continue
			}
			return nil, "", err
		}
		work, err := dirAt(root, name, false, 0o700)
		if err != nil {
			_ = unix.Unlinkat(int(root.Fd()), name, unix.AT_REMOVEDIR)
			return nil, "", err
		}
		if err := work.Chmod(0o700); err != nil {
			work.Close()
			_ = unix.Unlinkat(int(root.Fd()), name, unix.AT_REMOVEDIR)
			return nil, "", err
		}
		return work, name, nil
	}
	return nil, "", errors.New("cannot allocate unique install directory")
}

func clearDir(root *os.File) error {
	entries, err := root.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			fd, err := unix.Openat(int(root.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			if err != nil {
				return err
			}
			child := os.NewFile(uintptr(fd), name)
			clearErr := clearDir(child)
			closeErr := child.Close()
			if clearErr != nil {
				return clearErr
			}
			if closeErr != nil {
				return closeErr
			}
			if err := unix.Unlinkat(int(root.Fd()), name, unix.AT_REMOVEDIR); err != nil {
				return err
			}
			continue
		}
		if err := unix.Unlinkat(int(root.Fd()), name, 0); err != nil {
			return err
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
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve toolchain root: %w", err)
	}
	opened, err := openTree(abs, false)
	if err != nil {
		return err
	}
	defer opened.Close()
	if err := matchRoot(opened, abs); err != nil {
		return err
	}
	if err := checkAt(ctx, opened, spec); err != nil {
		return err
	}
	return matchRoot(opened, abs)
}

func checkAt(ctx context.Context, root *os.File, spec catalog.Spec) error {
	info, err := root.Stat()
	if err != nil {
		return fmt.Errorf("inspect toolchain root: %w", err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o755 || !safeOwner(info) {
		return errors.New("toolchain root is not a regular directory")
	}
	if err := checkFence(ctx, root); err != nil {
		return err
	}
	for _, archive := range spec.Archives {
		if err := checkArc(ctx, root, archive); err != nil {
			return err
		}
	}
	return nil
}

func checkArc(ctx context.Context, root *os.File, archive catalog.Archive) error {
	component, err := openDir(root, archive.Dir)
	if err != nil {
		return fmt.Errorf("open component %s: %w", archive.Name, err)
	}
	defer component.Close()
	return checkComp(ctx, component, archive)
}

func checkComp(ctx context.Context, root *os.File, archive catalog.Archive) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := root.Stat()
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o755 || !safeOwner(info) {
		return fmt.Errorf("toolchain directory is invalid: %s", archive.Dir)
	}
	for _, check := range archive.Files {
		file, err := openRel(root, check.Path, true)
		if err != nil {
			return fmt.Errorf("invalid identity file for %s: %w", archive.Name, err)
		}
		info, statErr := file.Stat()
		if statErr != nil || !info.Mode().IsRegular() || info.Size() != check.Size {
			file.Close()
			return fmt.Errorf("identity file is invalid for %s: %s", archive.Name, check.Path)
		}
		data, readErr := io.ReadAll(io.LimitReader(&ctxRead{ctx: ctx, reader: file}, check.Size+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || int64(len(data)) != check.Size {
			return fmt.Errorf("read identity file for %s: %s", archive.Name, check.Path)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != check.SHA256 {
			return fmt.Errorf("identity file checksum mismatch for %s: %s", archive.Name, check.Path)
		}
		for _, want := range check.Want {
			if !strings.Contains(string(data), want) {
				return fmt.Errorf("identity file for %s misses %q", archive.Name, want)
			}
		}
	}
	for _, probe := range archive.Probes {
		if err := runProbe(ctx, root, archive.Name, probe); err != nil {
			return err
		}
	}
	return nil
}

func runProbe(ctx context.Context, root *os.File, name string, probe catalog.Probe) error {
	program, err := openRel(root, probe.Path, true)
	if err != nil {
		return fmt.Errorf("invalid probe for %s: %w", name, err)
	}
	defer program.Close()
	programInfo, err := program.Stat()
	if err != nil || !programInfo.Mode().IsRegular() || programInfo.Mode().Perm() != 0o755 || !safeOwner(programInfo) {
		return fmt.Errorf("resolve probe for %s: opened probe is invalid", name)
	}
	programPath := fdAlias(program)
	if err := checkMachine(programPath, probe.Machine); err != nil {
		return fmt.Errorf("probe machine for %s: %w", name, err)
	}
	output, err := runAt(
		ctx,
		root,
		program,
		probe.Path,
		probe.Machine == catalog.Script,
		probe.Args,
		mergeEnv(os.Environ(), probe.Env),
	)
	if err != nil {
		return fmt.Errorf("probe %s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	for _, want := range probe.Want {
		if !strings.Contains(string(output), want) {
			return fmt.Errorf("probe %s output misses %q", name, want)
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

func unpack(ctx context.Context, repo string, root *os.File, archive catalog.Archive) error {
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
	reader = &ctxRead{ctx: ctx, reader: reader}
	target, err := makeDir(root, archive.Dir)
	if err != nil {
		return fmt.Errorf("create archive root: %w", err)
	}
	if err := untar(ctx, reader, target, archive.Prefix); err != nil {
		target.Close()
		return err
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("close archive root: %w", err)
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("finish archive stream: %w", err)
	}
	if err := closeFn(); err != nil {
		return fmt.Errorf("close archive stream: %w", err)
	}
	if err := stream.finish(ctx, archive.SHA256); err != nil {
		return err
	}
	return nil
}

func untar(ctx context.Context, reader io.Reader, root *os.File, prefix string) error {
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
		parent, _, err := openParent(root, rel, true)
		if err != nil {
			return err
		}
		if err := parent.Close(); err != nil {
			return fmt.Errorf("close archive parent: %w", err)
		}
		if _, exists := seen[rel]; exists {
			return fmt.Errorf("duplicate archive path: %s", rel)
		}
		seen[rel] = struct{}{}
		if err := trackParents(root, rel, nodes); err != nil {
			return err
		}
		if _, err := nodeAt(root, rel); err == nil {
			if err := checkNode(nodes, rel, root); err != nil {
				return err
			}
		} else if !errors.Is(err, unix.ENOENT) {
			return fmt.Errorf("inspect archive path: %w", err)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			dir, err := makeDir(root, rel)
			if err != nil {
				return fmt.Errorf("create archive directory: %w", err)
			}
			if err := dir.Close(); err != nil {
				return fmt.Errorf("close archive directory: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			mode := os.FileMode(0o644)
			if header.Mode&0o111 != 0 {
				mode = 0o755
			}
			file, err := newFile(root, rel, mode)
			if err != nil {
				return fmt.Errorf("create archive file: %w", err)
			}
			written, copyErr := io.Copy(file, &ctxRead{ctx: ctx, reader: tarReader})
			chmodErr := file.Chmod(mode)
			closeErr := file.Close()
			if copyErr != nil {
				return fmt.Errorf("write archive file: %w", copyErr)
			}
			if chmodErr != nil {
				return fmt.Errorf("set archive file mode: %w", chmodErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close archive file: %w", closeErr)
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
			if err := newLink(root, rel, header.Linkname); err != nil {
				return fmt.Errorf("create archive link: %w", err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type %d: %s", header.Typeflag, rel)
		}
		if err := checkNode(nodes, rel, root); err != nil {
			return err
		}
	}
}

type fileID struct {
	device uint64
	inode  uint64
}

func checkNode(nodes map[fileID]string, rel string, root *os.File) error {
	id, err := nodeAt(root, rel)
	if err != nil {
		return fmt.Errorf("inspect archive node: %w", err)
	}
	if prior, exists := nodes[id]; exists && prior != rel {
		return fmt.Errorf("archive paths share one filesystem node: %s and %s", prior, rel)
	}
	nodes[id] = rel
	return nil
}

func trackParents(root *os.File, rel string, nodes map[fileID]string) error {
	parent := path.Dir(rel)
	if parent == "." {
		return nil
	}
	logical := ""
	for _, part := range strings.Split(parent, "/") {
		logical = path.Join(logical, part)
		if err := checkNode(nodes, logical, root); err != nil {
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

func openRel(root *os.File, rel string, follow bool) (*os.File, error) {
	clean := path.Clean(filepath.ToSlash(rel))
	if path.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return nil, fmt.Errorf("invalid relative path: %s", rel)
	}
	pending := strings.Split(clean, "/")
	links := 0
	for {
		dir, err := dupRoot(root)
		if err != nil {
			return nil, err
		}
		logical := make([]string, 0, len(pending))
		restart := false
		for index, name := range pending {
			final := index == len(pending)-1
			flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW
			if final {
				flags |= unix.O_NONBLOCK
			} else {
				flags |= unix.O_DIRECTORY
			}
			fd, openErr := unix.Openat(int(dir.Fd()), name, flags, 0)
			if openErr == nil {
				fileName := name
				if final {
					fileName = path.Join(pending...)
				}
				next := os.NewFile(uintptr(fd), fileName)
				if final {
					info, statErr := next.Stat()
					if statErr != nil || !info.Mode().IsRegular() {
						next.Close()
						dir.Close()
						if statErr != nil {
							return nil, statErr
						}
						return nil, fmt.Errorf("relative path is not a regular file: %s", rel)
					}
					dir.Close()
					return next, nil
				}
				info, statErr := next.Stat()
				if statErr != nil || !info.IsDir() || info.Mode().Perm() != 0o755 || !safeOwner(info) {
					next.Close()
					dir.Close()
					return nil, fmt.Errorf("relative directory is invalid: %s", rel)
				}
				dir.Close()
				dir = next
				logical = append(logical, name)
				continue
			}
			target, linkErr := readLink(dir, name)
			if linkErr != nil {
				dir.Close()
				return nil, fmt.Errorf("open relative path %s: %w", rel, openErr)
			}
			if final && !follow {
				dir.Close()
				return nil, fmt.Errorf("linked final path is not allowed: %s", rel)
			}
			if path.IsAbs(target) {
				dir.Close()
				return nil, fmt.Errorf("relative path escapes root: %s", rel)
			}
			links++
			if links > 40 {
				dir.Close()
				return nil, fmt.Errorf("too many links in relative path: %s", rel)
			}
			parts := append([]string{}, logical...)
			parts = append(parts, target)
			parts = append(parts, pending[index+1:]...)
			nextRel := path.Clean(path.Join(parts...))
			if nextRel == "." || nextRel == ".." || strings.HasPrefix(nextRel, "../") {
				dir.Close()
				return nil, fmt.Errorf("relative path escapes root: %s", rel)
			}
			pending = strings.Split(nextRel, "/")
			dir.Close()
			restart = true
			break
		}
		if !restart {
			dir.Close()
			return nil, fmt.Errorf("open relative path failed: %s", rel)
		}
	}
}

func dupRoot(root *os.File) (*os.File, error) {
	fd, err := unix.Dup(int(root.Fd()))
	if err != nil {
		return nil, fmt.Errorf("duplicate directory handle: %w", err)
	}
	unix.CloseOnExec(fd)
	return os.NewFile(uintptr(fd), root.Name()), nil
}

func readLink(root *os.File, name string) (string, error) {
	buffer := make([]byte, 4096)
	count, err := unix.Readlinkat(int(root.Fd()), name, buffer)
	if err != nil {
		return "", err
	}
	if count == len(buffer) {
		return "", errors.New("linked path is too long")
	}
	return string(buffer[:count]), nil
}

func safeOwner(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	uid := uint32(os.Geteuid())
	return stat.Uid == 0 || stat.Uid == uid
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

func openTree(name string, create bool) (*os.File, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, fmt.Errorf("resolve destination tree: %w", err)
	}
	abs = filepath.Clean(abs)
	if filepath.Dir(abs) == abs {
		return nil, fmt.Errorf("destination cannot be a filesystem root: %s", abs)
	}
	root, err := os.Open(string(filepath.Separator))
	if err != nil {
		return nil, fmt.Errorf("open filesystem root: %w", err)
	}
	rootInfo, err := root.Stat()
	if err != nil || !rootInfo.IsDir() || !safeOwner(rootInfo) || rootInfo.Mode().Perm()&0o022 != 0 {
		root.Close()
		return nil, errors.New("filesystem root is not secure")
	}
	parts := strings.Split(strings.TrimPrefix(abs, string(filepath.Separator)), string(filepath.Separator))
	current := root
	for index, part := range parts {
		final := index == len(parts)-1
		parentInfo, err := current.Stat()
		if err != nil {
			current.Close()
			return nil, fmt.Errorf("inspect destination parent: %w", err)
		}
		next, err := dirAt(current, part, create, 0o755)
		if err != nil {
			current.Close()
			return nil, fmt.Errorf("open destination directory %s: %w", part, err)
		}
		info, err := next.Stat()
		if err != nil {
			next.Close()
			current.Close()
			return nil, fmt.Errorf("inspect destination directory %s: %w", part, err)
		}
		stat, statOK := info.Sys().(*syscall.Stat_t)
		parentStat, parentOK := parentInfo.Sys().(*syscall.Stat_t)
		if statOK && parentOK {
			mode, fix := mountFix(
				info.Mode(), stat.Uid, uint64(stat.Dev), uint64(parentStat.Dev), !final && create,
			)
			if fix {
				if err := next.Chmod(mode); err != nil {
					next.Close()
					current.Close()
					return nil, fmt.Errorf("set destination mount mode %s: %w", part, err)
				}
				info, err = next.Stat()
				if err != nil {
					next.Close()
					current.Close()
					return nil, fmt.Errorf("inspect destination mount %s: %w", part, err)
				}
			}
		}
		valid := info.Mode().Perm() == 0o755 && safeOwner(info)
		if !final {
			valid = safeParent(info, parentInfo)
		}
		if !valid {
			next.Close()
			current.Close()
			return nil, fmt.Errorf("destination directory is not secure: %s", part)
		}
		current.Close()
		current = next
	}
	return current, nil
}

func safeParent(info, parent os.FileInfo) bool {
	if !info.IsDir() || !safeOwner(info) {
		return false
	}
	stat, statOK := info.Sys().(*syscall.Stat_t)
	parentStat, parentOK := parent.Sys().(*syscall.Stat_t)
	if !statOK || !parentOK {
		return false
	}
	return parentMode(info.Mode(), stat.Uid, uint64(stat.Dev), uint64(parentStat.Dev))
}

func mountFix(mode os.FileMode, uid uint32, dev, parentDev uint64, create bool) (os.FileMode, bool) {
	perm := mode.Perm()
	if !create || perm&0o020 == 0 || perm&0o002 != 0 || uid != uint32(os.Geteuid()) ||
		dev == parentDev || mode&os.ModeSticky != 0 {
		return mode, false
	}
	chmodMode := mode & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	return chmodMode | os.ModeSticky, true
}

func parentMode(mode os.FileMode, uid uint32, dev, parentDev uint64) bool {
	perm := mode.Perm()
	if perm&0o002 != 0 {
		return false
	}
	if perm&0o020 == 0 {
		return true
	}
	return uid == uint32(os.Geteuid()) && dev != parentDev && mode&os.ModeSticky != 0
}

func dirAt(root *os.File, name string, create bool, mode os.FileMode) (*os.File, error) {
	return dirAtWith(root, name, create, mode, unix.Mkdirat)
}

func dirAtWith(
	root *os.File,
	name string,
	create bool,
	mode os.FileMode,
	mk func(int, string, uint32) error,
) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	fd, err := unix.Openat(int(root.Fd()), name, flags, 0)
	if err != nil && create && errors.Is(err, unix.ENOENT) {
		if err := mk(int(root.Fd()), name, uint32(mode.Perm())); err != nil {
			return nil, err
		}
		fd, err = unix.Openat(int(root.Fd()), name, flags, 0)
		if err == nil {
			err = unix.Fchmod(fd, uint32(mode.Perm()))
		}
	}
	if err != nil {
		return nil, err
	}
	dir := os.NewFile(uintptr(fd), name)
	info, statErr := dir.Stat()
	if statErr != nil || !info.IsDir() || !safeOwner(info) {
		dir.Close()
		if statErr != nil {
			return nil, statErr
		}
		return nil, fmt.Errorf("directory is not owned safely: %s", name)
	}
	return dir, nil
}

func openDir(root *os.File, rel string) (*os.File, error) {
	parent, name, err := openParent(root, rel, false)
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	return dirAt(parent, name, false, 0o755)
}

func openParent(root *os.File, rel string, create bool) (*os.File, string, error) {
	clean := path.Clean(filepath.ToSlash(rel))
	if path.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return nil, "", fmt.Errorf("invalid relative path: %s", rel)
	}
	parts := strings.Split(clean, "/")
	dir, err := dupRoot(root)
	if err != nil {
		return nil, "", err
	}
	for _, name := range parts[:len(parts)-1] {
		next, err := dirAt(dir, name, create, 0o755)
		if err != nil {
			dir.Close()
			return nil, "", err
		}
		info, err := next.Stat()
		if err != nil || info.Mode().Perm() != 0o755 {
			next.Close()
			dir.Close()
			return nil, "", fmt.Errorf("relative directory is invalid: %s", rel)
		}
		dir.Close()
		dir = next
	}
	return dir, parts[len(parts)-1], nil
}

func makeDir(root *os.File, rel string) (*os.File, error) {
	parent, name, err := openParent(root, rel, true)
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	if err := unix.Mkdirat(int(parent.Fd()), name, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
		return nil, err
	}
	dir, err := dirAt(parent, name, false, 0o755)
	if err != nil {
		return nil, err
	}
	if err := dir.Chmod(0o755); err != nil {
		dir.Close()
		return nil, err
	}
	return dir, nil
}

func newFile(root *os.File, rel string, mode os.FileMode) (*os.File, error) {
	parent, name, err := openParent(root, rel, true)
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	flags := unix.O_WRONLY | unix.O_CREAT | unix.O_EXCL | unix.O_NOFOLLOW | unix.O_CLOEXEC
	fd, err := unix.Openat(int(parent.Fd()), name, flags, uint32(mode.Perm()))
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func newLink(root *os.File, rel, target string) error {
	parent, name, err := openParent(root, rel, true)
	if err != nil {
		return err
	}
	defer parent.Close()
	return unix.Symlinkat(target, int(parent.Fd()), name)
}

func nodeAt(root *os.File, rel string) (fileID, error) {
	parent, name, err := openParent(root, rel, false)
	if err != nil {
		return fileID{}, err
	}
	defer parent.Close()
	var stat unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return fileID{}, err
	}
	return fileID{device: uint64(stat.Dev), inode: uint64(stat.Ino)}, nil
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

type ctxRead struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *ctxRead) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
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

func (reader *partRead) finish(ctx context.Context, want string) error {
	if _, err := io.Copy(io.Discard, &ctxRead{ctx: ctx, reader: reader}); err != nil {
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

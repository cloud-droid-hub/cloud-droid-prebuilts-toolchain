// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package rehome

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// Move核验登记组件，保持新安装及未知旧文件，清除已确认的旧副本。
func Move(ctx context.Context, old, root *os.File, oldPath, dest string,
	names []string, check func(*os.File, string) error, fence string, out io.Writer) error {
	if err := busy(ctx, oldPath); err != nil {
		return err
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		prior, err := openDir(old, name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("preserved legacy component %s: %w", filepath.Join(oldPath, name), err)
		}
		err = moveOne(ctx, old, root, prior, oldPath, dest, name, check)
		prior.Close()
		if err != nil {
			return fmt.Errorf("preserved legacy component %s: %w", filepath.Join(oldPath, name), err)
		}
		if _, err := fmt.Fprintln(out, "retired verified legacy component "+filepath.Join(oldPath, name)); err != nil {
			return err
		}
	}
	entries, err := readDir(old)
	if err != nil {
		return err
	}
	if len(entries) == 1 && entries[0].Name() == "go.mod" {
		file, err := openFile(old, "go.mod")
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(file, int64(len(fence)+1)))
		if err == nil && (string(data) != fence || !sameChild(old, file, "go.mod")) {
			err = errors.New("legacy module boundary changed")
		}
		if err == nil {
			err = unix.Unlinkat(int(old.Fd()), "go.mod", 0)
		}
		file.Close()
		if err != nil {
			return err
		}
		entries = nil
	}
	if len(entries) != 0 {
		for _, entry := range entries {
			if _, err := fmt.Fprintln(out, "preserved unselected legacy entry "+filepath.Join(oldPath, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if err := removeEmpty(oldPath, old); err != nil {
		return err
	}
	parentPath := filepath.Dir(oldPath)
	parent, err := openPath(parentPath)
	if err != nil {
		return err
	}
	defer parent.Close()
	left, err := readDir(parent)
	if err != nil || len(left) != 0 {
		return err
	}
	return removeEmpty(parentPath, parent)
}

func moveOne(ctx context.Context, old, root, prior *os.File, oldPath, dest, name string,
	check func(*os.File, string) error) error {
	if err := check(prior, name); err != nil {
		return err
	}
	fresh, err := openDir(root, name)
	if errors.Is(err, os.ErrNotExist) {
		if err := matchPair(old, root, prior, oldPath, dest, name); err != nil {
			return err
		}
		if err := renameDir(old, root, name); err != nil {
			return err
		}
		if !sameChild(root, prior, name) {
			return errors.New("migrated component identity changed")
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer fresh.Close()
	if err := check(fresh, name); err != nil {
		return err
	}
	if err := equalTree(ctx, prior, fresh); err != nil {
		return err
	}
	if err := matchPair(old, root, prior, oldPath, dest, name); err != nil {
		return err
	}
	if !sameChild(root, fresh, name) {
		return errors.New("new component identity changed")
	}
	stage, token, err := makeStage(root)
	if err != nil {
		return err
	}
	defer stage.Close()
	defer func() {
		if sameChild(root, stage, token) {
			_ = unix.Unlinkat(int(root.Fd()), token, unix.AT_REMOVEDIR)
		}
	}()
	if err := renameDir(old, stage, name); err != nil {
		return err
	}
	if !sameChild(stage, prior, name) || !sameChild(root, fresh, name) {
		err = errors.New("component identity changed during retirement")
	} else {
		err = equalTree(ctx, prior, fresh)
	}
	if err != nil {
		backErr := renameDir(stage, old, name)
		if backErr != nil {
			return errors.Join(err, fmt.Errorf("legacy data retained at %s: %w", filepath.Join(dest, token, name), backErr))
		}
		return err
	}
	if err := clearTree(prior); err != nil {
		return fmt.Errorf("legacy cleanup incomplete at %s: %w", filepath.Join(dest, token, name), err)
	}
	if !sameChild(stage, prior, name) {
		return errors.New("retired component identity changed")
	}
	return unix.Unlinkat(int(stage.Fd()), name, unix.AT_REMOVEDIR)
}

func matchPair(old, root, prior *os.File, oldPath, dest, name string) error {
	for file, path := range map[*os.File]string{old: oldPath, root: dest} {
		info, err := file.Stat()
		got, pathErr := os.Lstat(path)
		if err != nil || pathErr != nil || !os.SameFile(info, got) {
			return fmt.Errorf("toolchain root changed: %s", path)
		}
	}
	if !sameChild(old, prior, name) {
		return errors.New("legacy component identity changed")
	}
	return nil
}

func equalTree(ctx context.Context, left, right *os.File) error {
	items, err := readDir(left)
	if err != nil {
		return err
	}
	other, err := readDir(right)
	if err != nil {
		return err
	}
	if len(items) != len(other) {
		return errors.New("legacy and new components have different entries")
	}
	for index, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := item.Name()
		if name != other[index].Name() {
			return fmt.Errorf("different component entry: %s", name)
		}
		var a, b unix.Stat_t
		if err := unix.Fstatat(int(left.Fd()), name, &a, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return err
		}
		if err := unix.Fstatat(int(right.Fd()), name, &b, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return err
		}
		if a.Mode != b.Mode || a.Uid != b.Uid || a.Gid != b.Gid {
			return fmt.Errorf("different component metadata: %s", name)
		}
		var err error
		switch a.Mode & unix.S_IFMT {
		case unix.S_IFDIR:
			var x, y *os.File
			x, err = openDir(left, name)
			if err == nil {
				y, err = openDir(right, name)
				if err == nil {
					err = equalTree(ctx, x, y)
					y.Close()
				}
				x.Close()
			}
		case unix.S_IFREG:
			var x, y [32]byte
			x, err = fileSum(ctx, left, name)
			if err == nil {
				y, err = fileSum(ctx, right, name)
				if err == nil && (a.Size != b.Size || x != y) {
					err = errors.New("different file bytes")
				}
			}
		case unix.S_IFLNK:
			var x, y string
			x, err = linkText(left, name)
			if err == nil {
				y, err = linkText(right, name)
				if err == nil && x != y {
					err = errors.New("different link target")
				}
			}
		default:
			err = errors.New("unregistered file type")
		}
		if err != nil {
			return fmt.Errorf("different or changed component entry %s: %w", name, err)
		}
	}
	return nil
}

func fileSum(ctx context.Context, root *os.File, name string) ([32]byte, error) {
	file, err := openFile(root, name)
	if err != nil {
		return [32]byte{}, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return [32]byte{}, errors.New("component file is not regular")
	}
	sum := sha256.New()
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return [32]byte{}, err
		}
		count, err := file.Read(buffer)
		_, _ = sum.Write(buffer[:count])
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return [32]byte{}, err
		}
	}
	after, err := file.Stat()
	if err != nil || !sameChild(root, file, name) || before.Size() != after.Size() ||
		before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		return [32]byte{}, errors.New("component file changed while reading")
	}
	var result [32]byte
	copy(result[:], sum.Sum(nil))
	return result, nil
}

func openFile(root *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(root.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func openDir(root *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(root.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	info, err := file.Stat()
	stat, ok := infoSys(info).(*syscall.Stat_t)
	if err != nil || !ok || stat.Uid != uint32(os.Geteuid()) && stat.Uid != 0 {
		file.Close()
		return nil, errors.New("directory owner is not trusted")
	}
	return file, nil
}

func infoSys(info os.FileInfo) any {
	if info == nil {
		return nil
	}
	return info.Sys()
}

func readDir(root *os.File) ([]os.DirEntry, error) {
	file, err := openDir(root, ".")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	items, err := file.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b os.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return items, nil
}

func linkText(root *os.File, name string) (string, error) {
	data := make([]byte, 4096)
	count, err := unix.Readlinkat(int(root.Fd()), name, data)
	if err != nil || count == len(data) {
		return "", errors.Join(err, errors.New("cannot read complete link target"))
	}
	return string(data[:count]), nil
}

func makeStage(root *os.File) (*os.File, string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return nil, "", err
	}
	name := ".cdh-rehome-" + hex.EncodeToString(data[:])
	if err := unix.Mkdirat(int(root.Fd()), name, 0o755); err != nil {
		return nil, "", err
	}
	file, err := openDir(root, name)
	if err == nil {
		if chmodErr := file.Chmod(0o755); chmodErr != nil {
			fmt.Fprintf(os.Stderr, "warning: new migration directory mode could not be set: %s: %v\n", name, chmodErr)
		}
	}
	return file, name, err
}

func sameChild(parent, child *os.File, name string) bool {
	var got unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &got, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return false
	}
	info, err := child.Stat()
	stat, ok := infoSys(info).(*syscall.Stat_t)
	return err == nil && ok && uint64(got.Dev) == uint64(stat.Dev) && uint64(got.Ino) == uint64(stat.Ino)
}

func clearTree(root *os.File) error {
	items, err := readDir(root)
	if err != nil {
		return err
	}
	for _, item := range items {
		name := item.Name()
		flags := 0
		if item.IsDir() {
			child, err := openDir(root, name)
			if err != nil {
				return err
			}
			err = clearTree(child)
			if err == nil && !sameChild(root, child, name) {
				err = errors.New("cleanup directory identity changed")
			}
			child.Close()
			if err != nil {
				return err
			}
			flags = unix.AT_REMOVEDIR
		}
		if err := unix.Unlinkat(int(root.Fd()), name, flags); err != nil {
			return err
		}
	}
	return nil
}

func openPath(name string) (*os.File, error) {
	root, err := os.Open(string(filepath.Separator))
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Clean(name), string(filepath.Separator)), string(filepath.Separator)) {
		next, err := openDir(root, part)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = next
	}
	return root, nil
}

func removeEmpty(name string, root *os.File) error {
	parent, err := openPath(filepath.Dir(name))
	if err != nil {
		return err
	}
	defer parent.Close()
	base := filepath.Base(name)
	if !sameChild(parent, root, base) {
		return errors.New("legacy directory identity changed before removal")
	}
	return unix.Unlinkat(int(parent.Fd()), base, unix.AT_REMOVEDIR)
}

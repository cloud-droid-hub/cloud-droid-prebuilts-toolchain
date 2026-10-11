// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package rehome

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func renameDir(from, to *os.File, name string) error {
	return unix.Renameat2(int(from.Fd()), name, int(to.Fd()), name, unix.RENAME_NOREPLACE)
}

func busy(ctx context.Context, name string) error {
	items, err := os.ReadDir("/proc")
	if err != nil {
		return fmt.Errorf("inspect legacy toolchain processes: %w", err)
	}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		pid := item.Name()
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		base := filepath.Join("/proc", pid)
		info, err := os.Stat(base)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		stat, ok := infoSys(info).(*syscall.Stat_t)
		if err != nil || !ok {
			return fmt.Errorf("inspect process %s: %w", pid, err)
		}
		if stat.Uid != uint32(os.Geteuid()) {
			continue
		}
		if pid == strconv.Itoa(os.Getpid()) {
			continue
		}
		paths := []string{filepath.Join(base, "exe"), filepath.Join(base, "cwd")}
		for _, dir := range []string{"fd"} {
			files, err := os.ReadDir(filepath.Join(base, dir))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return fmt.Errorf("inspect process %s %s: %w", pid, dir, err)
			}
			for _, file := range files {
				paths = append(paths, filepath.Join(base, dir, file.Name()))
			}
		}
		maps, err := os.ReadFile(filepath.Join(base, "maps"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect process %s mappings: %w", pid, err)
		}
		for _, line := range strings.Split(string(maps), "\n") {
			rest := line
			for range 5 {
				rest = strings.TrimLeft(rest, " \t")
				index := strings.IndexAny(rest, " \t")
				if index < 0 {
					rest = ""
					break
				}
				rest = rest[index:]
			}
			target := strings.TrimSuffix(strings.TrimLeft(rest, " \t"), " (deleted)")
			if target == name || strings.HasPrefix(target, name+string(filepath.Separator)) {
				return fmt.Errorf("legacy toolchain is mapped by process %s: %s", pid, target)
			}
		}
		for _, path := range paths {
			target, err := os.Readlink(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return fmt.Errorf("inspect process %s resource: %w", pid, err)
			}
			target = strings.TrimSuffix(target, " (deleted)")
			if target == name || strings.HasPrefix(target, name+string(filepath.Separator)) {
				return fmt.Errorf("legacy toolchain is in use by process %s: %s", pid, target)
			}
		}
	}
	return nil
}

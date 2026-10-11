// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/rehome"
)

func rehomeRoot(ctx context.Context, repo, dest string, root *os.File, spec catalog.Spec, out io.Writer) error {
	base := filepath.Dir(filepath.Dir(repo))
	want := filepath.Join(base, "output", "target", "common", "toolchains", spec.Host)
	if filepath.Base(filepath.Dir(repo)) != "toolchains" || filepath.Base(repo) != spec.Host || dest != want {
		return nil
	}
	oldPath := filepath.Join(base, "output", "toolchains", spec.Host)
	old, err := openTree(oldPath, false)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer old.Close()
	if err := checkFence(ctx, old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	names := make([]string, len(spec.Archives))
	byName := make(map[string]catalog.Archive, len(spec.Archives))
	for index, arc := range spec.Archives {
		names[index] = arc.Dir
		byName[arc.Dir] = arc
	}
	check := func(file *os.File, name string) error {
		return checkComp(ctx, file, byName[name])
	}
	return rehome.Move(ctx, old, root, oldPath, dest, names, check, fenceData, out)
}

// CheckLegacy只核对默认安装布局，保留未能安全迁移的旧条目并报告失败。
func CheckLegacy(repo, dest, host string) error {
	repo, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	base := filepath.Dir(filepath.Dir(repo))
	if filepath.Base(filepath.Dir(repo)) != "toolchains" || filepath.Base(repo) != host ||
		dest != filepath.Join(base, "output", "target", "common", "toolchains", host) {
		return nil
	}
	old := filepath.Join(base, "output", "toolchains")
	if _, err := os.Lstat(old); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("legacy toolchain entries remain and were preserved: %s; inspect the reported differences before removing them", old)
}

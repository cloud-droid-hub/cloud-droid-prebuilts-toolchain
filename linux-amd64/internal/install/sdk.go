// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package install

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cloud-droid-hub/cloud-droid-prebuilts-toolchain/internal/catalog"
	"golang.org/x/sys/unix"
)

// renameSDK核验完整SDK后改名，已有同名旧组件保存在sdk37-original。
func renameSDK(ctx context.Context, root *os.File, archive catalog.Archive) (bool, error) {
	if archive.Name != "Android SDK" || archive.Dir != "sdk37" {
		return false, nil
	}
	prior, err := openDir(root, "sdkapp37")
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer prior.Close()
	if err := checkComp(ctx, prior, archive); err != nil {
		return false, fmt.Errorf("legacy SDK name has invalid contents: %w", err)
	}
	dest, err := openDir(root, archive.Dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if !sameChild(root, prior, "sdkapp37") {
			return false, errors.New("SDK directory changed before rename")
		}
		if err := moveAs(root, "sdkapp37", archive.Dir); err != nil {
			return false, err
		}
		if !sameChild(root, prior, archive.Dir) {
			return false, errors.New("SDK directory identity changed during rename")
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer dest.Close()
	if err := checkComp(ctx, dest, archive); err == nil {
		return false, errors.New("both SDK names contain valid installations; preserve both directories")
	}
	var saved unix.Stat_t
	if err := unix.Fstatat(int(root.Fd()), "sdk37-original", &saved, unix.AT_SYMLINK_NOFOLLOW); !errors.Is(err, unix.ENOENT) {
		if err == nil {
			err = errors.New("SDK backup already exists")
		}
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !sameChild(root, prior, "sdkapp37") || !sameChild(root, dest, archive.Dir) {
		return false, errors.New("SDK directory changed before migration")
	}
	if err := moveAs(root, "sdkapp37", "sdk37-original"); err != nil {
		return false, err
	}
	if !sameChild(root, prior, "sdk37-original") || !sameChild(root, dest, archive.Dir) {
		return false, errors.New("SDK directory changed before exchange")
	}
	if err := swapAt(root, archive.Dir, "sdk37-original"); err != nil {
		if sameChild(root, prior, "sdk37-original") {
			err = errors.Join(err, moveAs(root, "sdk37-original", "sdkapp37"))
		}
		return false, err
	}
	if !sameChild(root, prior, archive.Dir) || !sameChild(root, dest, "sdk37-original") {
		return false, errors.New("SDK directory identity changed during exchange")
	}
	return true, nil
}

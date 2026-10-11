// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package rehome

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func renameDir(from, to *os.File, name string) error {
	return unix.RenameatxNp(int(from.Fd()), name, int(to.Fd()), name, unix.RENAME_EXCL)
}

func busy(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "/usr/sbin/lsof", "-nP", "-a", "-p", "^"+strconv.Itoa(os.Getpid()), "-F", "n", "+D", name)
	var errs strings.Builder
	cmd.Stderr = &errs
	out, err := cmd.Output()
	if len(out) != 0 || errs.Len() != 0 {
		return fmt.Errorf("legacy toolchain is in use or cannot be inspected: %s: %s%s", name, out, errs.String())
	}
	var code *exec.ExitError
	if err != nil && !(errors.As(err, &code) && code.ExitCode() == 1) {
		return fmt.Errorf("inspect legacy toolchain use: %w", err)
	}
	return ctx.Err()
}

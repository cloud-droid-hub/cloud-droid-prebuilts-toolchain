// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// macOS的出厂libarchive把XZ归档流式转换为TAR，不展开任何输入路径。
func nativeStream(ctx context.Context, raw *os.File) (io.Reader, func() error, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/tar", "-cf", "-", "@"+raw.Name())
	cmd.Dir = filepath.Dir(raw.Name())
	cmd.Env = append(os.Environ(), "TMPDIR="+cmd.Dir, "TMP="+cmd.Dir, "TEMP="+cmd.Dir)
	var log bytes.Buffer
	cmd.Stderr = &log
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		pipe.Close()
		return nil, nil, err
	}
	finish := func() error {
		close := pipe.Close()
		end := cmd.Wait()
		if end != nil {
			return errors.Join(close, fmt.Errorf("出厂TAR归档流转换：%s：%w", log.String(), end))
		}
		return close
	}
	return pipe, finish, nil
}

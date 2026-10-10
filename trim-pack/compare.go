// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
	"github.com/ulikunitz/xz"
)

func compareOld(ctx context.Context, plan spec, b *bundle, data *os.File, dir string) (ret error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".part-") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 || !strings.HasSuffix(names[0], ".part-000") {
		return errors.New("现有制品没有从000开始的分片")
	}
	base := strings.TrimSuffix(names[0], "000")
	work, clean, err := workpath.MkdirTemp("trim-compare-")
	if err != nil {
		return err
	}
	defer func() { ret = errors.Join(ret, clean()) }()
	raw, err := newFile(filepath.Join(work, "published.bin"))
	if err != nil {
		return err
	}
	defer func() { ret = errors.Join(ret, raw.Close()) }()
	hash := sha256.New()
	for index, name := range names {
		if name != fmt.Sprintf("%s%03d", base, index) {
			return fmt.Errorf("现有分片不连续或包含其它归档：%s", name)
		}
		file, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			file.Close()
			return fmt.Errorf("现有分片不是普通文件：%s：%v", name, err)
		}
		_, err = io.Copy(io.MultiWriter(raw, hash), ctxRead{ctx: ctx, in: file})
		if err = errors.Join(err, file.Close()); err != nil {
			return err
		}
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != plan.OldSHA {
		return fmt.Errorf("现有归档摘要不符：got=%s want=%s", got, plan.OldSHA)
	}
	if _, err := raw.Seek(0, io.SeekStart); err != nil {
		return err
	}
	reader := bufio.NewReader(ctxRead{ctx: ctx, in: raw})
	magic, err := reader.Peek(6)
	if err != nil {
		return err
	}
	var source io.Reader
	if magic[0] == 0x1f && magic[1] == 0x8b {
		gr, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer gr.Close()
		source = gr
	} else {
		if runtime.GOOS == "darwin" {
			if _, err := raw.Seek(0, io.SeekStart); err != nil {
				return err
			}
			var finish func() error
			source, finish, err = nativeStream(ctx, raw)
			if err != nil {
				return err
			}
			defer func() { ret = errors.Join(ret, finish()) }()
		} else {
			source, err = xz.NewReader(reader)
			if err != nil {
				return err
			}
		}
	}
	tr := tar.NewReader(source)
	seen := make(map[string]bool)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name, err := cleanName(h.Name)
		if err != nil {
			return err
		}
		if name == plan.Prefix {
			if h.Typeflag != tar.TypeDir {
				return errors.New("现有归档根不是目录")
			}
			continue
		}
		if !strings.HasPrefix(name, plan.Prefix+"/") {
			return fmt.Errorf("现有归档超出登记根：%s", name)
		}
		name = strings.TrimPrefix(name, plan.Prefix+"/")
		if seen[name] {
			return fmt.Errorf("现有归档路径重复：%s", name)
		}
		seen[name] = true
		item := b.Items[name]
		if item == nil || item.Kind != h.Typeflag && !(item.Kind == tar.TypeReg && h.Typeflag == tar.TypeRegA) {
			return fmt.Errorf("现有制品文件集合或类型不符：%s type=%d", name, h.Typeflag)
		}
		if h.Mode&0o111 != item.Mode&0o111 {
			return fmt.Errorf("现有制品执行权限不符：%s old=%04o new=%04o", name, h.Mode, item.Mode)
		}
		if item.Kind == tar.TypeSymlink {
			if h.Linkname != plan.Links[name] {
				return fmt.Errorf("现有制品链接目标不符：%s", name)
			}
		}
		if item.Kind != tar.TypeReg {
			continue
		}
		oldHash, newHash := sha256.New(), sha256.New()
		n, err := io.Copy(oldHash, tr)
		if err != nil || n != item.Size {
			return fmt.Errorf("现有制品大小不符：%s old=%d new=%d：%v", name, n, item.Size, err)
		}
		var input io.Reader = io.NewSectionReader(data, item.Off, item.Size)
		if item.Data != nil {
			input = strings.NewReader(string(item.Data))
		}
		if _, err := io.Copy(newHash, input); err != nil {
			return err
		}
		if fmt.Sprintf("%x", oldHash.Sum(nil)) != fmt.Sprintf("%x", newHash.Sum(nil)) {
			return fmt.Errorf("现有制品保留字节不符：%s", name)
		}
	}
	for name := range b.Items {
		if !seen[name] {
			return fmt.Errorf("新制品增加了原归档没有的路径：%s", name)
		}
	}
	_, err = io.Copy(io.Discard, source)
	return err
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
)

type part struct {
	Name   string
	Size   int64
	SHA256 string
}

type source struct {
	Name   string
	SHA256 string
	URL    string
}

type report struct {
	Host       string
	Profile    string
	Sources    []source
	SHA256     string
	Bytes      int64
	Data       int64
	Entries    int
	Dropped    []string
	Parts      []part
	Transforms []string
}

type countWrite struct {
	out io.Writer
	n   int64
}

func (w *countWrite) Write(buf []byte) (int, error) {
	n, err := w.out.Write(buf)
	w.n += int64(n)
	return n, err
}

func withSource(ctx context.Context, plan spec, dir string, call func(*bundle, *os.File) error) (ret error) {
	work, clean, err := workpath.MkdirTemp("trim-pack-")
	if err != nil {
		return err
	}
	defer func() { ret = errors.Join(ret, clean()) }()
	if err := freeSpace(work); err != nil {
		return err
	}
	data, err := newFile(filepath.Join(work, "payload.data"))
	if err != nil {
		return err
	}
	defer func() { ret = errors.Join(ret, data.Close()) }()
	b := &bundle{Items: make(map[string]*node)}
	tool := ""
	for _, in := range plan.Inputs {
		if filepath.Base(in.Name) != in.Name || in.Name == "." || in.Name == ".." {
			return errors.New("原始输入必须是登记的单个文件名")
		}
		raw, err := snapshot(ctx, in, dir, work)
		if err != nil {
			return err
		}
		if plan.Patch && in.Name == "patchelf-0.18.0.tar.gz" {
			tool, err = patcher(ctx, raw, work)
		} else {
			err = readInput(ctx, plan, in, raw, b, data)
		}
		end := raw.Close()
		remove := os.Remove(filepath.Join(work, "source.bin"))
		if err != nil || end != nil || remove != nil {
			return errors.Join(err, end, remove)
		}
	}
	if plan.Patch {
		if tool == "" {
			return errors.New("缺少固定patchelf源码输入")
		}
		if err := patchFiles(ctx, b, data, tool, work, plan); err != nil {
			return err
		}
	}
	if err := finish(plan, b, data); err != nil {
		return err
	}
	if len(b.Items) == 0 {
		return errors.New("策略没有生成文件")
	}
	return call(b, data)
}

func newReport(plan spec, b *bundle) report {
	rep := report{Host: plan.Host, Profile: plan.Comp + "-v1", Data: b.Size,
		Entries: len(b.Items), Dropped: b.Drop}
	for _, in := range plan.Inputs {
		rep.Sources = append(rep.Sources, source{Name: in.Name, SHA256: in.SHA256, URL: origin(in)})
		for _, pick := range in.Rules {
			if pick.Thin {
				rep.Transforms = []string{"Mach-O arm64 slice"}
				break
			}
		}
	}
	if plan.Patch {
		tag := "DT_RUNPATH"
		if plan.ForceR {
			tag = "DT_RPATH --force-rpath"
		}
		rep.Transforms = append(rep.Transforms, "patchelf 0.18.0 --set-rpath $ORIGIN/<relative lib/cdh> "+tag)
		var names []string
		for name := range plan.PreRPath {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			rep.Transforms = append(rep.Transforms, "patchelf intermediate "+name+" "+plan.PreRPath[name]+" before final relative RPATH")
		}
		names = nil
		for name, keep := range plan.RunPaths {
			if keep {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			rep.Transforms = append(rep.Transforms, "patchelf DT_RUNPATH preserved "+name)
		}
	}
	if len(plan.Aliases) != 0 {
		rep.Transforms = append(rep.Transforms, "typed aliases as ordinary files")
	}
	if len(plan.Packages) != 0 {
		rep.Transforms = append(rep.Transforms, "typed SDK package.xml with fixed license")
	}
	if plan.DataOnly {
		rep.Transforms = append(rep.Transforms, "target data regular files 0644")
	}
	return rep
}

func generate(ctx context.Context, plan spec, source, dest string, log io.Writer) (ret error) {
	dest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	root, err := openParent(dest)
	if err != nil {
		return err
	}
	defer func() { ret = errors.Join(ret, root.Close()) }()
	parent := filepath.Dir(dest)
	rootInfo, err := root.Stat()
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		return fmt.Errorf("保留已有或不可检查的输出：%s：%v", dest, err)
	}
	if err := freeSpace(parent); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(parent, "trim-pack-")
	if err != nil {
		return err
	}
	if err := os.Chmod(stage, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "警告：新候选设置0755失败：%v\n", err)
	}
	info, err := os.Lstat(stage)
	if err != nil {
		return err
	}
	defer func() {
		if got, err := os.Lstat(stage); err == nil && os.SameFile(info, got) {
			ret = errors.Join(ret, os.RemoveAll(stage))
		}
	}()
	var rep report
	err = withSource(ctx, plan, source, func(b *bundle, data *os.File) error {
		rep = newReport(plan, b)
		file, err := newFile(filepath.Join(stage, "payload.tar.gz"))
		if err != nil {
			return err
		}
		hash := sha256.New()
		count := &countWrite{out: io.MultiWriter(file, hash)}
		err = writeArc(ctx, plan, b, data, count)
		if err == nil {
			err = file.Sync()
		}
		if err == nil {
			_, err = file.Seek(0, io.SeekStart)
		}
		if err == nil {
			rep.SHA256, rep.Bytes = fmt.Sprintf("%x", hash.Sum(nil)), count.n
			rep.Parts, err = split(ctx, file, count.n, stage)
		}
		if err = errors.Join(err, file.Close()); err != nil {
			return err
		}
		return saveReport(stage, rep)
	})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	got, err := os.Lstat(parent)
	next, end := os.Lstat(stage)
	if err != nil || end != nil || !os.SameFile(rootInfo, got) || !os.SameFile(info, next) {
		return errors.New("发布前父目录或候选身份改变")
	}
	if err := freeSpace(parent); err != nil {
		return err
	}
	if err := publish(int(root.Fd()), filepath.Base(stage), filepath.Base(dest)); err != nil {
		return fmt.Errorf("不覆盖发布失败：%w", err)
	}
	if err := root.Sync(); err != nil {
		return err
	}
	_, err = fmt.Fprintf(log, "pack %s/%s files=%d bytes=%d sha256=%s\n", plan.Host, plan.Comp, rep.Entries, rep.Bytes, rep.SHA256)
	return err
}

func split(ctx context.Context, raw io.Reader, size int64, dir string) ([]part, error) {
	var parts []part
	for left, index := size, 0; left > 0; index++ {
		name := fmt.Sprintf("payload.tar.gz.part-%03d", index)
		file, err := newFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		hash := sha256.New()
		want := min(left, partLimit)
		n, err := io.CopyN(io.MultiWriter(file, hash), ctxRead{ctx: ctx, in: raw}, want)
		if err == nil {
			err = file.Sync()
		}
		err = errors.Join(err, file.Close())
		if err != nil || n != want {
			return nil, fmt.Errorf("写入分片%s bytes=%d want=%d：%v", name, n, want, err)
		}
		parts = append(parts, part{Name: name, Size: n, SHA256: fmt.Sprintf("%x", hash.Sum(nil))})
		left -= n
	}
	return parts, nil
}

func jsonBytes(rep report) ([]byte, error) {
	data, err := json.MarshalIndent(rep, "", "\t")
	return append(data, '\n'), err
}

func saveReport(dir string, rep report) error {
	file, err := newFile(filepath.Join(dir, "report.json"))
	if err != nil {
		return err
	}
	data, err := jsonBytes(rep)
	if err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	return errors.Join(err, file.Close())
}

func sumFile(ctx context.Context, name string) (string, int64, error) {
	info, err := os.Lstat(name)
	if err != nil || !info.Mode().IsRegular() {
		return "", 0, fmt.Errorf("制品不是普通文件：%s：%v", name, err)
	}
	file, err := os.Open(name)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	n, err := io.Copy(hash, ctxRead{ctx: ctx, in: file})
	got, end := file.Stat()
	cur, pathErr := os.Lstat(name)
	if err == nil && (end != nil || pathErr != nil || !os.SameFile(info, got) || !os.SameFile(info, cur) ||
		n != info.Size() || info.ModTime() != got.ModTime()) {
		err = errors.New("核验期间制品发生变化")
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), n, err
}

func verify(ctx context.Context, plan spec, source, dest, old string, log io.Writer) error {
	if err := safePath(dest); err != nil {
		return err
	}
	return withSource(ctx, plan, source, func(b *bundle, data *os.File) error {
		rep := newReport(plan, b)
		hash := sha256.New()
		count := &countWrite{out: hash}
		if err := writeArc(ctx, plan, b, data, count); err != nil {
			return err
		}
		rep.SHA256, rep.Bytes = fmt.Sprintf("%x", hash.Sum(nil)), count.n
		got, size, err := sumFile(ctx, filepath.Join(dest, "payload.tar.gz"))
		if err != nil || got != rep.SHA256 || size != rep.Bytes {
			return fmt.Errorf("归档与原始源重放不符：sha=%s size=%d：%v", got, size, err)
		}
		for left, index := rep.Bytes, 0; left > 0; index++ {
			name := fmt.Sprintf("payload.tar.gz.part-%03d", index)
			got, size, err := sumFile(ctx, filepath.Join(dest, name))
			if err != nil || size != min(left, partLimit) {
				return fmt.Errorf("分片%s长度不符：%v", name, err)
			}
			file, err := os.Open(filepath.Join(dest, "payload.tar.gz"))
			if err != nil {
				return err
			}
			hash := sha256.New()
			_, err = io.Copy(hash, io.NewSectionReader(file, rep.Bytes-left, size))
			err = errors.Join(err, file.Close())
			if err != nil || got != fmt.Sprintf("%x", hash.Sum(nil)) {
				return fmt.Errorf("分片%s字节不符：%v", name, err)
			}
			rep.Parts = append(rep.Parts, part{Name: name, Size: size, SHA256: got})
			left -= size
		}
		want := []string{"payload.tar.gz", "report.json"}
		for _, item := range rep.Parts {
			want = append(want, item.Name)
		}
		sort.Strings(want)
		entries, err := os.ReadDir(dest)
		if err != nil || len(entries) != len(want) {
			return fmt.Errorf("制品目录文件集合不符：%v", err)
		}
		for index, entry := range entries {
			if entry.Name() != want[index] {
				return fmt.Errorf("未知制品文件：%s", entry.Name())
			}
		}
		body, err := os.ReadFile(filepath.Join(dest, "report.json"))
		if err != nil {
			return err
		}
		canonical, err := jsonBytes(rep)
		if err != nil || !bytes.Equal(body, canonical) {
			return fmt.Errorf("报告与重放结果不符：%v", err)
		}
		if old != "" {
			if err := compareOld(ctx, plan, b, data, old); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(log, "verify %s/%s files=%d sha256=%s published=%t\n", plan.Host, plan.Comp, rep.Entries, rep.SHA256, old != "")
		return err
	})
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/cloud-droid-hub/cloud-droid-hub/pkg/devicekit/workpath"
	"golang.org/x/sys/unix"
)

type part struct {
	Name   string
	Size   int64
	SHA256 string
}

type report struct {
	Host       string
	Profile    string
	SourceSHA  string
	ArchiveSHA string
	Before     int64
	After      int64
	Bytes      int64
	Files      int
	Dropped    []string
	Parts      []part
}

type countWrite struct {
	out io.Writer
	n   int64
}

func (w *countWrite) Write(data []byte) (int, error) {
	n, err := w.out.Write(data)
	w.n += int64(n)
	return n, err
}

func withSource(ctx context.Context, spec plan, name string, call func(*bundle, *os.File) error) (err error) {
	work, clean, err := workpath.MkdirTemp("jdk-trim-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, clean()) }()
	if err := freeSpace(work); err != nil {
		return err
	}
	data, err := newFile(filepath.Join(work, "payload.data"))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, data.Close()) }()
	in, err := openSource(spec, name)
	if err != nil {
		return err
	}
	defer in.close()
	b, err := scan(ctx, spec, in.reader(), data)
	if err != nil {
		return err
	}
	if err := in.check(); err != nil {
		return err
	}
	if err := call(b, data); err != nil {
		return err
	}
	return in.check()
}

func newReport(spec plan, b *bundle) report {
	return report{Host: spec.Host, Profile: profile, SourceSHA: spec.SrcSHA,
		Before: b.Before, After: b.After, Files: len(b.Items), Dropped: b.Dropped}
}

func pack(ctx context.Context, spec plan, in, dest string) (report, error) {
	var rep report
	dest, err := filepath.Abs(dest)
	if err != nil {
		return rep, err
	}
	parent := filepath.Dir(dest)
	if err := safePath(parent); err != nil {
		return rep, err
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		return rep, fmt.Errorf("保留已有或不可检查的目标：%s err=%v", dest, err)
	}
	if err := freeSpace(parent); err != nil {
		return rep, err
	}
	fd, err := unix.Open(parent, unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return rep, err
	}
	root := os.NewFile(uintptr(fd), parent)
	defer root.Close()
	rootInfo, err := root.Stat()
	if err != nil {
		return rep, err
	}
	stat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() {
		return rep, errors.New("输出父目录不属于当前用户")
	}
	if rootInfo.Mode().Perm() != 0o755 {
		fmt.Fprintf(os.Stderr, "警告：已有目录权限为%04o，建议0755：%s\n", rootInfo.Mode().Perm(), parent)
	}
	stage, err := os.MkdirTemp(parent, "jdk-trim-")
	if err != nil {
		return rep, err
	}
	if err := os.Chmod(stage, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "警告：设置0755失败：%s：%v\n", stage, err)
	}
	stageInfo, err := os.Lstat(stage)
	if err != nil {
		return rep, err
	}
	defer func() {
		info, err := os.Lstat(stage)
		if err == nil && os.SameFile(stageInfo, info) {
			os.RemoveAll(stage)
		}
	}()
	err = withSource(ctx, spec, in, func(b *bundle, data *os.File) error {
		rep = newReport(spec, b)
		name := spec.Stem + ".tar.gz"
		out, err := newFile(filepath.Join(stage, name))
		if err != nil {
			return err
		}
		hash := sha256.New()
		count := &countWrite{out: io.MultiWriter(out, hash)}
		err = writeArc(ctx, b, data, count)
		if err == nil {
			err = out.Sync()
		}
		if err != nil {
			out.Close()
			return err
		}
		rep.ArchiveSHA, rep.Bytes = hex.EncodeToString(hash.Sum(nil)), count.n
		if _, err := out.Seek(0, io.SeekStart); err != nil {
			out.Close()
			return err
		}
		for left, index := count.n, 0; left > 0; index++ {
			size := min(left, partLimit)
			file := fmt.Sprintf("%s.part-%03d", name, index)
			if err := copyFile(ctx, out, filepath.Join(stage, file), size); err != nil {
				out.Close()
				return err
			}
			sum, got, err := fileSum(ctx, filepath.Join(stage, file))
			if err != nil || got != size {
				out.Close()
				return fmt.Errorf("核验分片%s：size=%d want=%d err=%v", file, got, size, err)
			}
			rep.Parts = append(rep.Parts, part{Name: file, Size: size, SHA256: sum})
			left -= size
		}
		if err := out.Close(); err != nil {
			return err
		}
		return writeReport(stage, rep)
	})
	if err != nil {
		return rep, err
	}
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	info, err := os.Lstat(parent)
	got, stageErr := os.Lstat(stage)
	if err != nil || stageErr != nil || !os.SameFile(rootInfo, info) || !os.SameFile(stageInfo, got) {
		return rep, errors.New("发布前父目录或候选身份改变")
	}
	if err := freeSpace(parent); err != nil {
		return rep, err
	}
	stageDir, err := os.Open(stage)
	if err != nil {
		return rep, err
	}
	syncErr := stageDir.Sync()
	closeErr := stageDir.Close()
	if syncErr != nil || closeErr != nil {
		return rep, errors.Join(syncErr, closeErr)
	}
	if err := publish(fd, filepath.Base(stage), filepath.Base(dest)); err != nil {
		return rep, fmt.Errorf("不覆盖发布失败：%w", err)
	}
	return rep, root.Sync()
}

func writeReport(dir string, rep report) error {
	file, err := newFile(filepath.Join(dir, "report.json"))
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(rep, "", "\t")
	if err == nil {
		_, err = file.Write(append(data, '\n'))
	}
	if err == nil {
		err = file.Sync()
	}
	end := file.Close()
	if err != nil {
		return err
	}
	return end
}

func fileSum(ctx context.Context, name string) (string, int64, error) {
	file, info, err := openFile(name)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	n, err := io.Copy(hash, ctxRead{ctx: ctx, in: file})
	if err == nil && n != info.Size() {
		err = errors.New("文件长度在读取期间改变")
	}
	got, end := file.Stat()
	path, pathErr := os.Lstat(name)
	if err == nil && (end != nil || pathErr != nil || !os.SameFile(info, got) || !os.SameFile(info, path) ||
		info.Size() != got.Size() || info.ModTime() != got.ModTime()) {
		err = errors.New("文件身份在读取期间改变")
	}
	return hex.EncodeToString(hash.Sum(nil)), n, err
}

func verify(ctx context.Context, spec plan, in, dest string) error {
	if err := safePath(dest); err != nil {
		return err
	}
	return withSource(ctx, spec, in, func(b *bundle, data *os.File) error {
		rep := newReport(spec, b)
		hash := sha256.New()
		count := &countWrite{out: hash}
		if err := writeArc(ctx, b, data, count); err != nil {
			return err
		}
		rep.ArchiveSHA, rep.Bytes = hex.EncodeToString(hash.Sum(nil)), count.n
		name := spec.Stem + ".tar.gz"
		sum, size, err := fileSum(ctx, filepath.Join(dest, name))
		if err != nil || size != rep.Bytes || sum != rep.ArchiveSHA {
			return fmt.Errorf("重放归档字节不符：sha=%s bytes=%d err=%v", sum, size, err)
		}
		joined := sha256.New()
		want := []string{name, "report.json"}
		for left, index := rep.Bytes, 0; left > 0; index++ {
			file := fmt.Sprintf("%s.part-%03d", name, index)
			want = append(want, file)
			path := filepath.Join(dest, file)
			sum, size, err := fileSum(ctx, path)
			if err != nil || size != min(left, partLimit) {
				return fmt.Errorf("分片不符：%s size=%d err=%v", file, size, err)
			}
			partFile, _, err := openFile(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(joined, ctxRead{ctx: ctx, in: partFile})
			partFile.Close()
			if err != nil {
				return err
			}
			rep.Parts = append(rep.Parts, part{Name: file, Size: size, SHA256: sum})
			left -= size
		}
		if hex.EncodeToString(joined.Sum(nil)) != rep.ArchiveSHA {
			return errors.New("连接后的分片不能重放归档")
		}
		expect, err := json.MarshalIndent(rep, "", "\t")
		if err != nil {
			return err
		}
		file, info, err := openFile(filepath.Join(dest, "report.json"))
		if err != nil {
			return err
		}
		defer file.Close()
		if info.Size() != int64(len(expect)+1) {
			return errors.New("生成报告长度不符")
		}
		got, err := io.ReadAll(io.LimitReader(file, info.Size()+1))
		if err != nil || !bytes.Equal(got, append(expect, '\n')) {
			return errors.New("生成报告不符")
		}
		latest, err := os.Lstat(filepath.Join(dest, "report.json"))
		if err != nil || !os.SameFile(info, latest) || info.ModTime() != latest.ModTime() || info.Size() != latest.Size() {
			return errors.New("核验期间生成报告改变")
		}
		entries, err := os.ReadDir(dest)
		if err != nil {
			return err
		}
		var names []string
		for _, item := range entries {
			names = append(names, item.Name())
		}
		sort.Strings(names)
		sort.Strings(want)
		if !equalNames(names, want) {
			return errors.New("制品目录含缺失或额外文件")
		}
		return nil
	})
}

func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

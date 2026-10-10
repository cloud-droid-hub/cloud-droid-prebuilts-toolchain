// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/sys/unix"
)

const profile = "python314-build-v1"
const pyVersion = "3.14.8"
const release = "20261009"

type source struct {
	Host   string
	Target string
	Size   int64
	SHA256 string
}

var sources = []source{
	{"darwin-arm64", "aarch64-apple-darwin", 61701302, "6888111560a89c16e471894cca0f50d66f3bc00e5a16e91fab24c50e567ef5ee"},
	{"linux-arm64", "aarch64-unknown-linux-gnu", 96827232, "e8b3b7186d7a11335f3d1f05ee1b19b469cd5bcaeabd6293afa3713f24ff7bf0"},
	{"linux-amd64", "x86_64-unknown-linux-gnu", 111243845, "d4d83a2dbb9b7ee12373219113e9aa59ad943b8d381a43837209945bb5596e66"},
}

type report struct {
	Before  int64
	After   int64
	Files   int
	Removed int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("python-trim", flag.ContinueOnError)
	host := fs.String("host", "", "darwin-arm64、linux-arm64或linux-amd64")
	in := fs.String("source", "", "登记的完整上游tar.zst")
	dest := fs.String("dest", "", "新裁剪tar.gz，已有目标不覆盖")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 || *in == "" || *dest == "" {
		return errors.New("source与dest必填，不接受位置参数")
	}
	for _, spec := range sources {
		if spec.Host == *host {
			return pack(ctx, spec, *in, *dest)
		}
	}
	return fmt.Errorf("未登记宿主：%q", *host)
}

// 移除构建对象、静态库、测试、缓存和手册，运行库与许可证保持原字节。
func drop(name string) bool {
	if name == "lib/libpython3.14.a" || strings.HasPrefix(name, "lib/python3.14/config-") &&
		(strings.HasSuffix(name, "/libpython3.14.a") || strings.HasSuffix(name, ".o")) {
		return true
	}
	if name == "share/man" || strings.HasPrefix(name, "share/man/") {
		return true
	}
	if !strings.HasPrefix(name, "lib/python3.14/") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "test" || part == "tests" || part == "__pycache__" {
			return true
		}
	}
	return strings.HasSuffix(name, ".pyc") || strings.HasSuffix(name, ".pyo")
}

func outputName(name string) (string, error) {
	name = strings.TrimSuffix(name, "/")
	if name == "" || path.Clean(name) != name || strings.Contains(name, "\\") ||
		(name != "python" && !strings.HasPrefix(name, "python/")) {
		return "", fmt.Errorf("非法归档路径：%q", name)
	}
	switch {
	case strings.HasPrefix(name, "python/install/"):
		rel := strings.TrimPrefix(name, "python/install/")
		if !drop(rel) {
			return "python/" + rel, nil
		}
	case strings.HasPrefix(name, "python/licenses/"):
		return name, nil
	case strings.HasPrefix(name, "python/LICENSE"):
		return "python/licenses/" + path.Base(name), nil
	case name == "python/PYTHON.json":
		return "python/upstream.json", nil
	}
	return "", nil
}

func trim(ctx context.Context, spec source, in io.Reader, out io.Writer) (report, error) {
	var rep report
	gz := gzip.NewWriter(out)
	gz.Header.OS = 255
	tw := tar.NewWriter(gz)
	tr := tar.NewReader(in)
	seen := make(map[string]bool)
	links := make(map[string]string)
	for {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return rep, err
		}
		rep.Before += h.Size
		name, err := outputName(h.Name)
		if err != nil {
			return rep, err
		}
		if name == "" {
			rep.Removed++
			continue
		}
		if seen[name] {
			return rep, fmt.Errorf("重复归档路径：%s", name)
		}
		seen[name] = true
		h.Name = name
		h.Uid, h.Gid, h.Uname, h.Gname = 0, 0, "", ""
		h.ModTime, h.AccessTime, h.ChangeTime = time.Unix(0, 0), time.Time{}, time.Time{}
		h.PAXRecords, h.Xattrs = nil, nil
		mode := h.Mode
		h.Mode = 0o644
		switch h.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			if mode&0o111 != 0 {
				h.Mode = 0o755
			}
		case tar.TypeDir:
			h.Mode = 0o755
		case tar.TypeSymlink:
			if path.IsAbs(h.Linkname) || strings.Contains(h.Linkname, "\\") {
				return rep, fmt.Errorf("非法链接目标：%s：%s", name, h.Linkname)
			}
			target := path.Clean(path.Join(path.Dir(name), h.Linkname))
			if !strings.HasPrefix(target, "python/") {
				return rep, fmt.Errorf("链接逃逸：%s：%s", name, target)
			}
			links[name] = target
			h.Mode = 0o755
		default:
			return rep, fmt.Errorf("不支持的归档条目：%s：%d", name, h.Typeflag)
		}
		if err := tw.WriteHeader(h); err != nil {
			return rep, err
		}
		if _, err := io.Copy(tw, tr); err != nil {
			return rep, err
		}
		rep.After += h.Size
		rep.Files++
	}
	for name, target := range links {
		if !seen[target] {
			return rep, fmt.Errorf("裁剪后链接目标缺失：%s：%s", name, target)
		}
	}
	for _, name := range []string{"bin/python3.14", "lib/python3.14/os.py", "lib/python3.14/venv/__init__.py",
		"lib/python3.14/ensurepip/__init__.py", "lib/python3.14/site-packages/pip/__init__.py", "licenses/LICENSE.cpython.txt"} {
		if !seen["python/"+name] {
			return rep, fmt.Errorf("缺少必要文件：%s", name)
		}
	}
	mark := fmt.Sprintf("profile=%s\nversion=%s\nhost=%s\nrelease=%s\nsource_sha256=%s\nremoved=build,libpython-static,test,tests,__pycache__,pyc,pyo,share/man\n",
		profile, pyVersion, spec.Host, release, spec.SHA256)
	if err := tw.WriteHeader(&tar.Header{Name: "python/cdh-python-trim.txt", Mode: 0o644, Size: int64(len(mark)), ModTime: time.Unix(0, 0)}); err != nil {
		return rep, err
	}
	if _, err := io.WriteString(tw, mark); err != nil {
		return rep, err
	}
	if err := tw.Close(); err != nil {
		return rep, err
	}
	return rep, gz.Close()
}

func pack(ctx context.Context, spec source, name, dest string) (err error) {
	in, err := os.Open(name)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != spec.Size {
		return fmt.Errorf("上游输入长度或类型不符：%s：%v", name, err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, in); err != nil {
		return err
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != spec.SHA256 {
		return errors.New("上游输入SHA256不符")
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		return err
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	parent := filepath.Dir(dest)
	for dir := parent; ; dir = filepath.Dir(dir) {
		got, err := os.Lstat(dir)
		if err != nil || !got.IsDir() || got.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("输出父路径不是实际目录：%s：%v", dir, err)
		}
		if dir == filepath.Dir(dir) {
			break
		}
	}
	fd, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	root := os.NewFile(uintptr(fd), parent)
	defer root.Close()
	base, err := root.Stat()
	if err != nil {
		return err
	}
	if int(base.Sys().(*syscall.Stat_t).Uid) != os.Getuid() {
		return errors.New("输出父目录不属于当前用户")
	}
	if base.Mode().Perm() != 0o755 {
		fmt.Fprintf(os.Stderr, "警告：输出父目录模式%04o，建议0755：%s\n", base.Mode().Perm(), parent)
	}
	var space unix.Statfs_t
	if err := unix.Fstatfs(fd, &space); err != nil {
		return err
	}
	if uint64(space.Bavail)*uint64(space.Bsize) < 50*1024*1024*1024 {
		return errors.New("裁剪归档要求至少50GiB可用空间")
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		return fmt.Errorf("保留已有或不可检查的目标：%s：%v", dest, err)
	}
	out, err := os.CreateTemp(parent, "python-trim-")
	if err != nil {
		return err
	}
	stage, err := out.Stat()
	if err != nil {
		out.Close()
		return err
	}
	defer func() {
		out.Close()
		got, err := os.Lstat(out.Name())
		if err == nil && os.SameFile(stage, got) {
			os.Remove(out.Name())
		}
	}()
	if err := out.Chmod(0o644); err != nil {
		fmt.Fprintf(os.Stderr, "警告：设置0644失败：%s：%v\n", out.Name(), err)
	}
	hash.Reset()
	reader, err := zstd.NewReader(io.TeeReader(in, hash))
	if err != nil {
		return err
	}
	defer reader.Close()
	rep, err := trim(ctx, spec, reader, out)
	if err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	latest, err := in.Stat()
	got, pathErr := os.Stat(name)
	if err != nil || pathErr != nil || !os.SameFile(info, latest) || !os.SameFile(info, got) ||
		latest.Size() != info.Size() || latest.ModTime() != info.ModTime() || fmt.Sprintf("%x", hash.Sum(nil)) != spec.SHA256 {
		return errors.New("读取期间上游输入改变")
	}
	if err := out.Sync(); err != nil {
		return err
	}
	got, err = os.Lstat(parent)
	if err != nil || !os.SameFile(base, got) {
		return errors.New("发布前输出父目录改变")
	}
	got, err = os.Lstat(out.Name())
	if err != nil || !os.SameFile(stage, got) {
		return errors.New("发布前候选归档改变")
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		return err
	}
	hash.Reset()
	bytes, err := io.Copy(hash, out)
	if err != nil {
		return err
	}
	if bytes > 90*1024*1024 {
		return errors.New("裁剪归档超过90MiB分片上限")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := publish(fd, filepath.Base(out.Name()), filepath.Base(dest)); err != nil {
		return err
	}
	if err := root.Sync(); err != nil {
		return err
	}
	fmt.Printf("host=%s profile=%s before=%d after=%d files=%d removed=%d bytes=%d sha256=%x exit=0\n",
		spec.Host, profile, rep.Before, rep.After, rep.Files, rep.Removed, bytes, hash.Sum(nil))
	return nil
}

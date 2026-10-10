package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// workTree判断目录本身是不是Git工作树根。纯复制源码没有Git元数据时返回false，
// 调用方改走目录复制路径，而不是解析git的错误文本。
func workTree(src string) bool {
	top, err := gitRead(src, "rev-parse", "--show-toplevel")
	if err != nil || top == "" {
		return false
	}
	one, err := os.Stat(top)
	if err != nil {
		return false
	}
	two, err := os.Stat(src)
	if err != nil {
		return false
	}
	return os.SameFile(one, two)
}

// plainSelect把没有Git元数据的源码目录按普通文件读成同一套tar条目，
// 后续校验、发布路径与Git来源完全一致。
func plainSelect(src, dest, work string, check bool, paths []string) (returnErr error) {
	tree, err := newTree(work)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, tree.close()) }()
	tree.raw = true
	reader, writer := io.Pipe()
	go func() {
		writer.CloseWithError(walkTar(src, writer))
	}()
	keep := func(name string) bool {
		if name == "." {
			return false
		}
		if len(paths) == 0 {
			return true
		}
		for _, item := range paths {
			if name == item || strings.HasPrefix(name, item+"/") {
				return true
			}
		}
		return false
	}
	if err := tree.readTar(reader, keep); err != nil {
		reader.CloseWithError(err)
		return err
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	if check {
		err = tree.check(dest)
	} else {
		err = tree.emit(dest, "")
	}
	if err != nil {
		return err
	}
	fmt.Printf("git_url=\"\" git_tags=\"\" git_state=no-git\n")
	return nil
}

// walkTar按普通文件把源码目录写成tar流，拒绝多链接文件与特殊条目。符号链接与
// git archive一样写成链接条目，由tarTree拒绝越出源码根的目标并按目标内容发布。
func walkTar(src string, out io.Writer) error {
	writer := tar.NewWriter(out)
	err := filepath.WalkDir(src, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if rel == "." {
				return nil
			}
			return writer.WriteHeader(&tar.Header{
				Name: rel + "/", Typeflag: tar.TypeDir, Mode: int64(info.Mode().Perm()),
			})
		}
		if info.Mode()&os.ModeSymlink != 0 {
			// 只读取链接文本，不跟随宿主路径；权限与git archive的链接条目一致。
			link, err := os.Readlink(name)
			if err != nil {
				return err
			}
			return writer.WriteHeader(&tar.Header{
				Name: rel, Typeflag: tar.TypeSymlink, Linkname: link, Mode: 0o777,
			})
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("plain source tree rejects special entry: %s", name)
		}
		if err := regular(info); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := writer.WriteHeader(&tar.Header{
			Name: rel, Typeflag: tar.TypeReg, Mode: int64(info.Mode().Perm()), Size: info.Size(),
		}); err != nil {
			return err
		}
		file, err := os.Open(name)
		if err != nil {
			return err
		}
		defer file.Close()
		copied, err := io.Copy(writer, file)
		if err != nil {
			return err
		}
		if copied != info.Size() {
			return fmt.Errorf("plain source size changed while reading: %s", name)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return writer.Close()
}

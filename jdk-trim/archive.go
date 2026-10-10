// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type node struct {
	Name string
	Kind byte
	Link string
	Mode int64
	Size int64
	Off  int64
	Data []byte
}

type bundle struct {
	Items   map[string]*node
	Before  int64
	After   int64
	Dropped []string
}

type ctxRead struct {
	ctx context.Context
	in  io.Reader
}

func (r ctxRead) Read(buf []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.in.Read(buf)
}

func cleanName(name string) (string, error) {
	name = strings.TrimSuffix(name, "/")
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || name == "." || name == ".." ||
		strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00\t\r\n") {
		return "", fmt.Errorf("归档路径无效：%q", name)
	}
	return name, nil
}

func scan(ctx context.Context, spec plan, in io.Reader, data *os.File) (*bundle, error) {
	hash := sha256.New()
	raw := io.TeeReader(ctxRead{ctx: ctx, in: in}, hash)
	gr, err := gzip.NewReader(raw)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	b := &bundle{Items: make(map[string]*node)}
	seen := make(map[string]bool)
	var off int64
	for {
		head, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name, err := cleanName(head.Name)
		if err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, fmt.Errorf("归档路径重复：%s", name)
		}
		seen[name] = true
		if name == spec.Prefix || !strings.HasPrefix(name, spec.Prefix+"/") {
			continue
		}
		rel := strings.TrimPrefix(name, spec.Prefix+"/")
		b.Before += head.Size
		if dropped(rel) {
			b.Dropped = append(b.Dropped, rel)
			continue
		}
		item := &node{Name: rel, Kind: head.Typeflag, Link: head.Linkname, Mode: 0o644, Size: head.Size, Off: off}
		if head.Mode&0o111 != 0 {
			item.Mode = 0o755
		}
		switch head.Typeflag {
		case tar.TypeDir:
			item.Mode = 0o755
		case tar.TypeReg, tar.TypeRegA:
			item.Kind = tar.TypeReg
			n, err := io.Copy(data, ctxRead{ctx: ctx, in: tr})
			if err != nil || n != head.Size {
				return nil, fmt.Errorf("读取%s：bytes=%d want=%d err=%v", name, n, head.Size, err)
			}
			off += n
		case tar.TypeSymlink:
			if path.IsAbs(item.Link) || strings.ContainsAny(item.Link, "\\\x00\t\r\n") {
				return nil, fmt.Errorf("链接目标无效：%s", name)
			}
			item.Link = path.Join(path.Dir(rel), item.Link)
		case tar.TypeLink:
			if !strings.HasPrefix(item.Link, spec.Prefix+"/") {
				return nil, fmt.Errorf("硬链接超出JDK根：%s", name)
			}
			item.Link = strings.TrimPrefix(item.Link, spec.Prefix+"/")
		default:
			return nil, fmt.Errorf("不支持的归档类型：%d %s", head.Typeflag, name)
		}
		if item.Kind != tar.TypeReg && item.Size != 0 {
			return nil, fmt.Errorf("非文件条目含有数据：%s", name)
		}
		b.Items[rel] = item
	}
	if _, err := io.Copy(io.Discard, gr); err != nil {
		return nil, err
	}
	if _, err := io.Copy(io.Discard, raw); err != nil {
		return nil, err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != spec.SrcSHA {
		return nil, fmt.Errorf("上游归档摘要不符：got=%s want=%s", got, spec.SrcSHA)
	}
	for _, name := range mustFiles {
		if _, err := b.resolve(name); err != nil {
			return nil, fmt.Errorf("缺少必需JDK输入%s：%w", name, err)
		}
	}
	if b.Items[markName] != nil {
		return nil, errors.New("输入已经包含裁剪标记")
	}
	mark := []byte(fmt.Sprintf("profile=%s\nversion=21.0.12.1+1\nsource_sha256=%s\nremoved=jmods,man,lib/src.zip\n", profile, spec.SrcSHA))
	b.Items[markName] = &node{Name: markName, Kind: tar.TypeReg, Mode: 0o644, Size: int64(len(mark)), Data: mark}
	for name, item := range b.Items {
		for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
			if prior := b.Items[dir]; prior != nil && prior.Kind != tar.TypeDir {
				return nil, fmt.Errorf("父路径不是目录：%s", dir)
			}
			if b.Items[dir] == nil {
				b.Items[dir] = &node{Name: dir, Kind: tar.TypeDir, Mode: 0o755}
			}
		}
		if item.Kind != tar.TypeDir {
			target, err := b.resolve(name)
			if err != nil {
				return nil, err
			}
			b.After += target.Size
		}
	}
	sort.Strings(b.Dropped)
	return b, nil
}

func (b *bundle) resolve(name string) (*node, error) {
	seen := make(map[string]bool)
	for {
		if _, err := cleanName(name); err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, fmt.Errorf("链接形成循环：%s", name)
		}
		seen[name] = true
		item := b.Items[name]
		if item == nil {
			return nil, fmt.Errorf("链接或文件目标缺失：%s", name)
		}
		if item.Kind == tar.TypeReg {
			return item, nil
		}
		if item.Kind != tar.TypeSymlink && item.Kind != tar.TypeLink {
			return nil, fmt.Errorf("目标不是普通文件：%s", name)
		}
		name = item.Link
	}
}

func writeArc(ctx context.Context, b *bundle, data *os.File, out io.Writer) error {
	gw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	tw := tar.NewWriter(gw)
	names := []string{""}
	for name := range b.Items {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		head := tar.Header{Name: outPrefix + "/", Typeflag: tar.TypeDir, Mode: 0o755, ModTime: time.Unix(0, 0), Format: tar.FormatPAX}
		var target *node
		if name != "" {
			item := b.Items[name]
			head.Name, head.Typeflag, head.Mode = outPrefix+"/"+name, item.Kind, item.Mode
			if item.Kind != tar.TypeDir {
				target, err = b.resolve(name)
				if err != nil {
					return err
				}
				head.Typeflag, head.Size, head.Mode = tar.TypeReg, target.Size, target.Mode
			}
		}
		if err := tw.WriteHeader(&head); err != nil {
			return err
		}
		if target == nil {
			continue
		}
		var in io.Reader = io.NewSectionReader(data, target.Off, target.Size)
		if target.Data != nil {
			in = strings.NewReader(string(target.Data))
		}
		if n, err := io.Copy(tw, ctxRead{ctx: ctx, in: in}); err != nil || n != target.Size {
			return fmt.Errorf("写入%s：bytes=%d want=%d err=%v", name, n, target.Size, err)
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gw.Close()
}

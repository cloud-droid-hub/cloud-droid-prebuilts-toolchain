// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"archive/tar"
	"fmt"
	"path"
	"sort"
	"strings"
)

func (b *bundle) copyName(rel string) (string, error) {
	if rel == "" || rel == "." || path.Clean(rel) != rel || path.IsAbs(rel) {
		return "", fmt.Errorf("invalid copy path: %q", rel)
	}
	return arcName(b.prefix+"/"+rel, b.prefix)
}

func (b *bundle) addCopies(copies []string) error {
	added := make(map[string]*entry)
	for _, spec := range copies {
		dst, src, ok := strings.Cut(spec, "=")
		if !ok {
			return fmt.Errorf("copy must use dest=source: %q", spec)
		}
		dst, err := b.copyName(dst)
		if err != nil {
			return err
		}
		src, err = b.copyName(src)
		if err != nil {
			return err
		}
		if b.items[dst] != nil || added[dst] != nil {
			return fmt.Errorf("copy destination already exists: %s", dst)
		}
		for dir := path.Dir(dst); inPrefix(dir, b.prefix); dir = path.Dir(dir) {
			parent := b.items[dir]
			if parent == nil || parent.head.Typeflag != tar.TypeDir {
				return fmt.Errorf("copy parent must be an existing directory: %s", dir)
			}
		}
		for _, name := range b.names {
			if strings.HasPrefix(name, dst+"/") {
				return fmt.Errorf("copy destination contains archive entries: %s", dst)
			}
		}
		target, err := b.resolve(src)
		if err != nil {
			return err
		}
		item := *target
		item.head.Name = dst
		item.head.Linkname = ""
		item.from = target.head.Name
		added[dst] = &item
	}
	for name, item := range added {
		b.items[name] = item
		b.names = append(b.names, name)
	}
	sort.Strings(b.names)
	return nil
}

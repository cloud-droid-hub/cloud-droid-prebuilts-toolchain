package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

func sourceData(src, dest, work, sum string) (returnErr error) {
	check, err := hex.DecodeString(sum)
	if err != nil || len(check) != sha256.Size {
		return fmt.Errorf("source requires a registered --sha256")
	}
	file, err := openData(src)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if string(hash.Sum(nil)) != string(check) {
		return fmt.Errorf("source archive SHA-256 mismatch")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer reader.Close()
	tree, err := newTree(work)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, tree.close()) }()
	if err := tree.readTar(reader, func(name string) bool { return name != "." }); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	return tree.emit(dest, "")
}

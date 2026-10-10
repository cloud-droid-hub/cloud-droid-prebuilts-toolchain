package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceData(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	zip := gzip.NewWriter(&buf)
	data := sampleTar(t, []sample{{"source/COPYING", tar.TypeReg, "", "license", 0o644}})
	if _, err := zip.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zip.Close(); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "source.tar.gz")
	if err := os.WriteFile(src, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	dest := filepath.Join(root, "good")
	if err := sourceData(src, dest, root, sum); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "source", "COPYING"))
	if err != nil || string(got) != "license" {
		t.Fatalf("source data: got %q error=%v", got, err)
	}
	bad := filepath.Join(root, "bad")
	if err := sourceData(src, bad, root, fmt.Sprintf("%064x", 0)); err == nil {
		t.Fatal("incorrect source SHA-256 was accepted")
	}
	if _, err := os.Lstat(bad); !os.IsNotExist(err) {
		t.Fatal("failed source verification created its destination")
	}
}

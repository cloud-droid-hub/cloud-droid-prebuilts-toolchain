// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func debData(raw *os.File) (io.Reader, string, error) {
	var magic [8]byte
	if _, err := io.ReadFull(raw, magic[:]); err != nil || !bytes.Equal(magic[:], []byte("!<arch>\n")) {
		return nil, "", fmt.Errorf("DEB不是ar归档：%v", err)
	}
	for {
		var head [60]byte
		if _, err := io.ReadFull(raw, head[:]); err != nil {
			return nil, "", err
		}
		if string(head[58:60]) != "`\n" {
			return nil, "", errors.New("DEB成员头无效")
		}
		name := strings.TrimSuffix(strings.TrimSpace(string(head[:16])), "/")
		size, err := strconv.ParseInt(strings.TrimSpace(string(head[48:58])), 10, 64)
		if err != nil || size < 0 {
			return nil, "", errors.New("DEB成员长度无效")
		}
		if strings.HasPrefix(name, "data.tar") {
			return io.LimitReader(raw, size), strings.TrimPrefix(name, "data."), nil
		}
		if _, err := raw.Seek(size+size%2, io.SeekCurrent); err != nil {
			return nil, "", err
		}
	}
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"bytes"
	"testing"
)

func TestCLIRejects(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

// Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)

package main

import (
	"fmt"
	"io"
	"time"
)

type logSink struct {
	out io.Writer
	err error
}

func (w *logSink) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.out.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}

func writeLog(out io.Writer, format string, args ...any) error {
	w := logSink{out: out}
	if _, err := fmt.Fprintf(&w, format, args...); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func finish(out, fail io.Writer, start time.Time, err error) int {
	if err == nil {
		err = writeLog(out, "exit=0 elapsed=%s\n", time.Since(start))
		if err == nil {
			return 0
		}
	}
	if err := writeLog(fail, "exit=1 error=%v elapsed=%s\n", err, time.Since(start)); err != nil {
		return 1
	}
	return 1
}

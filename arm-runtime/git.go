package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

type gitState struct {
	rev  string
	url  string
	tags string
}

func gitRead(src string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", src}, args...)...)
	cmd.Env = gitEnv()
	data, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Git metadata command failed: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func gitEnv() []string {
	var env []string
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "GIT_") {
			env = append(env, item)
		}
	}
	return append(env, "GIT_NO_LAZY_FETCH=1", "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=", "GIT_OPTIONAL_LOCKS=0")
}

func gitClean(src string) (gitState, error) {
	var state gitState
	status, err := gitRead(src, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return state, fmt.Errorf("source Git working tree is unreadable: %s", src)
	}
	// 未提交改动不再拒绝构建，但归档读的是已提交版本，来源差异必须明确提示。
	if status != "" {
		fmt.Fprintf(os.Stderr, "==> source Git working tree has uncommitted changes, archiving the committed revision: %s\n", src)
	}
	index, err := gitRead(src, "ls-files", "-v", "-z")
	if err != nil {
		return state, err
	}
	for _, item := range strings.Split(index, "\x00") {
		if item != "" && (item[0] == 'S' || (item[0] >= 'a' && item[0] <= 'z')) {
			return state, fmt.Errorf("source Git index contains hidden file flags")
		}
	}
	state.rev, err = gitRead(src, "rev-parse", "--verify", "@^{commit}")
	if err != nil {
		return state, err
	}
	state.tags, err = gitRead(src, "tag", "--points-at", state.rev)
	if err != nil {
		return state, err
	}
	state.url, _ = gitRead(src, "remote", "get-url", "origin")
	return state, nil
}

func gitData(src, dest, work string) error {
	return gitSelect(src, dest, work, false, nil)
}

func gitSelect(src, dest, work string, check bool, paths []string) (returnErr error) {
	for _, name := range paths {
		if name == "" || name == "." || name == ".." || path.IsAbs(name) || path.Clean(name) != name ||
			strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00\r\n:*?[") {
			return fmt.Errorf("invalid Git source selection: %q", name)
		}
	}
	// 纯复制源码没有Git元数据，按目录内容读取同一套条目，不因缺少Git而失败。
	if !workTree(src) {
		return plainSelect(src, dest, work, check, paths)
	}
	before, err := gitClean(src)
	if err != nil {
		return err
	}
	tree, err := newTree(work)
	if err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, tree.close()) }()
	tree.raw = true
	args := []string{"-C", src, "archive", "--format=tar", before.rev}
	if len(paths) > 0 {
		args = append(append(args, "--"), paths...)
	}
	cmd := exec.Command("git", args...)
	cmd.Env = gitEnv()
	cmd.Stderr = io.Discard
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	err = tree.readTar(reader, func(name string) bool { return name != "." })
	if err != nil {
		cmd.Process.Kill()
	} else {
		_, err = io.Copy(io.Discard, reader)
	}
	wait := cmd.Wait()
	if err != nil {
		return err
	}
	if wait != nil {
		return fmt.Errorf("Git archive command failed: %w", wait)
	}
	after, err := gitClean(src)
	if err != nil || before != after {
		return fmt.Errorf("source Git state changed while reading the archive")
	}
	if check {
		err = tree.check(dest)
	} else {
		err = tree.emit(dest, "")
	}
	if err != nil {
		return err
	}
	fmt.Printf("git_url=%q git_tags=%q git_state=clean\n", before.url, before.tags)
	return nil
}

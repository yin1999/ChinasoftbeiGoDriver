package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
)

const (
	execGo = "/usr/bin/go"
	execPy = "/usr/local/bin/pipreqs"
)

type execErr struct {
	cmd    string
	path   string
	errMsg string
}

func builder(p programInfo) error {
	switch p.file {
	case python2:
		return py2Builder(p.dir)
	case python3:
		return py3Builder(p.dir)
	case golang:
		return goBuilder(p.dir)
	default:
		return errTypeErr
	}
}

func py2Builder(dir string) error {
	cmd := exec.Cmd{
		Path: execPy,
		Args: []string{"./"},
		Dir:  dir,
	}
	return runCmd(&cmd)
}

func py3Builder(dir string) error {
	cmd := exec.Cmd{
		Path: execPy,
		Args: []string{"./"},
		Dir:  dir,
	}
	return runCmd(&cmd)
}

func goBuilder(dir string) error {
	// create go.mod
	cmd := exec.Cmd{
		Path: execGo,
		Args: []string{"mod", "init", "main"},
		Dir:  dir,
	}
	err := runCmd(&cmd)
	if err != nil {
		return err
	}

	// build
	cmd.Args = []string{"build", "-ldflags", "-s -w"}
	err = runCmd(&cmd)
	return err
}

func (t execErr) Error() string {
	return fmt.Sprintf("cmd: %s, path: %s, errMsg: %s", t.cmd, t.path, t.errMsg)
}

func runCmd(cmd *exec.Cmd) error {
	out, err := cmd.StderrPipe()
	if err = cmd.Start(); err != nil {
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	buf, err := ioutil.ReadAll(out)
	var cmdStr string
	if len(cmd.Args) == 0 {
		cmdStr = cmd.Path
	} else {
		cmdStr = cmd.Path + " " + cmd.Args[0]
	}
	if len(buf) != 0 {
		return execErr{
			cmd:    cmdStr,
			path:   cmd.Dir,
			errMsg: string(buf),
		}
	}
	return err
}
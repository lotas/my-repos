package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

type GitCallback func(path string, d fs.DirEntry)

var git string

func init() {
	var err error
	git, err = exec.LookPath("git")
	if err != nil {
		panic(err)
	}
}

func gitCommand(path string, args ...string) {
	fullArgs := append([]string{"--git-dir", path}, args...)

	cmd := exec.Command(git, fullArgs...)

	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf("%s Cannot process:\n%s\n", path, stdoutStderr)
	}

	fmt.Printf("%s All good:\n%s\n", path, stdoutStderr)
}

func gitStatus(path string, d fs.DirEntry) {
	gitCommand(path, "status")
}

func gitPull(path string, d fs.DirEntry) {
	gitCommand(path, "pull")
}

func gitLog(path string, d fs.DirEntry) {
	gitCommand(
		path,
		"log",
		"--author", "user1@email.com",
		"--since", "2023-01-01",
	)
}

func walk(s string, d fs.DirEntry, err error, callback GitCallback) error {
	if err != nil {
		return err
	}

	if d.IsDir() && d.Name() == ".git" {
		callback(s, d)
	}

	return nil
}

func main() {
	var dir string
	var callback GitCallback = gitStatus

	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}

	if len(os.Args) > 2 {
		switch os.Args[2] {
		case "log":
			callback = gitLog
		case "status":
		default:
			callback = gitStatus
		}
	}

	fmt.Printf("Scanning from %s\n", dir)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		return walk(path, d, err, callback)
	})
}

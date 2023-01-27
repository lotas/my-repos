package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var git string

func init() {
	var err error
	git, err = exec.LookPath("git")
	if err != nil {
		panic(err)
	}
}

func gitCheck(d fs.DirEntry) {
	o := new(strings.Builder)
	e := new(strings.Builder)

	cmd := exec.Command(git, "--git-dir", d.Name(), "status")
	cmd.Stdout = o
	cmd.Stderr = e

	fmt.Printf("Running")
	cmd.Run()
	fmt.Printf("stdout=%v\nstdin=%v\n", o, e)
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if d.IsDir() && d.Name() == ".git" {
		fmt.Printf("s=%s d=%v\n", s, d)
		gitCheck(d)
	}

	return nil
}

func main() {
	var dir string

	if len(os.Args) == 2 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}

	fmt.Printf("Scanning from %s\n", dir)
	filepath.WalkDir(dir, walk)
}

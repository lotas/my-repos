package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type GitCallback func(path string, d fs.DirEntry)

var git string
var totalWalked int
var totalMatched int

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

	fmt.Printf("%s %s All good:\n%s\n", args[0], path, stdoutStderr)
}

func gitStatus(path string, d fs.DirEntry) { gitCommand(path, "status") }
func gitPull(path string, d fs.DirEntry)   { gitCommand(path, "pull") }
func gitFetch(path string, d fs.DirEntry)  { gitCommand(path, "fetch") }

func gitLog(path string, d fs.DirEntry) {
	gitCommand(
		path,
		"log",
		"--oneline",
		"--author", "user1@email.com",
		"--author", "user2@email.com",
		"--author", "user3@email.com",
		"--since", "2023-01-01",
	)
}

func nop(path string, d fs.DirEntry) { fmt.Printf("NOP: %s\n", path) }

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
		case "pull":
			callback = gitPull
		case "fetch":
			callback = gitFetch
		case "nop":
			callback = nop
		case "status":
		default:
			callback = gitStatus
		}
	}

	var wg sync.WaitGroup

	var visitedMap map[string]bool = make(map[string]bool)

	fmt.Printf("Scanning from %s\n", dir)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		baseDir := filepath.Dir(path)

		// skip going into the folders that already had .git in it
		if visitedMap[baseDir] {
			return filepath.SkipDir
		}

		totalWalked++
		if d.Name() == ".git" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				callback(path, d)
			}()
			totalMatched++

			visitedMap[baseDir] = true

			return filepath.SkipDir
		}

		return nil
	})

	wg.Wait()

	fmt.Printf("Scanned folders: %d, processed: %d", totalWalked, totalMatched)
}

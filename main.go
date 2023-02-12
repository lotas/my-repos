package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type GitCallback func(path string, d fs.DirEntry) (string, error)
type AfterCallback func()

type SafeSummary struct {
	sync.RWMutex
	summaries map[string]string
}

func (ss *SafeSummary) Add(k, v string) {
	ss.Lock()
	defer ss.Unlock()
	ss.summaries[k] = v
}

var summaries = &SafeSummary{
	summaries: map[string]string{},
}

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

func gitCommand(path string, args ...string) (string, error) {
	fullArgs := append([]string{"--git-dir", path}, args...)

	cmd := exec.Command(git, fullArgs...)

	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

func diskUsage(path string) string {
	cmd := exec.Command("du", "-hs", path)
	stdoutStderr, _ := cmd.CombinedOutput()
	return string(stdoutStderr)
}

func gitStatus(path string, d fs.DirEntry) (string, error) { return gitCommand(path, "status") }
func gitPull(path string, d fs.DirEntry) (string, error)   { return gitCommand(path, "pull") }
func gitFetch(path string, d fs.DirEntry) (string, error)  { return gitCommand(path, "fetch") }

func gitLog(path string, d fs.DirEntry) (string, error) {
	return gitCommand(
		path,
		"log",
		"--oneline",
		"--author", "user1@email.com",
		"--since", "2023-01-01",
	)
}

func gitNop(path string, d fs.DirEntry) (string, error) { return fmt.Sprintf("NOP: %s", path), nil }

func nop() {}

func firstLine(s string) string {
	return strings.Split(s, "\n")[0]
}

func gitSummary(path string, d fs.DirEntry) (string, error) {
	baseDir := filepath.Dir(path)
	info := ""

	branch, err := gitCommand(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info += fmt.Sprintf("Branch: %s", firstLine(branch))
	}

	remotes, err := gitCommand(path, "remote", "-v")
	if err == nil {
		info += fmt.Sprintf("\nremote: %s ", firstLine(remotes))
	}

	du := diskUsage(path)
	info += fmt.Sprintf("\nSize: %s", strings.Split(du, "\t")[0])

	summaries.Add(baseDir, info)

	return "", nil
}
func printSummary() {
	fmt.Println("")
	for k, v := range summaries.summaries {
		fmt.Printf("%s:\n%s\n\n", k, v)
	}
}

func showHelp() {
	fmt.Printf(`
Usage: my-repos <path> <cmd>
Example: my-repos ~/dev fetch

cmd is one of the existing git commands:
	log
	pull
	fetch
	status

Or one of the extra commands:
	summary - show how many repos there are
`)
	os.Exit(0)
}

func scan(dir string, callback GitCallback) {
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
				out, err2 := callback(path, d)
				if out != "" || err2 != nil {
					fmt.Printf("%s: %v %s\n", baseDir, err2, out)
				}
			}()
			totalMatched++

			visitedMap[baseDir] = true

			return filepath.SkipDir
		}

		return nil
	})

	wg.Wait()
}

func main() {
	var dir string
	var callback GitCallback = gitStatus
	var afterCallback AfterCallback = nop

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
			callback = gitNop
		case "status":
			callback = gitStatus
		case "summary":
			callback = gitSummary
			afterCallback = printSummary
		default:
			showHelp()
		}
	} else {
		showHelp()
	}

	scan(dir, callback)

	fmt.Printf("Scanned folders: %d, git repos: %d\n", totalWalked, totalMatched)
	afterCallback()
}

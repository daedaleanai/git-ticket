// +build ignore

package main

import (
	"os"

	rb "github.com/daedaleanai/git-ticket/misc/random_bugs"
	"github.com/daedaleanai/git-ticket/repository"
)

// This program will randomly generate a collection of bugs in the repository
// of the current path
func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	repo, err := repository.NewGitRepo(dir)
	if err != nil {
		panic(err)
	}

	rb.CommitRandomBugs(repo, rb.DefaultOptions())
}

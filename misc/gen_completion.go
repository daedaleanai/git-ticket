package main

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/commands"
)

func main() {
	fmt.Println("Generating completion files ...")

	root := commands.NewRootCommand()

	tasks := map[string]func(*cobra.Command) error{
		"Bash":       genBash,
		"Fish":       genFish,
		"PowerShell": genPowerShell,
		"ZSH":        genZsh,
	}

	var wg sync.WaitGroup
	for name, f := range tasks {
		wg.Add(1)
		go func(name string, f func(*cobra.Command) error) {
			defer wg.Done()
			err := f(root)
			if err != nil {
				fmt.Printf("  - %s: %v\n", name, err)
				return
			}
			fmt.Printf("  - %s: ok\n", name)
		}(name, f)
	}

	wg.Wait()
}

func genBash(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "misc", "bash_completion")
	_ = os.Mkdir(dir, os.ModePerm)
	return root.GenBashCompletionFile(path.Join(dir, "git-ticket"))
}

func genFish(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "misc", "fish_completion")
	_ = os.Mkdir(dir, os.ModePerm)
	return root.GenFishCompletionFile(path.Join(dir, "git-ticket"), true)
}

func genPowerShell(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "misc", "powershell_completion")
	_ = os.Mkdir(dir, os.ModePerm)
	return root.GenPowerShellCompletionFile(path.Join(dir, "git-ticket"))
}

func genZsh(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "misc", "zsh_completion")
	_ = os.Mkdir(dir, os.ModePerm)
	return root.GenZshCompletionFile(path.Join(dir, "git-ticket"))
}

package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/interrupt"
)

// Env is the environment of a command
type Env struct {
	repo    repository.ClockedRepo
	backend *cache.RepoCache
	out     out
	err     out
}

func newEnv() *Env {
	return &Env{
		repo: nil,
		out:  out{WriteCloser: os.Stdout},
		err:  out{WriteCloser: os.Stderr},
	}
}

type out struct {
	io.WriteCloser
}

func (o out) Printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(o, format, a...)
}

func (o out) Print(a ...interface{}) {
	_, _ = fmt.Fprint(o, a...)
}

func (o out) Println(a ...interface{}) {
	_, _ = fmt.Fprintln(o, a...)
}

// loadRepo is a pre-run function that load the repository for use in a command
func loadRepo(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to get the current working directory: %q", err)
		}

		env.repo, err = repository.NewGitRepo(cwd, []repository.ClockLoader{bug.ClockLoader, identity.ClockLoader})
		if err == repository.ErrNotARepo {
			return fmt.Errorf("%s must be run from within a git repo", rootCommandName)
		}

		if err != nil {
			return err
		}

		return nil
	}
}

// loadRepoEnsureUser is the same as loadRepo, but also ensure that the user has configured
// an identity. Use this pre-run function when an error after using the configured user won't
// do.
func loadRepoEnsureUser(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := loadRepo(env)(cmd, args)
		if err != nil {
			return err
		}

		_, err = identity.GetUserIdentity(env.repo)
		if err != nil {
			return err
		}

		return nil
	}
}

// loadBackend is a pre-run function that load the repository and the backend for use in a command
// When using this function you also need to use closeBackend as a post-run
func loadBackend(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := loadRepo(env)(cmd, args)
		if err != nil {
			return err
		}

		env.backend, err = cache.NewRepoCache(env.repo, RebuildCache)
		if err != nil {
			return err
		}

		cleaner := func(env *Env) interrupt.CleanerFunc {
			return func() error {
				if env.backend != nil {
					err := env.backend.Close()
					env.backend = nil
					return err
				}
				return nil
			}
		}

		// Cleanup properly on interrupt
		interrupt.RegisterCleaner(cleaner(env))
		return nil
	}
}

// loadBackendEnsureUser is the same as loadBackend, but also ensure that the user has configured
// an identity. Use this pre-run function when an error after using the configured user won't
// do.
func loadBackendEnsureUser(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := loadBackend(env)(cmd, args)
		if err != nil {
			return err
		}

		_, err = identity.GetUserIdentity(env.repo)
		if err != nil {
			return err
		}

		return nil
	}
}

// closeBackend is a post-run function that will close the backend properly
// if it has been opened.
func closeBackend(env *Env) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if env.backend == nil {
			return nil
		}
		err := env.backend.Close()
		env.backend = nil
		return err
	}
}

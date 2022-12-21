// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
	"sync"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"

	"github.com/daedaleanai/git-ticket/util/lamport"
)

const (
	clockPath = "git-bug"
)

var _ ClockedRepo = &GitRepo{}
var _ TestedRepo = &GitRepo{}

// GitRepo represents an instance of a (local) git repository.
type GitRepo struct {
	path string

	clocksMutex sync.Mutex
	clocks      map[string]lamport.Clock

	// memoized go-git repo representing the same repository,
	// for reading commits.
	repo *goGit.Repository
}

// LocalConfig give access to the repository scoped configuration
func (repo *GitRepo) LocalConfig() Config {
	return newGitConfig(repo, false)
}

// GlobalConfig give access to the git global configuration
func (repo *GitRepo) GlobalConfig() Config {
	return newGitConfig(repo, true)
}

// Run the given git command with the given I/O reader/writers, returning an error if it fails.
func (repo *GitRepo) runGitCommandWithIO(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	// make sure that the working directory for the command
	// always exist, in particular when running "git init".
	repopath := repo.path
	if path.Base(repopath) == ".git" {
		repopath = strings.TrimSuffix(repopath, ".git")
	}

	// fmt.Printf("[%s] Running git %s\n", path, strings.Join(args, " "))

	cmd := exec.Command("git", args...)
	cmd.Dir = repopath
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

// Run the given git command and return its stdout, or an error if the command fails.
func (repo *GitRepo) runGitCommandRaw(stdin io.Reader, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := repo.runGitCommandWithIO(stdin, &stdout, &stderr, args...)
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// Run the given git command and return its stdout, or an error if the command fails.
func (repo *GitRepo) runGitCommandWithStdin(stdin io.Reader, args ...string) (string, error) {
	stdout, stderr, err := repo.runGitCommandRaw(stdin, args...)
	if err != nil {
		if stderr == "" {
			stderr = "Error running git command: " + strings.Join(args, " ")
		}
		err = fmt.Errorf(stderr)
	}
	return stdout, err
}

// Run the given git command and return its stdout, or an error if the command fails.
func (repo *GitRepo) runGitCommand(args ...string) (string, error) {
	return repo.runGitCommandWithStdin(nil, args...)
}

// NewGitRepo determines if the given working directory is inside of a git repository,
// and returns the corresponding GitRepo instance if it is.
func NewGitRepo(path string, clockLoaders []ClockLoader) (*GitRepo, error) {
	repo := &GitRepo{
		path:   path,
		clocks: make(map[string]lamport.Clock),
	}

	// Check the repo and retrieve the root path
	stdout, err := repo.runGitCommand("rev-parse", "--absolute-git-dir")

	// Now dir is fetched with "git rev-parse --git-dir". May be it can
	// still return nothing in some cases. Then empty stdout check is
	// kept.
	if err != nil || stdout == "" {
		return nil, ErrNotARepo
	}

	// Fix the path to be sure we are at the root
	repo.path = stdout

	for _, loader := range clockLoaders {
		allExist := true
		for _, name := range loader.Clocks {
			if _, err := repo.getClock(name); err != nil {
				allExist = false
			}
		}

		if !allExist {
			err = loader.Witnesser(repo)
			if err != nil {
				return nil, err
			}
		}
	}

	return setupGitRepo(repo)
}

// NewGitRepoNoInit returns a GitRepo instance for the given directory, does not
// attempt to initialise clocks if not already done
func NewGitRepoNoInit(path string) (*GitRepo, error) {
	repo := &GitRepo{
		path:   path,
		clocks: make(map[string]lamport.Clock),
	}

	// Check the repo and retrieve the root path
	stdout, err := repo.runGitCommand("rev-parse", "--absolute-git-dir")

	// Now dir is fetched with "git rev-parse --git-dir". May be it can
	// still return nothing in some cases. Then empty stdout check is
	// kept.
	if err != nil || stdout == "" {
		return nil, ErrNotARepo
	}

	// Fix the path to be sure we are at the root
	repo.path = stdout

	return setupGitRepo(repo)
}

// InitGitRepo create a new empty git repo at the given path
func InitGitRepo(path string) (*GitRepo, error) {
	repo := &GitRepo{
		path:   path + "/.git",
		clocks: make(map[string]lamport.Clock),
	}

	_, err := repo.runGitCommand("init", path)
	if err != nil {
		return nil, err
	}

	return setupGitRepo(repo)
}

// InitBareGitRepo create a new --bare empty git repo at the given path
func InitBareGitRepo(path string) (*GitRepo, error) {
	repo := &GitRepo{
		path:   path,
		clocks: make(map[string]lamport.Clock),
	}

	_, err := repo.runGitCommand("init", "--bare", path)
	if err != nil {
		return nil, err
	}

	return setupGitRepo(repo)
}

func setupGitRepo(repo *GitRepo) (*GitRepo, error) {
	var err error

	repo.repo, err = goGit.PlainOpen(repo.path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// GetPath returns the path to the repo.
func (repo *GitRepo) GetPath() string {
	return repo.path
}

// GetUserName returns the name the the user has used to configure git
func (repo *GitRepo) GetUserName() (string, error) {
	return repo.runGitCommand("config", "user.name")
}

// GetUserEmail returns the email address that the user has used to configure git.
func (repo *GitRepo) GetUserEmail() (string, error) {
	return repo.runGitCommand("config", "user.email")
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (repo *GitRepo) GetCoreEditor() (string, error) {
	return repo.runGitCommand("var", "GIT_EDITOR")
}

// GetRemotes returns the configured remotes repositories.
func (repo *GitRepo) GetRemotes() (map[string]string, error) {
	stdout, err := repo.runGitCommand("remote", "--verbose")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(stdout, "\n")
	remotes := make(map[string]string, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		elements := strings.Fields(line)
		if len(elements) != 3 {
			return nil, fmt.Errorf("git remote: unexpected output format: %s", line)
		}

		remotes[elements[0]] = elements[1]
	}

	return remotes, nil
}

// FetchRefs fetch git refs from a remote
func (repo *GitRepo) FetchRefs(remote, refSpec string) (string, error) {
	stdout, err := repo.runGitCommand("fetch", remote, refSpec)

	if err != nil {
		return stdout, fmt.Errorf("failed to fetch from the remote '%s': %v", remote, err)
	}

	return stdout, err
}

// PushRefs push git refs to a remote
func (repo *GitRepo) PushRefs(remote string, refSpec string) (string, error) {
	stdout, stderr, err := repo.runGitCommandRaw(nil, "push", remote, refSpec)

	if err != nil {
		return stdout + stderr, fmt.Errorf("failed to push to the remote '%s': %v", remote, stderr)
	}
	return stdout + stderr, nil
}

// StoreData will store arbitrary data and return the corresponding hash
func (repo *GitRepo) StoreData(data []byte) (Hash, error) {
	var stdin = bytes.NewReader(data)

	stdout, err := repo.runGitCommandWithStdin(stdin, "hash-object", "--stdin", "-w")

	return Hash(stdout), err
}

// ReadData will attempt to read arbitrary data from the given hash
func (repo *GitRepo) ReadData(hash Hash) ([]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := repo.runGitCommandWithIO(nil, &stdout, &stderr, "cat-file", "-p", string(hash))

	if err != nil {
		return []byte{}, err
	}

	return stdout.Bytes(), nil
}

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GitRepo) StoreTree(entries []TreeEntry) (Hash, error) {
	buffer := prepareTreeEntries(entries)

	stdout, err := repo.runGitCommandWithStdin(&buffer, "mktree")

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// StoreCommit will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommit(treeHash Hash) (Hash, error) {
	return repo.storeCommitRaw(treeHash)
}

// StoreCommitWithParent will store a Git commit with the given Git tree
func (repo *GitRepo) StoreCommitWithParent(treeHash Hash, parent Hash) (Hash, error) {
	return repo.storeCommitRaw(treeHash, "-p", string(parent))
}

func (repo *GitRepo) storeCommitRaw(treeHash Hash, extraArgs ...string) (Hash, error) {
	args := []string{"commit-tree"}

	// `git commit-tree` uses user.signingkey and gpg.program, but not commit.gpgsign.
	// We read commit.gpgsign ourselves and simply pass -S to `git commit-tree`.
	config := repo.LocalConfig()
	gpgsign, err := config.ReadBool("commit.gpgsign")
	if err != nil && err != ErrNoConfigEntry {
		// There are more than one entries, or some other error.
		return "", errors.Wrap(err, "failed to read local commit.gpgsign")
	}

	if !gpgsign {
		return "", fmt.Errorf("Signing is disabled in your git configuration but is mandatory in git-ticket workflow")
	}

	args = append(args, "-S")

	args = append(args, extraArgs...)

	args = append(args, string(treeHash))

	stdout, err := repo.runGitCommand(args...)

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// UpdateRef will create or update a Git reference
func (repo *GitRepo) UpdateRef(ref string, hash Hash) error {
	_, err := repo.runGitCommand("update-ref", ref, string(hash))

	return err
}

// RemoveRef will remove a Git reference
func (repo *GitRepo) RemoveRef(ref string) error {
	_, err := repo.runGitCommand("update-ref", "-d", ref)

	return err
}

// ListRefs will return a list of Git ref matching the given refspec
func (repo *GitRepo) ListRefs(refspec string) ([]string, error) {
	stdout, err := repo.runGitCommand("for-each-ref", "--format=%(refname)", refspec)

	if err != nil {
		return nil, err
	}

	split := strings.Split(stdout, "\n")

	if len(split) == 1 && split[0] == "" {
		return []string{}, nil
	}

	return split, nil
}

// RefExist will check if a reference exist in Git
func (repo *GitRepo) RefExist(ref string) (bool, error) {
	stdout, err := repo.runGitCommand("for-each-ref", ref)

	if err != nil {
		return false, err
	}

	return stdout != "", nil
}

// CopyRef will create a new reference with the same value as another one
func (repo *GitRepo) CopyRef(source string, dest string) error {
	_, err := repo.runGitCommand("update-ref", dest, source)

	return err
}

// Resolve the reference to the commit hash it represents
func (repo *GitRepo) ResolveRef(ref string) (Hash, error) {
	stdout, err := repo.runGitCommand("show-ref", "-s", ref)
	return Hash(stdout), err
}

// ListCommits will return the list of commit hashes of a ref, in chronological order
func (repo *GitRepo) ListCommits(ref string) ([]Hash, error) {
	stdout, err := repo.runGitCommand("rev-list", "--first-parent", "--reverse", ref)

	if err != nil {
		return nil, err
	}

	split := strings.Split(stdout, "\n")

	casted := make([]Hash, len(split))
	for i, line := range split {
		casted[i] = Hash(line)
	}

	return casted, nil

}

// CommitsBetween will return the commits reachable from 'mainRef' which are not reachable from 'excludeRef'
func (repo *GitRepo) CommitsBetween(excludeRef, mainRef string) ([]Hash, error) {
	stdout, err := repo.runGitCommand("rev-list", "^"+excludeRef, mainRef)
	if err != nil {
		return nil, err
	}
	if stdout == "" {
		// Return a nil slice if no commits are between the two references
		return nil, nil
	}

	split := strings.Split(stdout, "\n")

	casted := make([]Hash, len(split))
	for i, line := range split {
		casted[i] = Hash(line)
	}

	return casted, nil
}

// LastCommit will return the latest commit hash of a ref
func (repo *GitRepo) LastCommit(ref string) (Hash, error) {
	stdout, err := repo.runGitCommand("rev-list", "-1", ref)
	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// ReadTree will return the list of entries in a Git tree
func (repo *GitRepo) ReadTree(hash Hash) ([]TreeEntry, error) {
	stdout, err := repo.runGitCommand("ls-tree", string(hash))

	if err != nil {
		return nil, err
	}

	return readTreeEntries(stdout)
}

// FindCommonAncestor will return the last common ancestor of two chain of commit
func (repo *GitRepo) FindCommonAncestor(hash1 Hash, hash2 Hash) (Hash, error) {
	stdout, err := repo.runGitCommand("merge-base", string(hash1), string(hash2))

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

// GetTreeHash return the git tree hash referenced in a commit
func (repo *GitRepo) GetTreeHash(commit Hash) (Hash, error) {
	stdout, err := repo.runGitCommand("rev-parse", string(commit)+"^{tree}")

	if err != nil {
		return "", err
	}

	return Hash(stdout), nil
}

func (repo *GitRepo) CommitObject(h plumbing.Hash) (*object.Commit, error) {
	return repo.repo.CommitObject(h)
}

func (repo *GitRepo) ResolveRevision(rev plumbing.Revision) (*plumbing.Hash, error) {
	return repo.repo.ResolveRevision(rev)
}

// GetOrCreateClock return a Lamport clock stored in the Repo.
// If the clock doesn't exist, it's created.
func (repo *GitRepo) GetOrCreateClock(name string) (lamport.Clock, error) {
	c, err := repo.getClock(name)
	if err == nil {
		return c, nil
	}
	if err != ErrClockNotExist {
		return nil, err
	}

	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	p := path.Join(repo.path, clockPath, name+"-clock")

	c, err = lamport.NewPersistedClock(p)
	if err != nil {
		return nil, err
	}

	repo.clocks[name] = c
	return c, nil
}

func (repo *GitRepo) getClock(name string) (lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	if c, ok := repo.clocks[name]; ok {
		return c, nil
	}

	p := path.Join(repo.path, clockPath, name+"-clock")

	c, err := lamport.LoadPersistedClock(p)
	if err == nil {
		repo.clocks[name] = c
		return c, nil
	}
	if err == lamport.ErrClockNotExist {
		return nil, ErrClockNotExist
	}
	return nil, err
}

// AddRemote add a new remote to the repository
// Not in the interface because it's only used for testing
func (repo *GitRepo) AddRemote(name string, url string) error {
	_, err := repo.runGitCommand("remote", "add", name, url)

	return err
}

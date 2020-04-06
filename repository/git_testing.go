package repository

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

// This is intended for testing only

func CreateTestRepo(bare bool) *GitRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("Creating repo:", dir)

	var creator func(string) (*GitRepo, error)

	if bare {
		creator = InitBareGitRepo
	} else {
		creator = InitGitRepo
	}

	repo, err := creator(dir)
	if err != nil {
		log.Fatal(err)
	}

	config := repo.LocalConfig()
	if err := config.StoreString("user.name", "testuser"); err != nil {
		log.Fatal("failed to set user.name for test repository: ", err)
	}
	if err := config.StoreString("user.email", "testuser@example.com"); err != nil {
		log.Fatal("failed to set user.email for test repository: ", err)
	}

	setupSigningKey(config)

	return repo
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func setupSigningKey(config Config) {
	// Generate a key pair for signing commits.
	entity, err := openpgp.NewEntity("First Last", "", "fl@example.org", nil)
	check(err)

	if err := config.StoreString("user.signingkey", entity.PrivateKey.KeyIdString()); err != nil {
		log.Fatal("failed to set user.signingkey for test repository: ", err)
	}

	// Armor the private part.
	privBuilder := &strings.Builder{}
	w, err := armor.Encode(privBuilder, openpgp.PrivateKeyType, nil)
	check(err)
	err = entity.SerializePrivate(w, nil)
	check(err)
	err = w.Close()
	check(err)
	armoredPriv := privBuilder.String()

	// Armor the public part.
	pubBuilder := &strings.Builder{}
	w, err = armor.Encode(pubBuilder, openpgp.PublicKeyType, nil)
	check(err)
	err = entity.Serialize(w)
	check(err)
	err = w.Close()
	check(err)
	armoredPub := pubBuilder.String()

	// Create a custom gpg keyring to be used when creating commits with `git`.
	keyring, err := ioutil.TempFile("", "keyring")
	check(err)

	// Import the armored private key to the custom keyring.
	priv, err := ioutil.TempFile("", "privkey")
	check(err)
	_, err = fmt.Fprintf(priv, armoredPriv)
	err = priv.Close()
	check(err)
	err = exec.Command("gpg", "--no-default-keyring", "--keyring", keyring.Name(), "--import", priv.Name()).Run()
	check(err)

	// Import the armored public key to the custom keyring.
	pub, err := ioutil.TempFile("", "pubkey")
	check(err)
	_, err = fmt.Fprintf(pub, armoredPub)
	err = pub.Close()
	check(err)
	err = exec.Command("gpg", "--no-default-keyring", "--keyring", keyring.Name(), "--import", pub.Name()).Run()
	check(err)

	// Use a gpg wrapper to use a custom keyring containing GPGKeyID.
	gpgWrapper := createGPGWrapper(keyring.Name())
	if err := config.StoreString("gpg.program", gpgWrapper); err != nil {
		log.Fatal("failed to set gpg.program for test repository: ", err)
	}

	if err := config.StoreString("commit.gpgsign", "true"); err != nil {
		log.Fatal("failed to set commit.gpgsign for test repository: ", err)
	}
}

// createGPGWrapper creates a shell script running gpg with a specific keyring.
func createGPGWrapper(keyringPath string) string {
	file, err := ioutil.TempFile("", "gpgwrapper")
	check(err)

	_, err = fmt.Fprintf(file, `#!/bin/sh
exec gpg --no-default-keyring --keyring="%s" "$@"
`, keyringPath)
	check(err)

	err = file.Close()
	check(err)

	err = os.Chmod(file.Name(), os.FileMode(0700))
	check(err)

	return file.Name()
}

func CleanupTestRepos(t testing.TB, repos ...Repo) {
	var firstErr error
	for _, repo := range repos {
		path := repo.GetPath()
		if strings.HasSuffix(path, "/.git") {
			// for a normal repository (not --bare), we want to remove everything
			// including the parent directory where files are checked out
			path = strings.TrimSuffix(path, "/.git")

			// Testing non-bare repo should also check path is
			// only .git (i.e. ./.git), but doing so, we should
			// try to remove the current directory and hav some
			// trouble. In the present case, this case should not
			// occur.
			// TODO consider warning or error when path == ".git"
		}
		// fmt.Println("Cleaning repo:", path)
		err := os.RemoveAll(path)
		if err != nil {
			log.Println(err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if firstErr != nil {
		t.Fatal(firstErr)
	}
}

func SetupReposAndRemote(t testing.TB) (repoA, repoB, remote *GitRepo) {
	repoA = CreateTestRepo(false)
	repoB = CreateTestRepo(false)
	remote = CreateTestRepo(true)

	remoteAddr := "file://" + remote.GetPath()

	err := repoA.AddRemote("origin", remoteAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = repoB.AddRemote("origin", remoteAddr)
	if err != nil {
		t.Fatal(err)
	}

	return repoA, repoB, remote
}

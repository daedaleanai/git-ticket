package bug

import (
	"io"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
)

func pullIdent(repo repository.ClockedRepo, remote string) error {
	_, err := repo.FetchRefs(remote, identity.Namespace)
	if err != nil {
		return err
	}

	for merge := range identity.MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

func pushIdent(repo repository.Repo, remote string, out io.Writer) error {
	_, err := repo.PushRefs(remote, identity.Namespace)
	return err
}

func pull(repo repository.ClockedRepo, remote string) error {
	_, err := repo.FetchRefs(remote, Namespace)
	if err != nil {
		return err
	}

	for merge := range MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure for ticket %s: %s", merge.Id.Human(), merge.Reason)
		}
	}

	return nil
}

func push(repo repository.Repo, remote string, out io.Writer) error {
	_, err := repo.PushRefs(remote, Namespace)
	return err
}

func TestPushPull(t *testing.T) {
	repoA, repoB, remote := repository.SetupReposAndRemote()
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	repository.SetupSigningKey(t, repoA, "a@e.org")
	repository.SetupSigningKey(t, repoB, "a@e.org")

	reneA := identity.NewIdentity("René Descartes", "rene@descartes.fr")

	bug1, _, err := Create(reneA, time.Now().Unix(), "bug1", "message")
	require.NoError(t, err)
	assert.True(t, bug1.NeedCommit())
	err = bug1.Commit(repoA)
	require.NoError(t, err)
	assert.False(t, bug1.NeedCommit())

	// distribute the identity
	err = pushIdent(repoA, "origin", io.Discard)
	require.NoError(t, err)
	err = pullIdent(repoB, "origin")
	require.NoError(t, err)

	// A --> remote --> B
	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	err = pull(repoB, "origin")
	require.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	reneB, err := identity.ReadLocal(repoA, reneA.Id())
	require.NoError(t, err)

	bug2, _, err := Create(reneB, time.Now().Unix(), "bug2", "message")
	require.NoError(t, err)
	err = bug2.Commit(repoB)
	require.NoError(t, err)

	err = push(repoB, "origin", io.Discard)
	require.NoError(t, err)

	err = pull(repoA, "origin")
	require.NoError(t, err)

	bugs = allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func allBugs(t testing.TB, bugs <-chan StreamedBug) []*Bug {
	var result []*Bug
	for streamed := range bugs {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Bug)
	}
	return result
}

func TestRebaseTheirs(t *testing.T) {
	_RebaseTheirs(t)
}

func BenchmarkRebaseTheirs(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseTheirs(b)
	}
}

func _RebaseTheirs(t testing.TB) {
	repoA, repoB, remote := repository.SetupReposAndRemote()
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	repository.SetupSigningKey(t, repoA, "a@e.org")
	repository.SetupSigningKey(t, repoB, "a@e.org")

	reneA := identity.NewIdentity("René Descartes", "rene@descartes.fr")

	bug1, _, err := Create(reneA, time.Now().Unix(), "bug1", "message")
	require.NoError(t, err)
	assert.True(t, bug1.NeedCommit())
	err = bug1.Commit(repoA)
	require.NoError(t, err)
	assert.False(t, bug1.NeedCommit())

	// distribute the identity
	err = pushIdent(repoA, "origin", io.Discard)
	require.NoError(t, err)
	err = pullIdent(repoB, "origin")
	require.NoError(t, err)

	// A --> remote

	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	// remote --> B
	err = pull(repoB, "origin")
	require.NoError(t, err)

	bug2, err := ReadLocalBug(repoB, bug1.Id())
	require.NoError(t, err)
	assert.False(t, bug2.NeedCommit())

	reneB, err := identity.ReadLocal(repoA, reneA.Id())
	require.NoError(t, err)

	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message2")
	require.NoError(t, err)
	assert.True(t, bug2.NeedCommit())
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message3")
	require.NoError(t, err)
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message4")
	require.NoError(t, err)
	err = bug2.Commit(repoB)
	require.NoError(t, err)
	assert.False(t, bug2.NeedCommit())

	// B --> remote
	err = push(repoB, "origin", io.Discard)
	require.NoError(t, err)

	// remote --> A
	err = pull(repoA, "origin")
	require.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := ReadLocalBug(repoA, bug1.Id())
	require.NoError(t, err)

	if nbOps(bug3) != 4 {
		t.Fatal("Unexpected number of operations")
	}
}

func TestRebaseOurs(t *testing.T) {
	_RebaseOurs(t)
}

func BenchmarkRebaseOurs(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseOurs(b)
	}
}

func _RebaseOurs(t testing.TB) {
	repoA, repoB, remote := repository.SetupReposAndRemote()
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	repository.SetupSigningKey(t, repoA, "a@e.org")
	repository.SetupSigningKey(t, repoB, "a@e.org")

	reneA := identity.NewIdentity("René Descartes", "rene@descartes.fr")

	bug1, _, err := Create(reneA, time.Now().Unix(), "bug1", "message")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	// distribute the identity
	err = pushIdent(repoA, "origin", io.Discard)
	require.NoError(t, err)
	err = pullIdent(repoB, "origin")
	require.NoError(t, err)

	// A --> remote
	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	// remote --> B
	err = pull(repoB, "origin")
	require.NoError(t, err)

	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message2")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message3")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message4")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message5")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message6")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message7")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message8")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message9")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message10")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	// remote --> A
	err = pull(repoA, "origin")
	require.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug2, err := ReadLocalBug(repoA, bug1.Id())
	require.NoError(t, err)

	if nbOps(bug2) != 10 {
		t.Fatal("Unexpected number of operations")
	}
}

func nbOps(b *Bug) int {
	it := NewOperationIterator(b)
	counter := 0
	for it.Next() {
		counter++
	}
	return counter
}

func TestRebaseConflict(t *testing.T) {
	_RebaseConflict(t)
}

func BenchmarkRebaseConflict(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseConflict(b)
	}
}

func _RebaseConflict(t testing.TB) {
	repoA, repoB, remote := repository.SetupReposAndRemote()
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	repository.SetupSigningKey(t, repoA, "a@e.org")
	repository.SetupSigningKey(t, repoB, "a@e.org")

	reneA := identity.NewIdentity("René Descartes", "rene@descartes.fr")

	bug1, _, err := Create(reneA, time.Now().Unix(), "bug1", "message")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	// distribute the identity
	err = pushIdent(repoA, "origin", io.Discard)
	require.NoError(t, err)
	err = pullIdent(repoB, "origin")
	require.NoError(t, err)

	// A --> remote
	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	// remote --> B
	err = pull(repoB, "origin")
	require.NoError(t, err)

	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message2")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message3")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message4")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message5")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message6")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message7")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message8")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message9")
	require.NoError(t, err)
	_, err = AddComment(bug1, reneA, time.Now().Unix(), "message10")
	require.NoError(t, err)
	err = bug1.Commit(repoA)
	require.NoError(t, err)

	bug2, err := ReadLocalBug(repoB, bug1.Id())
	require.NoError(t, err)

	reneB, err := identity.ReadLocal(repoA, reneA.Id())
	require.NoError(t, err)

	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message11")
	require.NoError(t, err)
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message12")
	require.NoError(t, err)
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message13")
	require.NoError(t, err)
	err = bug2.Commit(repoB)
	require.NoError(t, err)

	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message14")
	require.NoError(t, err)
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message15")
	require.NoError(t, err)
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message16")
	require.NoError(t, err)
	err = bug2.Commit(repoB)
	require.NoError(t, err)

	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message17")
	require.NoError(t, err)
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message18")
	require.NoError(t, err)
	_, err = AddComment(bug2, reneB, time.Now().Unix(), "message19")
	require.NoError(t, err)
	err = bug2.Commit(repoB)
	require.NoError(t, err)

	// A --> remote
	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	// remote --> B
	err = pull(repoB, "origin")
	require.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := ReadLocalBug(repoB, bug1.Id())
	require.NoError(t, err)

	if nbOps(bug3) != 19 {
		t.Fatal("Unexpected number of operations")
	}

	// B --> remote
	err = push(repoB, "origin", io.Discard)
	require.NoError(t, err)

	// remote --> A
	err = pull(repoA, "origin")
	require.NoError(t, err)

	bugs = allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug4, err := ReadLocalBug(repoA, bug1.Id())
	require.NoError(t, err)

	if nbOps(bug4) != 19 {
		t.Fatal("Unexpected number of operations")
	}
}

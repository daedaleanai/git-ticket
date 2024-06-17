package identity

import (
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/repository"
)

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

	identity1 := NewIdentity("name1", "email1")
	err := identity1.Commit(repoA)
	require.NoError(t, err)

	// A --> remote --> B
	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	err = pull(repoB, "origin")
	require.NoError(t, err)

	identities := allIdentities(t, ReadAllLocalIdentities(repoB))

	if len(identities) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	identity2 := NewIdentity("name2", "email2")
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	err = push(repoB, "origin", io.Discard)
	require.NoError(t, err)

	err = pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Update both

	identity1.addVersionForTest(&Version{
		name:  "name1b",
		email: "email1b",
	})
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	identity2.addVersionForTest(&Version{
		name:  "name2b",
		email: "email2b",
	})
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	err = pull(repoB, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	err = push(repoB, "origin", io.Discard)
	require.NoError(t, err)

	err = pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Concurrent update

	identity1.addVersionForTest(&Version{
		name:  "name1c",
		email: "email1c",
	})
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	identity1B, err := ReadLocal(repoB, identity1.Id())
	require.NoError(t, err)

	identity1B.addVersionForTest(&Version{
		name:  "name1concurrent",
		email: "email1concurrent",
	})
	err = identity1B.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	err = push(repoA, "origin", io.Discard)
	require.NoError(t, err)

	// Pulling a non-fast-forward update should fail
	err = pull(repoB, "origin")
	require.Error(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	// Pushing a non-fast-forward update should fail
	err = push(repoB, "origin", io.Discard)
	require.Error(t, err)

	err = pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func allIdentities(t testing.TB, identities <-chan StreamedIdentity) []*Identity {
	var result []*Identity
	for streamed := range identities {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Identity)
	}
	return result
}

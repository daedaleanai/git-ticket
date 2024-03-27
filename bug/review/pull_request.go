package review

import (
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

// UserStatus is status change by user
type UserStatus interface {
	Author() identity.Interface
	Timestamp() timestamp.Timestamp
	Status() string
}

// Change is the smallest change like single comment or new commit
type Change interface {
	Summary() string
}

// TimelineEvent is interface for event happened to the PullRequest like new review, added commit or status change
type TimelineEvent interface {
	Author() identity.Interface
	Timestamp() timestamp.Timestamp
	Changes() []Change
	Summary() string
}

// IdentityResolver is subset of cache.RepoCache interface used to avoid circular dependency cache->bug->cache
type IdentityResolver interface {
	ResolveIdentityPhabID(phabID string) (identity.Interface, error)
	ResolveIdentityGiteaID(giteaId int64) (identity.Interface, error)
}

// PullRequest is a generic interface for pull request or phabricator revision
type PullRequest interface {
	Id() string
	Title() string

	History() []TimelineEvent

	IsEmpty() bool

	EnsureIdentities(resolver identity.Resolver, found map[entity.Id]identity.Interface) error
	FetchIdentities(resolver IdentityResolver) error

	Merge(update PullRequest)

	LatestOverallStatus() string

	LatestUserStatuses() map[string]UserStatus
}

package review

import (
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

type UserStatus interface {
	Author() identity.Interface
	Timestamp() timestamp.Timestamp
	Status() string
}

type Change interface {
	Summary() string
}

type TimelineEvent interface {
	Author() identity.Interface
	Timestamp() timestamp.Timestamp
	Changes() []Change
}

type IdentityResolver interface {
	IdentityFromPhabId(phabID string) (identity.Interface, error)
	IdentityFromName(name string) (identity.Interface, error)
}

type Pull interface {
	Id() string
	Title() string

	History() []TimelineEvent

	IsEmpty() bool

	EnsureIdentities(resolver identity.Resolver) error
	FetchIdentities(resolver IdentityResolver) error

	Merge(update Pull)

	LatestOverallStatus() string

	LatestUserStatuses() map[string]UserStatus
}

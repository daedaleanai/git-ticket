package identity

import (
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/repository"
)

// Resolver define the interface of an Identity resolver, able to load
// an identity from, for example, a repo or a cache.
type Resolver interface {
	ResolveIdentity(id entity.Id) (Interface, error)
}

// DefaultResolver is a Resolver loading Identities directly from a Repo
type SimpleResolver struct {
	repo repository.ClockedRepo
}

func NewSimpleResolver(repo repository.ClockedRepo) *SimpleResolver {
	return &SimpleResolver{repo: repo}
}

func (r *SimpleResolver) ResolveIdentity(id entity.Id) (Interface, error) {
	return ReadLocal(r.repo, id)
}

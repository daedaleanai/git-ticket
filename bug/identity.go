package bug

import (
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
)

// EnsureIdentities walk the graph of operations and make sure that all Identity
// are properly loaded. That is, it replace all the IdentityStub with the full
// Identity, loaded through a Resolver.
func (bug *Bug) EnsureIdentities(resolver identity.Resolver) error {
	it := NewOperationIterator(bug)

	// only resolve each identity once
	found := make(map[entity.Id]identity.Interface)

	for it.Next() {
		op := it.Value()
		base := op.base()

		if stub, ok := base.Author.(*identity.IdentityStub); ok {
			entity := stub.Id()

			if _, ok := found[entity]; !ok {
				id, err := resolver.ResolveIdentity(entity)
				if err != nil {
					return err
				}
				found[entity] = id
			}

			base.Author = found[entity]
		}

		// if it's an operation which sets the assignee we also need to
		// resolve that
		if setAss, ok := op.(*SetAssigneeOperation); ok {
			entity := setAss.Assignee.Id()

			if _, ok := found[entity]; !ok {
				id, err := resolver.ResolveIdentity(entity)
				if err != nil {
					return err
				}
				found[entity] = id
			}

			setAss.Assignee = found[entity]
		}

		// and for review operations, each of the update authors
		if setRev, ok := op.(*SetReviewOperation); ok {
			err := setRev.Review.EnsureIdentities(resolver, found)
			if err != nil {
				return err
			}
		}

		// and for ccb operations, the user being added / removed
		if setCcb, ok := op.(*SetCcbOperation); ok {
			entity := setCcb.Ccb.User.Id()

			if _, ok := found[entity]; !ok {
				id, err := resolver.ResolveIdentity(entity)
				if err != nil {
					return err
				}
				found[entity] = id
			}

			setCcb.Ccb.User = found[entity]
		}
	}
	return nil
}

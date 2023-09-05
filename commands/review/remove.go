package review

import (
	"github.com/daedaleanai/git-ticket/identity"
)

type RemoveReview struct {
	ReviewId string
}

func (r *RemoveReview) Id() string {
	return r.ReviewId
}

func (r *RemoveReview) Title() string {
	return ""
}

func (r *RemoveReview) History() []TimelineEvent {
	return nil
}

func (r *RemoveReview) IsEmpty() bool {
	return false
}

func (r *RemoveReview) EnsureIdentities(identity.Resolver) error {
	return nil
}

func (r *RemoveReview) FetchIdentities(IdentityResolver) error {
	return nil
}

func (r *RemoveReview) Merge(Pull) {

}

func (r *RemoveReview) LatestOverallStatus() string {
	return "REMOVED"
}

func (r *RemoveReview) LatestUserStatuses() map[string]UserStatus {
	return map[string]UserStatus{}
}

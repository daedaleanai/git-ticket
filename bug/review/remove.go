package review

import (
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
)

// Dummy marker for removing review info
type RemoveReview struct {
	ReviewId string
}

func (r *RemoveReview) Id() string {
	return r.ReviewId
}

func (r *RemoveReview) ReviewUrl() string {
	return r.ReviewId
}

func (r *RemoveReview) Title() string {
	return ""
}

func (r *RemoveReview) History() []TimelineEvent {
	return nil
}

func (r *RemoveReview) IsEmpty() bool {
	return true
}

func (r *RemoveReview) EnsureIdentities(identity.Resolver, map[entity.Id]identity.Interface) error {
	return nil
}

func (r *RemoveReview) FetchIdentities(IdentityResolver) error {
	return nil
}

func (r *RemoveReview) Merge(PullRequest) {

}

func (r *RemoveReview) LatestOverallStatus() string {
	return "REMOVED"
}

func (r *RemoveReview) LatestUserStatuses() map[string]UserStatus {
	return map[string]UserStatus{}
}

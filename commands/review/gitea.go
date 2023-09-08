package review

import (
	"code.gitea.io/sdk/gitea"
	"encoding/json"
	"fmt"
	termtext "github.com/MichaelMure/go-term-text"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/colors"
	"github.com/daedaleanai/git-ticket/util/timestamp"
	"strings"
)

type Comment struct {
	gitea.PullReviewComment
}

func (c *Comment) Summary() string {
	// Put the comment on one line and output the first 50 characters
	output := termtext.LeftPadMaxLine(strings.ReplaceAll(c.Body, "\n", " "), 50, 0)

	// If it's an inline comment append the file and line number
	if c.Path != "" {
		output = output + fmt.Sprintf(" [%s:%d@%s]", c.Path, c.LineNum, c.CommitID)
	}

	return output
}

type GiteaReview struct {
	RawReview gitea.PullReview
	Comments  []Comment
	AuthorId  identity.Interface
}

func (r *GiteaReview) Author() identity.Interface {
	return r.AuthorId
}

func (r *GiteaReview) Timestamp() timestamp.Timestamp {
	return timestamp.Timestamp(r.RawReview.Submitted.Unix())
}

func (r *GiteaReview) Status() string {
	if r.RawReview.Stale {
		return fmt.Sprintf("STALE[%s] @ %s", r.RawReview.State, r.RawReview.CommitID)
	} else {
		return fmt.Sprintf("%s @ %s", r.RawReview.State, r.RawReview.CommitID)
	}
}

func (r *GiteaReview) Changes() []Change {
	result := []Change{}
	for _, c := range r.Comments {
		cc := c // Without it golang store pointer to changing loop variable
		result = append(result, &cc)
	}
	return result
}

type GiteaInfo struct {
	Owner      string
	Repository string
	PullId     int64

	RawPull gitea.PullRequest
	Reviews []GiteaReview
}

func (g *GiteaInfo) Id() string {
	return fmt.Sprintf("%s/%s/pulls/%d", g.Owner, g.Repository, g.PullId)
}

func (g *GiteaInfo) Title() string {
	return g.RawPull.Title
}

func (g *GiteaInfo) History() []TimelineEvent {
	result := []TimelineEvent{}

	for _, r := range g.Reviews {
		upd := r
		result = append(result, &upd)
	}
	return result
}

func (g *GiteaInfo) IsEmpty() bool {
	return false
}

func (g *GiteaInfo) EnsureIdentities(resolver identity.Resolver) error {
	for i, _ := range g.Reviews {
		user, err := resolver.ResolveIdentity(g.Reviews[i].AuthorId.Id())
		if err != nil {
			return err
		}
		g.Reviews[i].AuthorId = user
	}
	return nil
}

func (g *GiteaInfo) FetchIdentities(resolver IdentityResolver) error {
	for i, t := range g.Reviews {
		user, err := resolver.IdentityFromName(t.RawReview.Reviewer.FullName)

		if err != nil {
			return fmt.Errorf("%s: %s", err, t.RawReview.Reviewer.FullName)
		}

		g.Reviews[i].AuthorId = user
	}

	return nil
}

func (g *GiteaInfo) Merge(update Pull) {
	if update == nil {
		return
	}

	u := update.(*GiteaInfo)
	g.Owner = u.Owner
	g.Repository = u.Repository
	g.PullId = u.PullId
	g.RawPull = u.RawPull
	g.Reviews = u.Reviews
}

func (g *GiteaInfo) LatestOverallStatus() string {
	result := map[string]*GiteaReview{}

	for _, r := range g.Reviews {
		if s, ok := result[r.RawReview.Reviewer.Email]; !ok || s.Timestamp() < r.Timestamp() {
			us := r
			result[r.RawReview.Reviewer.Email] = &us
		}
	}

	approved := false
	rejected := false

	for _, r := range result {
		if !r.RawReview.Stale && r.RawReview.State == gitea.ReviewStateApproved {
			approved = true
		} else if !r.RawReview.Stale && r.RawReview.State == gitea.ReviewStateRequestChanges {
			rejected = true
		}
	}

	if approved && !rejected {
		return colors.Green(string(gitea.ReviewStateApproved))
	} else if rejected {
		return colors.Red(string(gitea.ReviewStateRequestChanges))
	} else {
		return string(gitea.ReviewStatePending)
	}
}

func (g *GiteaInfo) LatestUserStatuses() map[string]UserStatus {
	result := map[string]UserStatus{}

	for _, r := range g.Reviews {
		if s, ok := result[r.RawReview.Reviewer.Email]; !ok || s.Timestamp() < r.Timestamp() {
			us := r // Without it golang store pointer to changing loop variable
			result[r.RawReview.Reviewer.Email] = &us
		}
	}

	return result
}

func FetchGiteaReviewInfo(owner string, repo string, id int64) (Pull, error) {
	const PAGE_SIZE = 10

	giteaClient, err := repository.GetGiteaClient()

	if err != nil {
		return nil, err
	}

	pull, _, err := giteaClient.GetPullRequest(owner, repo, id)

	if err != nil {
		return nil, err
	}

	result := GiteaInfo{
		PullId:     id,
		Owner:      owner,
		Repository: repo,
		RawPull:    *pull,
	}

	var page = 1
	for {
		reviews, _, err := giteaClient.ListPullReviews(owner, repo, id, gitea.ListPullReviewsOptions{gitea.ListOptions{Page: page, PageSize: PAGE_SIZE}})

		if err != nil {
			return nil, err
		}

		for _, review := range reviews {
			elem := GiteaReview{
				RawReview: *review,
			}

			if review.CodeCommentsCount > 0 {
				comments, _, err := giteaClient.ListPullReviewComments(owner, repo, id, review.ID)
				if err != nil {
					return nil, err
				}

				for _, c := range comments {
					elem.Comments = append(elem.Comments, Comment{*c})
				}
			}

			result.Reviews = append(result.Reviews, elem)
		}

		if len(reviews) < PAGE_SIZE {
			break
		}

		page = page + 1
	}

	return &result, nil
}

// UnmarshalJSON fulfils the Marshaler interface so that we can handle the author identity
func (u *GiteaReview) UnmarshalJSON(data []byte) error {
	type rawUpdate struct {
		RawReview gitea.PullReview
		Comments  []Comment
		AuthorId  json.RawMessage
	}

	var raw rawUpdate
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	u.RawReview = raw.RawReview
	u.Comments = raw.Comments

	if raw.AuthorId != nil {
		author, err := identity.UnmarshalJSON(raw.AuthorId)
		if err != nil {
			return err
		}
		u.AuthorId = author
	}

	return nil
}

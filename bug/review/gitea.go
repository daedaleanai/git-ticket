package review

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	termtext "github.com/MichaelMure/go-term-text"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/colors"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

// Comment holds data about single review comment
type Comment struct {
	RawComment gitea.PullReviewComment
	Update     bool
}

// Summary returns a string containing the comment text, and it's an inline
// comment the file & line details, on a single line. Comments over 50 characters
// are truncated.
func (c *Comment) Summary() string {
	// Put the comment on one line and output the first 50 characters
	output := termtext.LeftPadMaxLine(strings.ReplaceAll(strings.ReplaceAll(c.RawComment.Body, "\n", " "), "\r", ""), 50, 0)

	// If it's an inline comment append the file and line number
	if c.RawComment.Path != "" {
		output = output + fmt.Sprintf(" [%s:%d@%s]", c.RawComment.Path, c.RawComment.LineNum, c.RawComment.CommitID)
	}

	return output
}

// GiteaCommitChange is a wrapper to represent Change made by commit
type GiteaCommitChange struct {
	RawCommit gitea.Commit
}

// Summary returns a string containing the commit message. Comments over 50 characters
// are truncated.
func (c *GiteaCommitChange) Summary() string {
	// Put the comment on one line and output the first 50 characters
	message := termtext.LeftPadMaxLine(strings.ReplaceAll(c.RawCommit.RepoCommit.Message, "\n", " "), 50, 0)

	return fmt.Sprintf("%s [commit %s]", message, c.RawCommit.SHA)
}

// GiteaCommit holds info about commit from Gitea pull request
type GiteaCommit struct {
	RawCommit gitea.Commit
	AuthorId  identity.Interface
	reviewId  string
}

// Author returns commit author
func (r *GiteaCommit) Author() identity.Interface {
	return r.AuthorId
}

// Timestamp returns commit time
func (r *GiteaCommit) Timestamp() timestamp.Timestamp {
	return timestamp.Timestamp(r.RawCommit.Created.Unix())
}

// Status returns "COMMIT" constant for commit
func (r *GiteaCommit) Status() string {
	return "COMMIT"
}

// Changes returns list of a single change representing this commit
func (r *GiteaCommit) Changes() []Change {
	return []Change{&GiteaCommitChange{r.RawCommit}}
}

// Summary returns a string containing the commit message. Comments over 50 characters
// are truncated.
func (g *GiteaCommit) Summary() string {
	// Put the comment on one line and output the first 50 characters
	message := termtext.LeftPadMaxLine(strings.ReplaceAll(g.RawCommit.RepoCommit.Message, "\n", " "), 50, 0)

	return fmt.Sprintf("[commit %s] %s", g.RawCommit.SHA, message)
}

// GiteaReview holds single review event from Gitea
type GiteaReview struct {
	RawReview gitea.PullReview
	Comments  []Comment
	AuthorId  identity.Interface
	reviewId  string
}

// Author returns author of the change
func (r *GiteaReview) Author() identity.Interface {
	return r.AuthorId
}

// Timestamp returns timestamp of the event
func (r *GiteaReview) Timestamp() timestamp.Timestamp {
	return timestamp.Timestamp(r.RawReview.Submitted.Unix())
}

// Status returns status change by this event (possibly just comment)
func (r *GiteaReview) Status() string {
	if r.RawReview.Stale {
		return fmt.Sprintf("STALE[%s] @ %s", r.RawReview.State, r.RawReview.CommitID)
	} else {
		return fmt.Sprintf("%s @ %s", r.RawReview.State, r.RawReview.CommitID)
	}
}

// Changes returns list of all changes in event (e.g. all comments)
func (r *GiteaReview) Changes() []Change {
	result := []Change{}
	for _, c := range r.Comments {
		cc := c // Without it golang store pointer to changing loop variable
		result = append(result, &cc)
	}
	return result
}

// Summary returns a short description of the event
func (g *GiteaReview) Summary() string {
	var output strings.Builder

	if g.RawReview.State != gitea.ReviewStateComment {
		output.WriteString("[" + string(g.RawReview.State) + "] ")
	}

	comments := len(g.Comments)
	if g.RawReview.Body != "" {
		comments = comments + 1
	}

	if comments > 1 {
		output.WriteString("[" + strconv.Itoa(comments) + " comments] ")
	} else if comments > 0 {
		output.WriteString("[1 comment] ")
	}

	return output.String()
}

// GiteaInfo is Gitea-specific implementation of PullRequest
type GiteaInfo struct {
	Owner      string
	Repository string
	PullId     int64

	RawPull gitea.PullRequest
	Reviews []GiteaReview
	Commits []GiteaCommit
}

// Id returns Phabricator revision id
func (g *GiteaInfo) Id() string {
	giteaUrl, _, _ := repository.GetGiteaConfig()
	if giteaUrl != "" {
		return fmt.Sprintf("%s/%s/%s/pulls/%d", giteaUrl, g.Owner, g.Repository, g.PullId)
	} else {
		return fmt.Sprintf("%s/%s#%d", g.Owner, g.Repository, g.PullId)
	}
}

// Title returns Phabricator revision title
func (g *GiteaInfo) Title() string {
	return g.RawPull.Title
}

// History returns all events from revision sorted by time
func (g *GiteaInfo) History() []TimelineEvent {
	result := []TimelineEvent{}

	for _, r := range g.Reviews {
		upd := r
		result = append(result, &upd)
	}
	for _, c := range g.Commits {
		upd := c
		result = append(result, &upd)
	}
	return result
}

// IsEmpty check if there is any changes
func (g *GiteaInfo) IsEmpty() bool {
	return len(g.Reviews)+len(g.Commits) == 0
}

// EnsureIdentities validated if all users are resolved
func (g *GiteaInfo) EnsureIdentities(resolver identity.Resolver, found map[entity.Id]identity.Interface) error {
	for i := range g.Reviews {
		entity := g.Reviews[i].AuthorId.Id()

		if _, ok := found[entity]; !ok {
			id, err := resolver.ResolveIdentity(entity)
			if err != nil {
				return err
			}
			found[entity] = id
		}

		g.Reviews[i].AuthorId = found[entity]
	}

	for i := range g.Commits {
		entity := g.Commits[i].AuthorId.Id()

		if _, ok := found[entity]; !ok {
			id, err := resolver.ResolveIdentity(entity)
			if err != nil {
				return err
			}
			found[entity] = id
		}

		g.Commits[i].AuthorId = found[entity]
	}
	return nil
}

// FetchIdentities resolves users from pull request to git-ticket identities
func (g *GiteaInfo) FetchIdentities(resolver IdentityResolver) error {
	for i, t := range g.Reviews {
		user, err := resolver.ResolveIdentityGiteaID(t.RawReview.Reviewer.ID)

		if err != nil {
			return fmt.Errorf("%s: %s (Gitea ID: %v)", err, t.RawReview.Reviewer.FullName, t.RawReview.Reviewer.ID)
		}

		g.Reviews[i].AuthorId = user
	}

	for i, t := range g.Commits {
		user, err := resolver.ResolveIdentityGiteaID(t.RawCommit.Author.ID)

		if err != nil {
			return fmt.Errorf("%s: %s (Gitea ID: %v)", err, t.RawCommit.Author.FullName, t.RawCommit.Author.ID)
		}

		g.Commits[i].AuthorId = user
	}

	return nil
}

// Merge updates state from new one
func (g *GiteaInfo) Merge(update PullRequest) {
	if update == nil {
		return
	}

	u := update.(*GiteaInfo)
	g.Owner = u.Owner
	g.Repository = u.Repository
	g.PullId = u.PullId
	g.RawPull = u.RawPull
	g.Reviews = append(g.Reviews, u.Reviews...)
	g.Commits = append(g.Commits, u.Commits...)
}

// LatestOverallStatus returns the latest overall status set for this review.
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

// LatestUserStatuses returns a map of users and the latest status they set for
// this review.
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

// FetchGiteaReviewInfo exports review comments and status info from Gitea for
// the given pull request and returns in a PullRequest object. If since review is specified
// only updates will be returned
func FetchGiteaReviewInfo(owner string, repo string, id int64, since *GiteaInfo) (PullRequest, error) {
	const PAGE_SIZE = 10

	giteaClient, err := repository.GetGiteaClient()

	if err != nil {
		return nil, err
	}

	knownCommits := map[string]struct{}{}
	knownReviews := map[int64]GiteaReview{}

	if since != nil {
		for _, r := range since.Reviews {
			knownReviews[r.RawReview.ID] = r
		}

		for _, c := range since.Commits {
			knownCommits[c.RawCommit.SHA] = struct{}{}
		}
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
			knownComments := map[int64]gitea.PullReviewComment{}
			isKnown := false
			if r, ok := knownReviews[review.ID]; ok {
				isKnown = true
				for _, c := range r.Comments {
					knownComments[c.RawComment.ID] = c.RawComment
				}
			}

			elem := GiteaReview{
				RawReview: *review,
				reviewId:  result.Id(),
			}

			if review.CodeCommentsCount > 0 {
				comments, _, err := giteaClient.ListPullReviewComments(owner, repo, id, review.ID)
				if err != nil {
					return nil, err
				}

				for _, c := range comments {

					if o, ok := knownComments[c.ID]; ok {
						if o.Updated.Before(c.Updated) {
							elem.Comments = append(elem.Comments, Comment{RawComment: *c, Update: true})
						}
						continue
					}

					elem.Comments = append(elem.Comments, Comment{RawComment: *c, Update: false})
				}
			}
			if isKnown && len(elem.Comments) == 0 {
				continue
			}
			result.Reviews = append(result.Reviews, elem)
		}

		if len(reviews) < PAGE_SIZE {
			break
		}

		page = page + 1
	}

	page = 1
	for {
		commits, _, err := giteaClient.ListPullRequestCommits(owner, repo, id, gitea.ListPullRequestCommitsOptions{gitea.ListOptions{Page: page, PageSize: PAGE_SIZE}})
		if err != nil {
			return nil, err
		}

		for _, c := range commits {

			if _, ok := knownCommits[c.SHA]; ok {
				continue
			}

			result.Commits = append(result.Commits, GiteaCommit{
				RawCommit: *c,
				reviewId:  result.Id(),
			})
		}

		if len(commits) < PAGE_SIZE {
			break
		}

		page = page + 1

	}

	return &result, nil
}

// UnmarshalJSON fulfils the Marshaler interface so that we can handle the author identity
func (u *GiteaCommit) UnmarshalJSON(data []byte) error {
	type rawUpdate struct {
		RawCommit gitea.Commit
		AuthorId  json.RawMessage
	}

	var raw rawUpdate
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	u.RawCommit = raw.RawCommit

	if raw.AuthorId != nil {
		author, err := identity.UnmarshalJSON(raw.AuthorId)
		if err != nil {
			return err
		}
		u.AuthorId = author
	}

	return nil
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

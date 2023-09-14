package bug

import (
	"fmt"
	review2 "github.com/daedaleanai/git-ticket/bug/review"
	"regexp"
	"strconv"
)

// FetchReviewInfo exports review comments and status info from Phabricator or Gitea for
// the given differential ID and returns in a PullRequest struct. If a since review
// is specified then only updates since then are returned (only for Phabricator).
func FetchReviewInfo(id string, since review2.PullRequest) (review2.PullRequest, error) {
	prRefRegex := regexp.MustCompile(`^([a-zA-Z0-9-_]+)/([a-zA-Z0-9-_]+)#(\d+)$`)
	prUrlRegex := regexp.MustCompile(`^([a-zA-Z0-9-_]+)/([a-zA-Z0-9-_]+)/pulls/(\d+)$`)
	if matched, _ := regexp.MatchString(`^D\d+$`, id); matched {
		lastTransaction := ""
		if since != nil {
			lastTransaction = since.(*review2.PhabReviewInfo).LastTransaction
		}
		return review2.FetchPhabricatorReviewInfo(id, lastTransaction)
	} else {
		matched := prRefRegex.FindStringSubmatch(id)
		if matched == nil {
			matched = prUrlRegex.FindStringSubmatch(id)
		}
		if matched != nil {
			idx, err := strconv.Atoi(matched[3])
			if err != nil {
				return nil, fmt.Errorf("unable to parse id: %s", err)
			}
			var old *review2.GiteaInfo
			if since != nil {
				old = since.(*review2.GiteaInfo)
			}
			return review2.FetchGiteaReviewInfo(matched[1], matched[2], int64(idx), old)
		} else {
			return nil, fmt.Errorf("differential/pr id '%s' unexpected format (Dnnn for Phabricator, <owner>/<repo>#<id> or <owner>/<repo>/pulls/<id> for Gitea) ", id)
		}
	}
}

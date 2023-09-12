package bug

import (
	"fmt"
	review2 "github.com/daedaleanai/git-ticket/bug/review"
	"regexp"
	"strconv"
)

// FetchReviewInfo exports review comments and status info from Phabricator or Gitea for
// the given differential ID and returns in a PullRequest struct. If a since
// transaction ID is specified then only updates since then are returned (only for Phabricator).
func FetchReviewInfo(id string, since string) (review2.PullRequest, error) {
	prRefRegex := regexp.MustCompile(`^([a-zA-Z0-9-_]+)/([a-zA-Z0-9-_]+)#(\d+)$`)
	prUrlRegex := regexp.MustCompile(`^([a-zA-Z0-9-_]+)/([a-zA-Z0-9-_]+)/pulls/(\d+)$`)
	if matched, _ := regexp.MatchString(`^D\d+$`, id); matched {
		return review2.FetchPhabricatorReviewInfo(id, since)
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
			return review2.FetchGiteaReviewInfo(matched[1], matched[2], int64(idx))
		} else {
			return nil, fmt.Errorf("differential/pr id '%s' unexpected format (Dnnn for Phabricator, <owner>/<repo>#<id> or <owner>/<repo>/pulls/<id> for Gitea) ", id)
		}
	}
}

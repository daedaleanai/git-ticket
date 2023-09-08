package bug

import (
	"fmt"
	"github.com/daedaleanai/git-ticket/commands/review"
	"github.com/daedaleanai/git-ticket/repository"
	"regexp"
	"strconv"
	"strings"
)

// FetchReviewInfo exports review comments and status info from Phabricator for
// the given differential ID and returns in a ReviewInfo struct. If a since
// transaction ID is specified then only updates since then are returned.
func FetchReviewInfo(id string, since string) (review.Pull, error) {
	prUrlRegex := regexp.MustCompile(`^.*/([a-zA-Z0-9-_]+)/([a-zA-Z0-9-_]+)/pulls/(\d+)$`)
	giteaUrl, _, err := repository.GetGiteaConfig()
	if matched, _ := regexp.MatchString(`^D\d+$`, id); matched {
		return review.FetchPhabricatorReviewInfo(id, since)
	} else if err != nil {
		return nil, fmt.Errorf("differential/pr id '%s' unexpected format (Dnnn) and Gitea url is not configures ", id)
	} else if matched := prUrlRegex.FindStringSubmatch(id); strings.HasPrefix(id, giteaUrl) && matched != nil {
		idx, err := strconv.Atoi(matched[3])
		if err != nil {
			return nil, err
		}
		return review.FetchGiteaReviewInfo(matched[1], matched[2], int64(idx))
	} else {
		return nil, fmt.Errorf("differential/pr id '%s' unexpected format (Dnnn or %s/<owner>/<repo>/pulls/<id>) ", id, giteaUrl)
	}
}

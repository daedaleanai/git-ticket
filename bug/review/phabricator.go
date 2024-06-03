package review

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	termtext "github.com/MichaelMure/go-term-text"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/colors"
	"github.com/daedaleanai/git-ticket/util/timestamp"
	"github.com/thought-machine/gonduit/requests"
)

type TransactionType int

const (
	_ TransactionType = iota
	CommentTransaction
	StatusTransaction
	UserStatusTransaction
	DiffTransaction
)

// PhabTransaction holds data received from Phabricator
type PhabTransaction struct {
	TransId   string
	PhabUser  string
	Timestamp int64

	Type TransactionType
	// comment specific fields
	Diff int    `json:",omitempty"` // diff id comment was made againt, inline comments only
	Path string `json:",omitempty"` // file path, inline comments only
	Line int    `json:",omitempty"` // line number, inline comments only
	Text string `json:",omitempty"`
	// status and userstatus specific fields
	Status string `json:",omitempty"`
	// diff specific fields
	DiffId int `json:",omitempty"`
}

// ReviewUpdate extends the Phabricator data with git ticket information
type ReviewUpdate struct {
	PhabTransaction
	AuthorId identity.Interface `json:"Author,omitempty"`
}

// Author returns author of the change
func (u *ReviewUpdate) Author() identity.Interface {
	return u.AuthorId
}

// Timestamp returns timestamp of the event
func (u *ReviewUpdate) Timestamp() timestamp.Timestamp {
	return timestamp.Timestamp(u.PhabTransaction.Timestamp)
}

// Status returns status change by this event or empty string
func (u *ReviewUpdate) Status() string {
	return u.PhabTransaction.Status
}

// PhabUpdateGroup is Phabricator-specific implementation of TimelineEvent
type PhabUpdateGroup struct {
	timestamp  timestamp.Timestamp
	author     identity.Interface
	updates    []ReviewUpdate
	revisionId string
}

// Author returns author of the event
func (g *PhabUpdateGroup) Author() identity.Interface {
	return g.author
}

// Timestamp returns timestamp of the event
func (g *PhabUpdateGroup) Timestamp() timestamp.Timestamp {
	return g.timestamp
}

// Changes returns list of all changes in event (e.g. all comments)
func (g *PhabUpdateGroup) Changes() []Change {
	result := []Change{}

	for _, u := range g.updates {
		upd := u // Without it golang store pointer to changing loop variable
		result = append(result, &upd)
	}
	return result
}

// Summary returns a short description of the event
func (g *PhabUpdateGroup) Summary() string {
	var output strings.Builder
	var comments int

	for _, u := range g.updates {
		switch u.Type {
		case UserStatusTransaction:
			output.WriteString("[" + u.Status() + "] ")
		case DiffTransaction:
			output.WriteString("[diff>" + strconv.Itoa(u.DiffId) + "] ")
		case CommentTransaction:
			comments = comments + 1
		}
	}

	if comments > 1 {
		output.WriteString("[" + strconv.Itoa(comments) + " comments] ")
	} else if comments > 0 {
		output.WriteString("[1 comment] ")
	}

	return output.String()
}

// Summary returns a string containing the comment text, and it's an inline
// comment the file & line details, on a single line. Comments over 50 characters
// are truncated.
func (c *ReviewUpdate) Summary() string {
	if c.Type != CommentTransaction {
		return ""
	}

	// Put the comment on one line and output the first 50 characters
	output := termtext.LeftPadMaxLine(strings.ReplaceAll(c.Text, "\n", " "), 50, 0)

	// If it's an inline comment append the file and line number
	if c.Path != "" {
		output = output + fmt.Sprintf(" [%s:%d@%d]", c.Path, c.Line, c.Diff)
	}

	return output
}

// PhabReviewInfo is Phabricator-specific implementation of PullRequest
type PhabReviewInfo struct {
	RevisionId      string // e.g. D1234
	RevisionTitle   string `json:"Title"`
	LastTransaction string
	Updates         []ReviewUpdate
}

// Id returns Phabricator revision id
func (r *PhabReviewInfo) Id() string {
	return r.RevisionId
}

func (r *PhabReviewInfo) ReviewUrl() string {
	phabUrl, _, _ := repository.GetPhabConfig()
	if phabUrl != "" {
		return fmt.Sprintf("%s/%s", phabUrl, r.RevisionId)
	}

	// Fallback to the ID if URL is unknown
	return r.Id()
}

// Title returns Phabricator revision title
func (r *PhabReviewInfo) Title() string {
	return r.RevisionTitle
}

// History returns all events from revision sorted by time
func (r *PhabReviewInfo) History() []TimelineEvent {
	// Create a map of timeline items to changes, we'll assume that all changes that
	// happened at the same time were done by the same person
	timelineMap := make(map[int64]*PhabUpdateGroup)

	for _, u := range r.Updates {
		if tl, exists := timelineMap[u.PhabTransaction.Timestamp]; exists {
			// Not the first change at this timestamp, update the one in the map
			tl.updates = append(tl.updates, u)
			timelineMap[u.PhabTransaction.Timestamp] = tl
		} else {
			// First one, create a new timeline item using the update author and timestamp
			item := &PhabUpdateGroup{
				author:     u.Author(),
				timestamp:  u.Timestamp(),
				updates:    []ReviewUpdate{u},
				revisionId: r.RevisionId,
			}
			timelineMap[u.PhabTransaction.Timestamp] = item
		}
	}

	result := []TimelineEvent{}

	// Add all the timeline items to the snapshot, finally sort them
	for _, tl := range timelineMap {
		result = append(result, tl)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp() > result[j].Timestamp()
	})
	return result
}

// IsEmpty checks is revision has any events
func (r *PhabReviewInfo) IsEmpty() bool {
	return len(r.Updates) == 0
}

// EnsureIdentities validated if all users are resolved
func (r *PhabReviewInfo) EnsureIdentities(resolver identity.Resolver, found map[entity.Id]identity.Interface) error {
	for i, u := range r.Updates {
		entity := u.Author().Id()

		if _, ok := found[entity]; !ok {
			id, err := resolver.ResolveIdentity(entity)
			if err != nil {
				return err
			}
			found[entity] = id
		}

		r.Updates[i].AuthorId = found[entity]
	}

	return nil
}

// FetchIdentities resolves users from revision to git-ticket identities
func (r *PhabReviewInfo) FetchIdentities(resolver IdentityResolver) error {
	for i, t := range r.Updates {
		user, err := resolver.ResolveIdentityPhabID(t.PhabUser)
		if err != nil {
			return fmt.Errorf("%s: %s", err, t.PhabUser)
		}
		r.Updates[i].AuthorId = user
	}
	return nil
}

// Merge combines old state with incremental update
func (r *PhabReviewInfo) Merge(update PullRequest) {
	if update == nil {
		return
	}

	u := update.(*PhabReviewInfo)
	r.RevisionId = u.RevisionId
	r.RevisionTitle = u.RevisionTitle

	r.LastTransaction = u.LastTransaction
	r.Updates = append(r.Updates, u.Updates...)
}

// LatestOverallStatus returns the latest overall status set for this review.
func (r *PhabReviewInfo) LatestOverallStatus() string {
	var ls ReviewUpdate

	for _, s := range r.Updates {
		if s.Type == StatusTransaction && s.Timestamp() > ls.Timestamp() {
			ls = s
		}
	}

	if ls.Status() == "accepted" {
		return colors.Green("accepted")
	} else {
		return ls.Status()
	}
}

// LatestUserStatuses returns a map of users and the latest status they set for
// this review.
func (r *PhabReviewInfo) LatestUserStatuses() map[string]UserStatus {
	// Create a map of the latest status change made by all users
	userStatusChange := make(map[string]UserStatus)

	for _, s := range r.Updates {
		if s.Type != UserStatusTransaction {
			continue
		}
		if sc, present := userStatusChange[s.PhabUser]; !present || s.Timestamp() > sc.Timestamp() {
			var upd = s // Without it golang store pointer to changing loop variable
			userStatusChange[s.PhabUser] = &upd
		}
	}

	return userStatusChange
}

// statusActionToState maps states returned by Phabricator on to more readable strings
var statusActionToState = map[string]string{
	"accept":          "accepted",
	"close":           "closed",
	"create":          "created",
	"request-changes": "changes requested",
	"request-review":  "review requested",
}

// FetchPhabricatorReviewInfo exports review comments and status info from Phabricator for
// the given differential ID and returns in a PullRequest object. If a since
// transaction ID is specified then only updates since then are returned.
func FetchPhabricatorReviewInfo(id string, since string) (PullRequest, error) {
	result := PhabReviewInfo{RevisionId: id}

	phabClient, err := repository.GetPhabClient()
	if err != nil {
		return nil, err
	}

	var before string
	var after string
	var deltaUpdate bool

	// If since is set then only get the transactions since then, else get them all
	if since != "" {
		before = since
		deltaUpdate = true
	}

	for {

		request := requests.TransactionSearchRequest{ObjectID: id,
			Before: before,
			After:  after,
			Limit:  100}

		response, err := phabClient.TransactionSearch(request)
		if err != nil {
			return nil, err
		}

		if len(response.Data) == 0 {
			break
		}

		// If the Cursor.Before field is blank this response includes the latest
		// transactions, position 0 has the newest
		if response.Cursor.Before == nil {
			result.LastTransaction = strconv.Itoa(response.Data[0].ID)
		}

		// Loop through all transactions
		for _, t := range response.Data {

			transData := ReviewUpdate{
				PhabTransaction: PhabTransaction{
					TransId:   strconv.Itoa(t.ID),
					PhabUser:  t.AuthorPHID,
					Timestamp: time.Time(t.DateCreated).Unix()}}

			if !strings.HasPrefix(transData.PhabUser, "PHID-USER-") {
				// Silently drop transaction data which wasn't created by an actual user
				continue
			}

			switch t.Type {
			// The types: inline & comment hold comments made to a Differential

			case "inline":
				// If it's an inline comment the Fields contains the file path, line and diff ID
				diff := t.Fields["diff"].(map[string]interface{})
				commentDiff := int(diff["id"].(float64))
				commentPath := t.Fields["path"].(string)
				commentLine := int(t.Fields["line"].(float64))

				transData.Type = CommentTransaction

				for _, c := range t.Comments {
					transData.Diff = commentDiff
					transData.Path = commentPath
					transData.Line = commentLine
					transData.Text = c.Content["raw"].(string)

					result.Updates = append(result.Updates, transData)
				}

			case "comment":
				transData.Type = CommentTransaction

				for _, c := range t.Comments {
					transData.Text = c.Content["raw"].(string)

					result.Updates = append(result.Updates, transData)
				}

			case "status":
				transData.Type = StatusTransaction
				transData.PhabTransaction.Status = t.Fields["new"].(string)

				result.Updates = append(result.Updates, transData)

			case "accept", "close", "create", "request-changes", "request-review":
				transData.Type = UserStatusTransaction
				transData.PhabTransaction.Status = statusActionToState[t.Type]

				result.Updates = append(result.Updates, transData)

			case "title":
				result.RevisionTitle = t.Fields["new"].(string)

			case "update":
				// if it's an update then query Phabricator for the Diff id rather than storing the PHID for it
				phidDiff := t.Fields["new"].(string)
				searchConstraint := map[string]interface{}{"phids": [...]string{phidDiff}}
				request := requests.SearchRequest{Constraints: searchConstraint, Limit: 1}

				response, err := phabClient.DifferentialDiffSearch(request)
				if err != nil {
					return nil, err
				}
				if len(response.Data) < 1 {
					return nil, fmt.Errorf("differential %s includes diff %s which gave zero results", id, phidDiff)
				}

				transData.Type = DiffTransaction
				transData.DiffId = response.Data[0].ID

				result.Updates = append(result.Updates, transData)
			}
		}

		if deltaUpdate {
			// If we requested only transactions after a certain one (by setting the request
			// "before" field) then Phabricator sends the oldest transactions first, if there's
			// more than the "limit" remaining then the Cursor.Before field will be set to
			// indicate more newer ones are available.
			if response.Cursor.Before == nil {
				// there's no more transactions to get
				break
			}
			before = response.Cursor.Before.(string)
		} else {
			// If we requested all transactions then Phabricator sends the newest transactions
			// first, if there's more than the "limit" remaining then the Cursor.After field
			// will be set to indicate more older ones are available.
			if response.Cursor.After == nil {
				// there's no more transactions to get
				break
			}
			after = response.Cursor.After.(string)
		}

	}

	return &result, nil
}

// UnmarshalJSON fulfils the Marshaler interface so that we can handle the author identity
func (u *ReviewUpdate) UnmarshalJSON(data []byte) error {
	type rawUpdate struct {
		PhabTransaction
		Author json.RawMessage
	}

	var raw rawUpdate
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	u.PhabTransaction = raw.PhabTransaction

	if raw.Author != nil {
		author, err := identity.UnmarshalJSON(raw.Author)
		if err != nil {
			return err
		}
		u.AuthorId = author
	}

	return nil
}

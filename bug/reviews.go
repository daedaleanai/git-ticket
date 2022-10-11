package bug

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	termtext "github.com/MichaelMure/go-term-text"
	"github.com/thought-machine/gonduit/requests"

	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/colors"
)

type TransactionType int

const (
	_ TransactionType = iota
	CommentTransaction
	StatusTransaction
	UserStatusTransaction
	DiffTransaction
)

// PhapTransaction holds data received from Phabricator
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
	Author identity.Interface `json:",omitempty"`
}

// ReviewInfo holds a set of comment and status updates related to a diff
type ReviewInfo struct {
	RevisionId      string // e.g. D1234
	Title           string
	LastTransaction string
	Updates         []ReviewUpdate
}

// statusActionToState maps states returned by Phabricator on to more readable strings
var statusActionToState = map[string]string{
	"accept":          "accepted",
	"close":           "closed",
	"create":          "created",
	"request-changes": "changes requested",
	"request-review":  "review requested",
}

const RemoveReviewInfo = "-1"

// OneLineComment returns a string containing the comment text, and it's an inline
// comment the file & line details, on a single line. Comments over 50 characters
// are truncated.
func (c PhabTransaction) OneLineComment() string {
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
		u.Author = author
	}

	return nil
}

// LatestOverallStatus returns the latest overall status set for this review.
func (r ReviewInfo) LatestOverallStatus() string {
	var ls ReviewUpdate

	for _, s := range r.Updates {
		if s.Type == StatusTransaction && s.Timestamp > ls.Timestamp {
			ls = s
		}
	}

	if ls.Status == "accepted" {
		return colors.Green("accepted")
	} else {
		return ls.Status
	}
}

// LatestUserStatuses returns a map of users and the latest status they set for
// this review.
func (r ReviewInfo) LatestUserStatuses() map[string]ReviewUpdate {
	// Create a map of the latest status change made by all users
	userStatusChange := make(map[string]ReviewUpdate)

	for _, s := range r.Updates {
		if s.Type != UserStatusTransaction {
			continue
		}
		if sc, present := userStatusChange[s.PhabUser]; !present || s.Timestamp > sc.Timestamp {
			userStatusChange[s.PhabUser] = s
		}
	}

	return userStatusChange
}

// FetchReviewInfo exports review comments and status info from Phabricator for
// the given differential ID and returns in a ReviewInfo struct. If a since
// transaction ID is specified then only updates since then are returned.
func FetchReviewInfo(id string, since string) (*ReviewInfo, error) {

	if matched, _ := regexp.MatchString(`^D\d+$`, id); !matched {
		return nil, fmt.Errorf("differential id '%s' unexpected format (Dnnn)", id)
	}

	result := ReviewInfo{RevisionId: id}

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
				transData.Status = t.Fields["new"].(string)

				result.Updates = append(result.Updates, transData)

			case "accept", "close", "create", "request-changes", "request-review":
				transData.Type = UserStatusTransaction
				transData.Status = statusActionToState[t.Type]

				result.Updates = append(result.Updates, transData)

			case "title":
				result.Title = t.Fields["new"].(string)

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

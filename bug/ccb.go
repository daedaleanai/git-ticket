package bug

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/pkg/errors"
)

// CcbState represents the state of an approver with respect to a ticket status
type CcbState int

const (
	_                CcbState = iota
	AddedCcbState             // added to the ticket, but not set a state
	ApprovedCcbState          // approved the ticket
	BlockedCcbState           // blocked the ticket
	RemovedCcbState           // removed from the ticket
)

// CcbInfo is stored in a ticket history every time an approver is added or removed,
// or has approved or blocked the ticket
type CcbInfo struct {
	User   identity.Interface // The approver
	Status Status             // The ticket status (e.g. vetted) that the approver is associated with
	State  CcbState           // The state of the approval
}

// Stringify function for CcbState
func (s CcbState) String() string {
	switch s {
	case AddedCcbState:
		return "Added"
	case ApprovedCcbState:
		return "Approved"
	case BlockedCcbState:
		return "Blocked"
	case RemovedCcbState:
		return "Removed"
	default:
		return "UNKNOWN"
	}
}

// ccbMembers holds a slice of users who are in the master CCB member list.
var ccbMembers []entity.Id

// readCcbMembers attempts to read the ccb group out of the current repository and store it in ccbMembers
func readCcbMembers() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get the current working directory: %q", err)
	}

	repo, err := repository.NewGitRepo(cwd, []repository.ClockLoader{ClockLoader})
	if err == repository.ErrNotARepo {
		return fmt.Errorf("must be run from within a git repo")
	}

	ccbData, err := config.GetConfig(repo, "ccb")
	if err != nil {
		return fmt.Errorf("unable to read ccb config: %q", err)
	}

	// Parse the CCB member list from the configuration. Configurations must be of the form "map[string]interface{}" so
	// is stored as {"ccbMembers" : ["<user id1>", "<user id2>", "..."]}.
	ccbMembersTemp := make(map[string][]entity.Id)

	err = json.Unmarshal(ccbData, &ccbMembersTemp)
	if err != nil {
		return fmt.Errorf("unable to load ccb: %q", err)
	}

	var present bool
	ccbMembers, present = ccbMembersTemp["ccbMembers"]
	if !present {
		return errors.New("unexpected ccb config format")
	}

	return nil
}

// IsCcbMember returns a flag indicating if the user is a ccb member, as defined in the repository configuration
func IsCcbMember(user identity.Interface) (bool, error) {
	if ccbMembers == nil {
		if err := readCcbMembers(); err != nil {
			return false, err
		}
	}
	for _, c := range ccbMembers {
		if c == user.Id() {
			return true, nil
		}
	}
	return false, nil
}

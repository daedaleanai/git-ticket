package bug

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
)

// CcbState represents the status of a CCB group member with respect to a ticket
type CcbState int

const (
	_                CcbState = iota
	AddedCcbState             // added to the ticket, but not set a state
	ApprovedCcbState          // approved the ticket
	BlockedCcbState           // blocked the ticket
	RemovedCcbState           // removed from the ticket
)

// CcbInfo is stored in a ticket history every time a user is added or removed,
// or has approved or blocked the ticket
type CcbInfo struct {
	User  identity.Interface
	State CcbState
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

// ccbMembers holds a map of users who are in the master CCB member list, which is defined in the git ticket
// configuration. Configurations must be of the form "map[string]interface{}" so is stored as {"<user id>":1}.
var ccbMembers map[entity.Id]int

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

	ccbMembersTemp := make(map[entity.Id]int)

	err = json.Unmarshal(ccbData, &ccbMembersTemp)
	if err != nil {
		return fmt.Errorf("unable to load ccb: %q", err)
	}

	ccbMembers = ccbMembersTemp

	return nil
}

// IsCcbMember returns a flag indicating if the user is a ccb member, as defined in the repository configuration
func IsCcbMember(user identity.Interface) (bool, error) {
	if ccbMembers == nil {
		if err := readCcbMembers(); err != nil {
			return false, err
		}
	}

	_, present := ccbMembers[user.Id()]

	return present, nil
}

package bug

import (
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/colors"
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

// CcbInfoByStatus provides functions to fulfill the sort interface
type CcbInfoByStatus []CcbInfo

func (a CcbInfoByStatus) Len() int           { return len(a) }
func (a CcbInfoByStatus) Less(i, j int) bool { return a[i].Status < a[j].Status }
func (a CcbInfoByStatus) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

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

// Colored strings function for CcbState
func (s CcbState) ColorString() string {
	switch s {
	case AddedCcbState:
		return colors.Blue("Added")
	case ApprovedCcbState:
		return colors.Green("Approved")
	case BlockedCcbState:
		return colors.Red("Blocked")
	case RemovedCcbState:
		return "Removed"
	default:
		return "UNKNOWN"
	}
}

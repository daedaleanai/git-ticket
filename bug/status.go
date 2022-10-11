package bug

import (
	"fmt"
	"strings"
)

type Status int

const (
	_ Status = iota
	ProposedStatus
	VettedStatus
	InProgressStatus
	InReviewStatus
	ReviewedStatus
	AcceptedStatus
	MergedStatus
	DoneStatus
	RejectedStatus
)

const FirstStatus = ProposedStatus
const LastStatus = RejectedStatus
const NumStatuses = 8

func (s Status) String() string {
	switch s {
	case ProposedStatus:
		return "proposed"
	case VettedStatus:
		return "vetted"
	case InProgressStatus:
		return "inprogress"
	case InReviewStatus:
		return "inreview"
	case ReviewedStatus:
		return "reviewed"
	case AcceptedStatus:
		return "accepted"
	case MergedStatus:
		return "merged"
	case DoneStatus:
		return "done"
	case RejectedStatus:
		return "rejected"
	default:
		return "unknown status"
	}
}

func (s Status) Action() string {
	switch s {
	case ProposedStatus:
		return "set PROPOSED"
	case VettedStatus:
		return "set VETTED"
	case InProgressStatus:
		return "set IN PROGRESS"
	case InReviewStatus:
		return "set IN REVIEW"
	case ReviewedStatus:
		return "set REVIEWED"
	case AcceptedStatus:
		return "set ACCEPTED"
	case MergedStatus:
		return "set MERGED"
	case DoneStatus:
		return "set DONE"
	case RejectedStatus:
		return "set REJECTED"
	default:
		return "unknown status"
	}
}

func StatusFromString(str string) (Status, error) {
	cleaned := strings.ToLower(strings.TrimSpace(str))

	switch cleaned {
	case "proposed":
		return ProposedStatus, nil
	case "vetted":
		return VettedStatus, nil
	case "inprogress":
		return InProgressStatus, nil
	case "inreview":
		return InReviewStatus, nil
	case "reviewed":
		return ReviewedStatus, nil
	case "accepted":
		return AcceptedStatus, nil
	case "merged":
		return MergedStatus, nil
	case "done":
		return DoneStatus, nil
	case "rejected":
		return RejectedStatus, nil
	default:
		return 0, fmt.Errorf("unknown status: %s", cleaned)
	}
}

func (s Status) Validate() error {
	if s < FirstStatus || s > LastStatus {
		return fmt.Errorf("invalid")
	}

	return nil
}

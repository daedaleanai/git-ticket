package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/colors"
)

type ChecklistState int

const (
	TBD ChecklistState = iota
	Passed
	Failed
	NotApplicable
)

type ChecklistQuestion struct {
	Question string
	Comment  string
	State    ChecklistState
}
type ChecklistSection struct {
	Title     string
	Questions []ChecklistQuestion
}
type Checklist struct {
	Label      Label
	Title      string
	Deprecated string
	Sections   []ChecklistSection
}

type ChecklistConfig map[Label]Checklist

// LoadChecklistConfig attempts to read the checklists configuration out of the
// current repository and use it to initialise the checklistStore
func LoadChecklistConfig(repo repository.ClockedRepo) (ChecklistConfig, error) {
	checklistData, err := GetConfig(repo, "checklists")
	if err != nil {
		if _, ok := err.(*NotFoundError); ok {
			return ChecklistConfig{}, nil
		}
		return nil, fmt.Errorf("unable to read checklists config: %q", err)
	}

	checklistStore := make(map[Label]Checklist)

	err = json.Unmarshal(checklistData, &checklistStore)
	if err != nil {
		return nil, fmt.Errorf("unable to load checklists: %q", err)
	}

	return checklistStore, nil
}

// GetChecklist returns a Checklist template out of the store
func (c ChecklistConfig) GetChecklist(label Label) (Checklist, error) {
	cl, present := c[label]

	if !present {
		return cl, fmt.Errorf("invalid checklist %s", label)
	}

	return cl, nil
}

// GetChecklistLabels returns a slice of all the available checklist labels
func (c ChecklistConfig) GetChecklistLabels() []Label {

	var labels []Label
	for _, cl := range c {
		if cl.Deprecated != "" {
			continue
		}
		labels = append(labels, cl.Label)
	}
	return labels
}

func (s ChecklistState) String() string {
	switch s {
	case TBD:
		return "TBD"
	case Passed:
		return "PASSED"
	case Failed:
		return "FAILED"
	case NotApplicable:
		return "NA"
	default:
		return "UNKNOWN"
	}
}

func (s ChecklistState) ShortString() string {
	switch s {
	case TBD:
		return "TBD"
	case Passed:
		return "P"
	case Failed:
		return "F"
	case NotApplicable:
		return "NA"
	default:
		return "UNKNOWN"
	}
}

func (s ChecklistState) ColorString() string {
	switch s {
	case TBD:
		return colors.Blue("TBD")
	case Passed:
		return colors.Green("PASSED")
	case Failed:
		return colors.Red("FAILED")
	case NotApplicable:
		return "NA"
	default:
		return "UNKNOWN"
	}
}

func (s ChecklistState) Validate() error {
	if s < TBD || s > NotApplicable {
		return fmt.Errorf("invalid")
	}

	return nil
}

// CompoundState returns an overall state for the checklist given the state of
// each of the questions. If any of the questions are Failed then the checklist
// Failed, else if any are TBD it's TBD, else it's Passed
func (c Checklist) CompoundState() ChecklistState {
	var tbdCount, failedCount int
	for _, s := range c.Sections {
		for _, q := range s.Questions {
			switch q.State {
			case TBD:
				tbdCount++
			case Failed:
				failedCount++
			}
		}
	}
	// If at least one question has Failed then return that state
	if failedCount > 0 {
		return Failed
	}
	// None have Failed, but if any are still TBD return that
	if tbdCount > 0 {
		return TBD
	}
	// None Failed or TBD, all questions are NotApplicable or Passed, return Passed
	return Passed
}

func (c Checklist) String() string {
	result := fmt.Sprintf("%s [%s]\n", c.Title, c.CompoundState().ColorString())

	for sn, s := range c.Sections {
		result = result + fmt.Sprintf("#### %s ####\n", s.Title)
		for qn, q := range s.Questions {
			result = result + fmt.Sprintf("(%d.%d) %s [%s]\n", sn+1, qn+1, q.Question, q.State.ColorString())
			if q.Comment != "" {
				result = result + fmt.Sprintf("# %s\n", strings.Replace(q.Comment, "\n", "\n# ", -1))
			}
		}
	}
	return result
}

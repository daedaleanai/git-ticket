package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/entity"
)

func TestSnapshot_GetChecklistCompoundStates(t *testing.T) {
	// Create an initial snapshot, one checklist reviewed by two people, all passed.
	snapshot := Snapshot{
		Labels: []Label{"checklist:XYZ"},
		Checklists: map[Label]map[entity.Id]ChecklistSnapshot{
			"checklist:XYZ": {
				"123": {
					Checklist: config.Checklist{
						Sections: []config.ChecklistSection{
							{
								Questions: []config.ChecklistQuestion{
									{State: config.Passed},
									{State: config.Passed},
								},
							},
							{
								Questions: []config.ChecklistQuestion{
									{State: config.Passed},
									{State: config.Passed},
								},
							},
						},
					},
					LastEdit: time.Time{},
				},
				"456": ChecklistSnapshot{
					Checklist: config.Checklist{
						Sections: []config.ChecklistSection{
							{
								Questions: []config.ChecklistQuestion{
									{State: config.Passed},
									{State: config.Passed},
								},
							},
							{
								Questions: []config.ChecklistQuestion{
									{State: config.Passed},
									{State: config.Passed},
								},
							},
						},
					},
					LastEdit: time.Time{},
				},
			},
		},
	}

	assert.Equal(t, snapshot.GetChecklistCompoundStates(), map[Label]config.ChecklistState{"checklist:XYZ": config.Passed})

	// one review has left an answer TBD, should still be overall pass
	snapshot.Checklists["checklist:XYZ"]["456"].Checklist.Sections[0].Questions[1].State = config.TBD
	assert.Equal(t, snapshot.GetChecklistCompoundStates(), map[Label]config.ChecklistState{"checklist:XYZ": config.Passed})

	// both reviewers have left an answer TBD, should be overall TBD
	snapshot.Checklists["checklist:XYZ"]["123"].Checklist.Sections[1].Questions[1].State = config.TBD
	assert.Equal(t, snapshot.GetChecklistCompoundStates(), map[Label]config.ChecklistState{"checklist:XYZ": config.TBD})

	// one review has left an answer failed, should be overall fail
	snapshot.Checklists["checklist:XYZ"]["456"].Checklist.Sections[0].Questions[1].State = config.Failed
	assert.Equal(t, snapshot.GetChecklistCompoundStates(), map[Label]config.ChecklistState{"checklist:XYZ": config.Failed})

	// the default state for an unreviewed checklist is TBD
	snapshot.Labels = append(snapshot.Labels, "checklist:ABC")
	assert.Equal(t, snapshot.GetChecklistCompoundStates(), map[Label]config.ChecklistState{"checklist:XYZ": config.Failed, "checklist:ABC": config.TBD})
}

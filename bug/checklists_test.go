package bug

import (
	"testing"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/stretchr/testify/assert"
)

func TestChecklists_ChecklistCompoundState(t *testing.T) {
	testChecklist := config.Checklist{Label: "XYZ",
		Title: "XYZ Checklist",
		Sections: []config.ChecklistSection{
			config.ChecklistSection{Title: "ABC",
				Questions: []config.ChecklistQuestion{
					config.ChecklistQuestion{Question: "1?", State: config.Passed},
					config.ChecklistQuestion{Question: "2?", State: config.Passed},
					config.ChecklistQuestion{Question: "3?", State: config.Passed},
				},
			},
			config.ChecklistSection{Title: "DEF",
				Questions: []config.ChecklistQuestion{
					config.ChecklistQuestion{Question: "4?", State: config.Passed},
					config.ChecklistQuestion{Question: "5?", State: config.Passed},
					config.ChecklistQuestion{Question: "6?", State: config.Passed},
				},
			},
		},
	}
	assert.Equal(t, testChecklist.CompoundState(), config.Passed)

	testChecklist.Sections[0].Questions[0].State = config.NotApplicable
	assert.Equal(t, testChecklist.CompoundState(), config.Passed)

	testChecklist.Sections[0].Questions[1].State = config.TBD
	assert.Equal(t, testChecklist.CompoundState(), config.TBD)

	testChecklist.Sections[0].Questions[2].State = config.Failed
	assert.Equal(t, testChecklist.CompoundState(), config.Failed)
}

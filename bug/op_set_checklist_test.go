package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/stretchr/testify/assert"
)

func TestOpSetChecklist_SetChecklist(t *testing.T) {
	var rene = identity.NewBare("René Descarte", "rene@descartes.fr")
	unix := time.Now().Unix()
	bug1 := NewBug()

	before, err := SetChecklist(bug1, rene, unix, config.Checklist{Label: "123",
		Title: "123 Checklist",
		Sections: []config.ChecklistSection{
			config.ChecklistSection{Title: "Section 1",
				Questions: []config.ChecklistQuestion{
					config.ChecklistQuestion{Question: "1?"},
					config.ChecklistQuestion{Question: "2?"},
					config.ChecklistQuestion{Question: "3?"},
				},
			},
			config.ChecklistSection{Title: "Section 2",
				Questions: []config.ChecklistQuestion{
					config.ChecklistQuestion{Question: "4?"},
					config.ChecklistQuestion{Question: "5?"},
					config.ChecklistQuestion{Question: "6?"},
				},
			},
		},
	})
	assert.NoError(t, err)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after SetChecklistOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the IDs
	before.Id()
	rene.Id()

	assert.Equal(t, before, &after)
}

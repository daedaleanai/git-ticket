package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := &OperationPack{}

	rene := identity.NewBare("René Descartes", "rene@descartes.fr")
	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)
	setTitleOp := NewSetTitleOp(rene, time.Now().Unix(), "title2", "title1")
	addCommentOp := NewAddCommentOp(rene, time.Now().Unix(), "message2", nil)
	setStatusOp := NewSetStatusOp(rene, time.Now().Unix(), MergedStatus)
	labelChangeOp := NewLabelChangeOperation(rene, time.Now().Unix(), []Label{"added"}, []Label{"removed"})
	setChecklistOp := NewSetChecklistOp(rene, time.Now().Unix(), config.Checklist{Label: "123",
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

	mickey := identity.NewBare("Mickey Mouse", "mm@disney.com")
	setAssigneeOp := NewSetAssigneeOp(rene, time.Now().Unix(), mickey)

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)
	opp.Append(setStatusOp)
	opp.Append(labelChangeOp)
	opp.Append(setChecklistOp)
	opp.Append(setAssigneeOp)

	opMeta := NewSetTitleOp(rene, time.Now().Unix(), "title3", "title2")
	opMeta.SetMetadata("key", "value")
	opp.Append(opMeta)

	assert.Equal(t, 1, len(opMeta.Metadata))

	opFile := NewAddCommentOp(rene, time.Now().Unix(), "message", []repository.Hash{
		"abcdef",
		"ghijkl",
	})
	opp.Append(opFile)

	assert.Equal(t, 2, len(opFile.Files))

	data, err := json.Marshal(opp)
	assert.NoError(t, err)

	var opp2 *OperationPack
	err = json.Unmarshal(data, &opp2)
	assert.NoError(t, err)

	ensureIDs(t, opp)
	mickey.Id()

	assert.Equal(t, opp, opp2)
}

func ensureIDs(t *testing.T, opp *OperationPack) {
	for _, op := range opp.Operations {
		id := op.Id()
		require.NoError(t, id.Validate())
		id = op.GetAuthor().Id()
		require.NoError(t, id.Validate())
	}
}

package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/daedaleanai/git-ticket/identity"
	"github.com/stretchr/testify/assert"
)

func TestSetCcbSerialize(t *testing.T) {
	var rene = identity.NewBare("René Descartes", "rene@descartes.fr")
	var mickey = identity.NewBare("Mickey Mouse", "mm@disney.com")
	unix := time.Now().Unix()
	before := NewSetCcbOp(rene, unix, mickey, VettedStatus, ApprovedCcbState)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after SetCcbOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the IDs
	before.Id()
	rene.Id()
	mickey.Id()

	assert.Equal(t, before, &after)
}

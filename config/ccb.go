package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
)

type CcbConfig []entity.Id

// LoadCcbConfig attempts to read the ccb group out of the current repository and store it in ccbMembers
func LoadCcbConfig(repo repository.ClockedRepo) (CcbConfig, error) {
	ccbData, err := GetConfig(repo, "ccb")
	if err != nil {
		if _, ok := err.(*NotFoundError); ok {
			return CcbConfig{}, nil
		}
		return nil, fmt.Errorf("unable to read ccb config: %q", err)
	}

	// Parse the CCB member list from the configuration. Configurations must be of the form "map[string]interface{}" so
	// is stored as {"ccbMembers" : ["<user id1>", "<user id2>", "..."]}.
	ccbMembersTemp := make(map[string][]entity.Id)

	err = json.Unmarshal(ccbData, &ccbMembersTemp)
	if err != nil {
		return nil, fmt.Errorf("unable to load ccb: %q", err)
	}

	var present bool
	ccbMembers, present := ccbMembersTemp["ccbMembers"]
	if !present {
		return nil, errors.New("unexpected ccb config format")
	}

	return ccbMembers, nil
}

// IsCcbMember returns a flag indicating if the user is a ccb member, as defined in the repository configuration
func (c CcbConfig) IsCcbMember(user identity.Interface) (bool, error) {
	for _, c := range c {
		if c == user.Id() {
			return true, nil
		}
	}
	return false, nil
}

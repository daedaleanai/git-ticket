package config

import (
	"encoding/json"
	"fmt"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
)

type CcbMember struct {
	Name string
	Id   entity.Id
}

type CcbTeam struct {
	Name    string
	Members []CcbMember
}

type CcbConfig []CcbTeam

// readCcbMembers attempts to read the ccb group out of the current repository and store it in ccbTeams
func LoadCcbConfig(repo repository.ClockedRepo) (CcbConfig, error) {
	ccbData, err := GetConfig(repo, "ccb-teams")
	if err != nil {
		if _, ok := err.(*NotFoundError); ok {
			return CcbConfig{}, nil
		}
		return nil, fmt.Errorf("unable to read ccb config: %q", err)
	}

	type ccbTeamsJson map[string][]CcbMember

	type config struct {
		Teams ccbTeamsJson `json:"ccbTeams"`
	}

	ccbTeamsTemp := config{}

	err = json.Unmarshal(ccbData, &ccbTeamsTemp)
	if err != nil {
		return nil, fmt.Errorf("unable to load ccb: %q", err)
	}

	resolver := identity.NewSimpleResolver(repo)
	ccbTeams := []CcbTeam{}
	for name, members := range ccbTeamsTemp.Teams {
		ccbTeams = append(ccbTeams, CcbTeam{
			Name:    name,
			Members: members,
		})

		for _, member := range members {
			id, err := resolver.ResolveIdentity(member.Id)
			if err != nil {
				return nil, err
			}

			if id.DisplayName() != member.Name {
				return nil, fmt.Errorf("CCB Config is incorrect. User name does not match their ID: Got %q. Expected %q", member.Name, id.DisplayName())
			}
		}
	}

	return ccbTeams, nil
}

// IsCcbMember returns a flag indicating if the user is a ccb member, as defined in the repository configuration
func (c CcbConfig) IsCcbMember(user identity.Interface) (bool, error) {
	for _, team := range c {
		for _, member := range team.Members {
			if member.Id == user.Id() {
				return true, nil
			}
		}
	}
	return false, nil
}

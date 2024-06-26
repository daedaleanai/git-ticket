package config

import "github.com/daedaleanai/git-ticket/repository"

type ConfigCache struct {
	CcbConfig
	LabelConfig
	ChecklistConfig
}

func LoadConfigCache(repo repository.ClockedRepo) (*ConfigCache, error) {
	ccbConfig, err := LoadCcbConfig(repo)
	if err != nil {
		return nil, err
	}

	labelConfig, err := LoadLabelConfig(repo)
	if err != nil {
		return nil, err
	}

	checklistConfig, err := LoadChecklistConfig(repo)
	if err != nil {
		return nil, err
	}

	return &ConfigCache{
		ccbConfig,
		*labelConfig,
		checklistConfig,
	}, nil
}

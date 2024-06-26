package bug

import (
	"fmt"
	"strings"
	"time"

	"github.com/daedaleanai/git-ticket/config"
)

type ChecklistSnapshot struct {
	config.Checklist
	LastEdit time.Time
}

func StateFromString(str string) (config.ChecklistState, error) {
	cleaned := strings.ToLower(strings.TrimSpace(str))

	if strings.HasPrefix("tbd", cleaned) {
		return config.TBD, nil
	} else if strings.HasPrefix("passed", cleaned) {
		return config.Passed, nil
	} else if strings.HasPrefix("failed", cleaned) {
		return config.Failed, nil
	} else if strings.HasPrefix("na", cleaned) {
		return config.NotApplicable, nil
	}

	return 0, fmt.Errorf("unknown state")
}

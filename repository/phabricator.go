package repository

import (
	"fmt"
	"os"

	"github.com/thought-machine/gonduit"
	"github.com/thought-machine/gonduit/core"
)

// getPhabConfig returns the Phabricator URL and API token from the repository config
func getPhabConfig() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("unable to get the current working directory: %q", err)
	}

	repo, err := NewGitRepoNoInit(cwd)
	if err == ErrNotARepo {
		return "", "", fmt.Errorf("must be run from within a git repo")
	}

	var phabUrl string
	if phabUrl, err = repo.LocalConfig().ReadString("phabricator.url"); err != nil {
		if phabUrl, err = repo.GlobalConfig().ReadString("phabricator.url"); err != nil {
			return "", "", fmt.Errorf("No Phabricator URL set. Set it with:\ngit config --global --replace-all phabricator.url <URL of phabricator server>")
		}
	}

	var apiToken string
	if apiToken, err = repo.LocalConfig().ReadString("phabricator.api-token"); err != nil {
		if apiToken, err = repo.GlobalConfig().ReadString("phabricator.api-token"); err != nil {
			msg := `No Phabricator API token set. Please go to
	%s/settings/user/<YOUR_USERNAME_HERE>/page/apitokens/
click on <Generate API Token>, and then paste the token into this command
	git config --global --replace-all phabricator.api-token <PASTE_TOKEN_HERE>`
			return phabUrl, "", fmt.Errorf(msg, phabUrl)
		}
	}

	return phabUrl, apiToken, nil
}

// GetPhabClient returns the connection ready to be queried. Must be called
// within a git repo which has the Phabricator URL and conduit API token set
// in the git config.
func GetPhabClient() (*gonduit.Conn, error) {
	phabUrl, apiToken, err := getPhabConfig()
	if err != nil {
		return nil, err
	}

	return gonduit.Dial(phabUrl, &core.ClientOptions{APIToken: apiToken})
}

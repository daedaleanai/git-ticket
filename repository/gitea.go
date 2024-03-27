package repository

import (
	"fmt"
	"os"

	"code.gitea.io/sdk/gitea"
)

// GetGiteaConfig returns the Gitea URL and API token from the repository config
func GetGiteaConfig() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("unable to get the current working directory: %q", err)
	}

	repo, err := NewGitRepoNoInit(cwd)
	if err == ErrNotARepo {
		return "", "", fmt.Errorf("must be run from within a git repo")
	}

	var giteaUrl string
	if giteaUrl, err = repo.LocalConfig().ReadString("gitea.url"); err != nil {
		if giteaUrl, err = repo.GlobalConfig().ReadString("gitea.url"); err != nil {
			return "", "", fmt.Errorf("No Gitea URL set. Set it with:\ngit config --global --replace-all gitea.url <URL of gitea server>")
		}
	}

	var apiToken string
	if apiToken, err = repo.LocalConfig().ReadString("gitea.api-token"); err != nil {
		if apiToken, err = repo.GlobalConfig().ReadString("gitea.api-token"); err != nil {
			msg := `No Gitea API token set. Please go to
	%s/user/settings/applications enter a token name,
click on <Generate Token> and then paste the token into this command
	git config --global --replace-all gitea.api-token <PASTE_TOKEN_HERE>`
			return giteaUrl, "", fmt.Errorf(msg, giteaUrl)
		}
	}

	return giteaUrl, apiToken, nil
}

// GetGiteaClient returns the connection ready to be queried. Must be called
// within a git repo which has the Gitea URL and API token set
// in the git config.
func GetGiteaClient() (*gitea.Client, error) {
	giteaUrl, apiToken, err := GetGiteaConfig()
	if err != nil {
		return nil, err
	}

	return gitea.NewClient(giteaUrl, gitea.SetToken(apiToken))
}

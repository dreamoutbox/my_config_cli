package cmd

import (
	"fmt"
	"strings"

	"github.com/dreamoutbox/my_config_cli/internal/config"
)

// resolveAndPersistRepo determines the repo URL from flag or stored settings,
// persisting the flag value to settings.json if provided.
func resolveAndPersistRepo(flagVal string) (string, error) {
	settings, err := config.LoadSettings()
	if err != nil {
		return "", fmt.Errorf("loading settings: %w", err)
	}

	trimmedFlag := strings.TrimSpace(flagVal)
	if trimmedFlag != "" {
		if settings.RepoURL != trimmedFlag {
			settings.RepoURL = trimmedFlag
			if err := config.SaveSettings(settings); err != nil {
				return "", fmt.Errorf("persisting repo url to settings: %w", err)
			}
		}
		return trimmedFlag, nil
	}

	if strings.TrimSpace(settings.RepoURL) != "" {
		return settings.RepoURL, nil
	}

	return "", fmt.Errorf("no repository configured; specify --repo <repo_url>")
}

package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/command"
)

func (p *Plugin) getAutocompleteIconData() string {
	iconData, err := command.GetIconData(p.API, "public/clickup-autocomplete.svg")
	if err == nil && iconData != "" {
		return iconData
	}

	// Fallback to embedded icon if bundle file is unavailable during development.
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return ""
	}

	icon, err := os.ReadFile(filepath.Join(bundlePath, "public", "clickup-autocomplete.svg"))
	if err != nil {
		icon, err = os.ReadFile(filepath.Join(bundlePath, "public", "clickup-icon.svg"))
		if err != nil {
			return ""
		}
	}

	return fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString(icon))
}

func (p *Plugin) getCommandIconURL() string {
	return p.getPluginURL() + "/public/clickup.png"
}

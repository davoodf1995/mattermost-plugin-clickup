package main

import (
	"encoding/base64"
	_ "embed"
)

//go:embed assets/clickup-icon.svg
var clickupIconSVG []byte

func getAutocompleteIconData() string {
	if len(clickupIconSVG) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(clickupIconSVG)
}

func (p *Plugin) getCommandIconURL() string {
	return p.getPluginURL() + "/public/clickup.png"
}

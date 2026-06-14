package main

import (
	"strings"
)

func (p *Plugin) getSiteURL() string {
	config := p.API.GetConfig()
	if config == nil || config.ServiceSettings.SiteURL == nil {
		return ""
	}
	return strings.TrimSuffix(*config.ServiceSettings.SiteURL, "/")
}

func (p *Plugin) getPluginURL() string {
	siteURL := p.getSiteURL()
	if siteURL == "" {
		return ""
	}
	return siteURL + "/plugins/" + manifest.Id
}

func dialogValue(submission map[string]any, key string) string {
	if submission == nil {
		return ""
	}
	value, ok := submission[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func (p *Plugin) listAllKVKeys() []string {
	var keys []string
	page := 0
	perPage := 100

	for {
		batch, appErr := p.API.KVList(page, perPage)
		if appErr != nil || len(batch) == 0 {
			break
		}
		keys = append(keys, batch...)
		if len(batch) < perPage {
			break
		}
		page++
	}

	return keys
}

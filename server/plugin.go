package main

import (
	"net/http"
	"sync"

	"github.com/mattermost/mattermost/server/public/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex
	configuration     *configuration

	clickup      *ClickUpClient
	botUserID    string
	reminderStop chan struct{}
}

func (p *Plugin) getClickUpClient() (*ClickUpClient, error) {
	config := p.getConfiguration()
	if config.ClickUpAPIToken == "" {
		return nil, errNotConfigured
	}

	if p.clickup == nil || p.clickup.token != config.ClickUpAPIToken {
		p.clickup = NewClickUpClient(config.ClickUpAPIToken)
	}

	return p.clickup, nil
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/webhook":
		p.handleClickUpWebhook(w, r)
	case "/dialog/create-task":
		p.handleCreateTaskDialog(w, r)
	case "/dialog/assign-task":
		p.handleAssignTaskDialog(w, r)
	case "/api/tasks":
		p.handleAPITasks(w, r)
	case "/api/link":
		p.handleAPILink(w, r)
	case "/action/create-task":
		p.handlePostActionCreateTask(w, r)
	default:
		http.NotFound(w, r)
	}
}

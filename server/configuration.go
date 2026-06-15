package main

import (
	"reflect"

	"github.com/pkg/errors"
)

type configuration struct {
	ClickUpAPIToken     string
	ClickUpTeamID       string
	DefaultListID       string
	EnableReminders     bool
	ReminderHoursBefore int
	WebhookSecret       string
}

func (c *configuration) Clone() *configuration {
	clone := *c
	return &clone
}

func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{EnableReminders: true, ReminderHoursBefore: 24}
	}

	return p.configuration
}

func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}
		// Clone to avoid accidental shared mutation; never panic during config reload.
		clone := configuration.Clone()
		p.configuration = clone
		return
	}

	p.configuration = configuration
}

func (p *Plugin) OnConfigurationChange() error {
	configuration := new(configuration)

	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	if configuration.ReminderHoursBefore <= 0 {
		configuration.ReminderHoursBefore = 24
	}

	p.setConfiguration(configuration)

	return nil
}

package main

import (
	"fmt"

	"github.com/blang/semver/v4"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

const minimumServerVersion = "11.0.0"

func (p *Plugin) checkServerVersion() error {
	serverVersion, err := semver.Parse(p.API.GetServerVersion())
	if err != nil {
		return errors.Wrap(err, "failed to parse server version")
	}

	r := semver.MustParseRange(">=" + minimumServerVersion)
	if !r(serverVersion) {
		return fmt.Errorf("this plugin requires Mattermost v%s or later", minimumServerVersion)
	}

	return nil
}

func (p *Plugin) ensureBot() error {
	bot := &model.Bot{
		Username:    "clickup",
		DisplayName:   "ClickUp",
		Description:   "ClickUp integration bot",
		OwnerId:       p.API.GetPluginID(),
	}
	userID, appErr := p.API.EnsureBotUser(bot)
	if appErr != nil {
		return appErr
	}
	p.botUserID = userID
	return nil
}

func (p *Plugin) OnActivate() error {
	if err := p.checkServerVersion(); err != nil {
		return err
	}

	if err := p.ensureBot(); err != nil {
		return errors.Wrap(err, "failed to ensure bot user")
	}

	if err := p.registerCommands(); err != nil {
		return errors.Wrap(err, "failed to register commands")
	}

	if err := p.ensureWebhook(); err != nil {
		p.API.LogWarn("failed to register ClickUp webhook", "error", err.Error())
	}

	p.startReminderJob()

	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.stopReminderJob()
	return p.removeWebhook()
}

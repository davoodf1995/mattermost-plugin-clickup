package main

import "github.com/pkg/errors"

var (
	errNotConfigured = errors.New("clickup is not configured: set API token in plugin settings")
	errNoLinkedList  = errors.New("this channel is not linked to a ClickUp list. Use `/clickup link <list_url_or_id>`")
)

const clickUpAPIBase = "https://api.clickup.com/api/v2"

const (
	kvChannelListPrefix = "channel_list_"
	kvWebhookID         = "clickup_webhook_id"
	kvReminderSent      = "reminder_sent_"
)

const commandTrigger = "clickup"

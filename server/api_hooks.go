package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) startReminderJob() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		p.runReminderCheck()

		for range ticker.C {
			p.runReminderCheck()
		}
	}()
}

func (p *Plugin) stopReminderJob() {}

func (p *Plugin) runReminderCheck() {
	config := p.getConfiguration()
	if !config.EnableReminders {
		return
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return
	}

	keys := p.listAllKVKeys()

	now := time.Now()
	horizon := now.Add(time.Duration(config.ReminderHoursBefore) * time.Hour)

	for _, key := range keys {
		if len(key) <= len(kvChannelListPrefix) || key[:len(kvChannelListPrefix)] != kvChannelListPrefix {
			continue
		}

		channelID := key[len(kvChannelListPrefix):]
		link, err := p.getChannelLink(channelID)
		if err != nil {
			continue
		}

		tasks, err := client.GetListTasks(link.ListID, false)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			if task.DueDate == nil {
				continue
			}

			dueMs, err := task.DueDate.Int64()
			if err != nil {
				continue
			}

			due := time.UnixMilli(dueMs)
			if due.Before(now) || due.After(horizon) {
				continue
			}

			reminderKey := kvReminderSent + task.ID
			if sent, _ := p.API.KVGet(reminderKey); len(sent) > 0 {
				continue
			}

			message := fmt.Sprintf(":alarm_clock: Reminder: ClickUp task **[ %s ](%s)** is due **%s** — %s",
				task.Name, task.URL, due.Format("2006-01-02 15:04"), assigneeNames(task))

			p.postToChannel(channelID, message)
			p.remindAssignees(task, due)
			_ = p.API.KVSet(reminderKey, []byte(strconv.FormatInt(time.Now().Unix(), 10)))
		}
	}
}

func (p *Plugin) remindAssignees(task ClickUpTask, due time.Time) {
	for _, assignee := range task.Assignees {
		mmUser, err := p.findMattermostUserByEmail(assignee.Email)
		if err != nil {
			continue
		}

		channel, appErr := p.API.GetDirectChannel(p.botUserID, mmUser.Id)
		if appErr != nil {
			continue
		}

		post := &model.Post{
			UserId:    p.botUserID,
			ChannelId: channel.Id,
			Message: fmt.Sprintf(":alarm_clock: Reminder: **[ %s ](%s)** is due **%s**",
				task.Name, task.URL, due.Format("2006-01-02 15:04")),
		}
		p.API.CreatePost(post)
	}
}

func (p *Plugin) handleAPITasks(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		http.Error(w, "channel_id required", http.StatusBadRequest)
		return
	}

	if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionReadChannel) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	client, err := p.getClickUpClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	listID, err := p.resolveListID(channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tasks, err := client.GetListTasks(listID, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	link, _ := p.getChannelLink(channelID)
	resp := map[string]interface{}{
		"tasks": tasks,
		"link":  link,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (p *Plugin) handleAPILink(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		http.Error(w, "channel_id required", http.StatusBadRequest)
		return
	}

	if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionReadChannel) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	link, err := p.getChannelLink(channelID)
	if err != nil {
		link = &channelLink{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(link)
}

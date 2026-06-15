package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) ensureWebhook() error {
	config := p.getConfiguration()
	if config.ClickUpAPIToken == "" || config.ClickUpTeamID == "" || config.WebhookSecret == "" {
		return errNotConfigured
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return err
	}

	existing, appErr := p.API.KVGet(kvWebhookID)
	if appErr == nil && len(existing) > 0 {
		return nil
	}

	endpoint := p.getPluginURL() + "/webhook"
	webhookID, err := client.CreateWebhook(config.ClickUpTeamID, endpoint, config.WebhookSecret)
	if err != nil {
		return err
	}

	if appErr := p.API.KVSet(kvWebhookID, []byte(webhookID)); appErr != nil {
		return appErr
	}
	return nil
}

func (p *Plugin) removeWebhook() error {
	data, appErr := p.API.KVGet(kvWebhookID)
	if appErr != nil || len(data) == 0 {
		return nil
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return err
	}

	_ = client.DeleteWebhook(string(data))
	_ = p.API.KVDelete(kvWebhookID)
	return nil
}

func (p *Plugin) handleClickUpWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := p.getConfiguration()
	if config.WebhookSecret != "" && r.Header.Get("X-Signature") == "" {
		// ClickUp may send signature in header; accept if secret matches body check when available
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var payload ClickUpWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go p.processWebhookEvent(payload)

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) processWebhookEvent(payload ClickUpWebhookPayload) {
	if payload.TaskID == "" {
		return
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return
	}

	task, err := client.GetTask(payload.TaskID)
	if err != nil {
		p.API.LogWarn("webhook: failed to load task", "task_id", payload.TaskID, "error", err.Error())
		return
	}

	listID := task.List.ID
	if listID == "" {
		return
	}

	channels := p.findChannelsForList(listID)
	if len(channels) == 0 {
		return
	}

	message := p.formatWebhookMessage(payload, task)
	for _, channelID := range channels {
		p.postToChannel(channelID, message)
	}

	p.notifyAssignees(payload, task)
}

func (p *Plugin) formatWebhookMessage(payload ClickUpWebhookPayload, task *ClickUpTask) string {
	actor := "Someone"
	if len(payload.History) > 0 && payload.History[0].User.Username != "" {
		actor = payload.History[0].User.Username
	}

	switch payload.Event {
	case "taskCreated":
		return fmt.Sprintf(":new: **%s** created ClickUp task **[ %s ](%s)** — %s",
			actor, task.Name, task.URL, assigneeNames(*task))
	case "taskStatusUpdated":
		return fmt.Sprintf(":arrows_counterclockwise: **%s** updated status of **[ %s ](%s)** to **%s**",
			actor, task.Name, task.URL, taskStatusLabel(*task))
	case "taskAssigneeUpdated":
		return fmt.Sprintf(":bust_in_silhouette: **%s** updated assignees on **[ %s ](%s)** — %s",
			actor, task.Name, task.URL, assigneeNames(*task))
	case "taskDueDateUpdated":
		due := formatDueDate(task.DueDate)
		if due == "" {
			due = "none"
		}
		return fmt.Sprintf(":calendar: **%s** set due date on **[ %s ](%s)** to **%s**",
			actor, task.Name, task.URL, due)
	case "taskCommentPosted":
		return fmt.Sprintf(":speech_balloon: **%s** commented on **[ %s ](%s)**",
			actor, task.Name, task.URL)
	case "taskPriorityUpdated":
		priority := "none"
		if task.Priority != nil {
			priority = task.Priority.Priority
		}
		return fmt.Sprintf(":exclamation: **%s** changed priority of **[ %s ](%s)** to **%s**",
			actor, task.Name, task.URL, priority)
	default:
		return fmt.Sprintf(":information_source: ClickUp task **[ %s ](%s)** was updated (%s)",
			task.Name, task.URL, payload.Event)
	}
}

func (p *Plugin) notifyAssignees(payload ClickUpWebhookPayload, task *ClickUpTask) {
	if payload.Event != "taskAssigneeUpdated" && payload.Event != "taskCreated" {
		return
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return
	}

	for _, assignee := range task.Assignees {
		mmUser, err := p.findMattermostUserByEmail(assignee.Email)
		if err != nil {
			continue
		}

		channel, appErr := p.API.GetDirectChannel(p.botUserID, mmUser.Id)
		if appErr != nil {
			continue
		}

		msg := fmt.Sprintf(":bell: You were assigned to ClickUp task **[ %s ](%s)**", task.Name, task.URL)
		if due := formatDueDate(task.DueDate); due != "" {
			msg += " — due **" + due + "**"
		}

		post := &model.Post{
			UserId:    p.botUserID,
			ChannelId: channel.Id,
			Message:   msg,
		}
		p.API.CreatePost(post)
		_ = client
	}
}

// Post action integration for creating tasks from messages.
func (p *Plugin) handlePostActionCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Context == nil || req.Context["action"] != "create_task" {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	post, appErr := p.API.GetPost(req.PostId)
	if appErr != nil {
		http.Error(w, "Post not found", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(post.Message)
	if len(name) > 80 {
		name = name[:80] + "..."
	}

	description := post.Message
	description += fmt.Sprintf("\n\nSource: %s/_redirect/pl/%s", p.getSiteURL(), post.Id)

	if err := p.openCreateTaskFromPost(req.TriggerId, post.ChannelId, post.Id, name, description); err != nil {
		p.API.LogWarn("failed to open create task dialog", "error", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.PostActionIntegrationResponse{})
}

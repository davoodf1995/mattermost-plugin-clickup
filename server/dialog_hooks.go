package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) handleCreateTaskDialog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.SubmitDialogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.State != "" {
		var state struct {
			ChannelID string `json:"channel_id"`
			PostID    string `json:"post_id,omitempty"`
		}
		_ = json.Unmarshal([]byte(req.State), &state)
		if state.ChannelID != "" {
			req.ChannelId = state.ChannelID
		}
	}

	client, err := p.getClickUpClient()
	if err != nil {
		p.respondDialogError(w, err.Error())
		return
	}

	listID, err := p.resolveListID(req.ChannelId)
	if err != nil {
		p.respondDialogError(w, err.Error())
		return
	}

	name := dialogValue(req.Submission, "name")
	if name == "" {
		p.respondDialogError(w, "Task name is required")
		return
	}

	createReq := CreateTaskRequest{
		Name:        name,
		Description: dialogValue(req.Submission, "description"),
	}

	if due := dialogValue(req.Submission, "due_date"); due != "" {
		dueMs, err := dueDateToMillis(due)
		if err != nil {
			p.respondDialogError(w, "Invalid due date. Use YYYY-MM-DD.")
			return
		}
		createReq.DueDate = dueMs
	}

	if priority := dialogValue(req.Submission, "priority"); priority != "" {
		createReq.Priority = priorityToInt(priority)
	}

	if assignee := dialogValue(req.Submission, "assignee"); assignee != "" {
		assignee = strings.TrimPrefix(assignee, "@")
		user, appErr := p.API.GetUserByUsername(assignee)
		if appErr != nil {
			p.respondDialogError(w, "Unknown user "+assignee)
			return
		}
		clickupID, err := p.findClickUpUserByEmail(client, user.Email)
		if err != nil {
			p.respondDialogError(w, err.Error())
			return
		}
		createReq.Assignees = []int{clickupID}
	}

	task, err := client.CreateTask(listID, createReq)
	if err != nil {
		p.respondDialogError(w, err.Error())
		return
	}

	p.postToChannel(req.ChannelId, fmt.Sprintf(":white_check_mark: Created ClickUp task **[ %s ](%s)** — %s",
		task.Name, task.URL, assigneeNames(*task)))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.SubmitDialogResponse{})
}

func (p *Plugin) handleAssignTaskDialog(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (p *Plugin) respondDialogError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(model.SubmitDialogResponse{
		Errors: map[string]string{"name": message},
	})
}

func (p *Plugin) openCreateTaskFromPost(triggerID, channelID, postID, defaultName, defaultDescription string) error {
	state, _ := json.Marshal(map[string]string{
		"channel_id": channelID,
		"post_id":    postID,
	})

	dialog := model.Dialog{
		Title:       "Create ClickUp Task",
		CallbackId:  "create-task",
		SubmitLabel: "Create",
		State:       string(state),
		Elements: []model.DialogElement{
			{
				DisplayName: "Task name",
				Name:        "name",
				Type:        "text",
				Default:     defaultName,
				Optional:    false,
			},
			{
				DisplayName: "Description",
				Name:        "description",
				Type:        "textarea",
				Default:     defaultDescription,
				Optional:    true,
			},
			{
				DisplayName: "Assignee (Mattermost username)",
				Name:        "assignee",
				Type:        "text",
				Optional:    true,
			},
			{
				DisplayName: "Due date",
				Name:        "due_date",
				Type:        "text",
				Placeholder: "YYYY-MM-DD",
				Optional:    true,
			},
			{
				DisplayName: "Priority",
				Name:        "priority",
				Type:        "select",
				Optional:    true,
				Options: []*model.PostActionOptions{
					{Text: "Urgent", Value: "urgent"},
					{Text: "High", Value: "high"},
					{Text: "Normal", Value: "normal"},
					{Text: "Low", Value: "low"},
				},
			},
		},
	}

	return p.API.OpenInteractiveDialog(model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       p.getPluginURL() + "/dialog/create-task",
		Dialog:    dialog,
	})
}

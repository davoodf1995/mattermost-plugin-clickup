package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) registerCommands() error {
	return p.API.RegisterCommand(&model.Command{
		Trigger:          commandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Manage ClickUp tasks from Mattermost",
		AutoCompleteHint: "[help|link|unlink|tasks|task|assign|done|comment|my|create]",
		DisplayName:      "ClickUp integration",
	})
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	fields := strings.Fields(args.Command)
	if len(fields) == 0 {
		return p.helpResponse(), nil
	}

	sub := strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	if sub != commandTrigger {
		return p.helpResponse(), nil
	}

	if len(fields) < 2 {
		return p.helpResponse(), nil
	}

	action := strings.ToLower(fields[1])
	switch action {
	case "help", "?":
		return p.helpResponse(), nil
	case "link":
		return p.handleLinkCommand(args, fields[2:])
	case "unlink":
		return p.handleUnlinkCommand(args)
	case "tasks", "list":
		return p.handleTasksCommand(args)
	case "task":
		return p.handleCreateCommand(c, args, fields[2:])
	case "create":
		return p.openCreateTaskDialog(args), nil
	case "assign":
		return p.handleAssignCommand(args, fields[2:])
	case "done", "complete":
		return p.handleDoneCommand(args, fields[2:])
	case "comment":
		return p.handleCommentCommand(args, fields[2:])
	case "my":
		return p.handleMyTasksCommand(args)
	default:
		return p.ephemeral(args.UserId, args.ChannelId, fmt.Sprintf("Unknown subcommand `%s`. Type `/clickup help`.", action)), nil
	}
}

func (p *Plugin) helpResponse() *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text: "#### ClickUp commands\n" +
			"| Command | Description |\n" +
			"|---|---|\n" +
			"| `/clickup link <list_id> [name]` | Link this channel to a ClickUp list |\n" +
			"| `/clickup unlink` | Remove channel link |\n" +
			"| `/clickup tasks` | Show open tasks for the linked list |\n" +
			"| `/clickup task <name>` | Create a task (add `--assign @user --due 2026-06-20 --priority high`) |\n" +
			"| `/clickup create` | Open task creation dialog |\n" +
			"| `/clickup assign <task_id> @user` | Assign a task |\n" +
			"| `/clickup done <task_id>` | Mark task complete |\n" +
			"| `/clickup comment <task_id> <text>` | Add a comment |\n" +
			"| `/clickup my` | Show tasks assigned to you |\n\n" +
			"**Tips:** Use the channel header ClickUp icon for a task panel. " +
			"Use **Create ClickUp Task** on any message. " +
			"Configure API token and Team ID in System Console → Plugins.",
	}
}

func (p *Plugin) handleLinkCommand(args *model.CommandArgs, rest []string) (*model.CommandResponse, *model.AppError) {
	if len(rest) < 1 {
		return p.ephemeral(args.UserId, args.ChannelId, "Usage: `/clickup link <list_id> [list_name]`"), nil
	}

	listID := rest[0]
	listName := ""
	if len(rest) > 1 {
		listName = strings.Join(rest[1:], " ")
	}

	if err := p.setChannelLink(args.ChannelId, listID, listName); err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to save link: "+err.Error()), nil
	}

	label := listID
	if listName != "" {
		label = listName + " (" + listID + ")"
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         fmt.Sprintf("Linked this channel to ClickUp list **%s**.", label),
	}, nil
}

func (p *Plugin) handleUnlinkCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if err := p.removeChannelLink(args.ChannelId); err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to unlink: "+err.Error()), nil
	}
	return p.ephemeral(args.UserId, args.ChannelId, "Channel unlinked from ClickUp."), nil
}

func (p *Plugin) handleTasksCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	client, err := p.getClickUpClient()
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	listID, err := p.resolveListID(args.ChannelId)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	tasks, err := client.GetListTasks(listID, false)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to fetch tasks: "+err.Error()), nil
	}

	if len(tasks) == 0 {
		return p.ephemeral(args.UserId, args.ChannelId, "No open tasks in this list."), nil
	}

	var b strings.Builder
	b.WriteString("#### Open tasks\n")
	limit := 15
	for i, task := range tasks {
		if i >= limit {
			b.WriteString(fmt.Sprintf("\n_...and %d more_", len(tasks)-limit))
			break
		}
		b.WriteString(fmt.Sprintf("- [%s](%s) — **%s** — %s\n", task.Name, task.URL, taskStatusLabel(task), assigneeNames(task)))
	}

	return p.ephemeral(args.UserId, args.ChannelId, b.String()), nil
}

func (p *Plugin) handleCreateCommand(c *plugin.Context, args *model.CommandArgs, rest []string) (*model.CommandResponse, *model.AppError) {
	if len(rest) == 0 || strings.ToLower(rest[0]) == "dialog" {
		return p.openCreateTaskDialog(args), nil
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	listID, err := p.resolveListID(args.ChannelId)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	req, name, err := p.parseCreateTaskArgs(rest, args)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}
	if name == "" {
		return p.ephemeral(args.UserId, args.ChannelId, "Task name is required."), nil
	}
	req.Name = name

	task, err := client.CreateTask(listID, req)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to create task: "+err.Error()), nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("Created ClickUp task **[ %s ](%s)** — %s", task.Name, task.URL, assigneeNames(*task)),
	}, nil
}

func (p *Plugin) parseCreateTaskArgs(rest []string, args *model.CommandArgs) (CreateTaskRequest, string, error) {
	req := CreateTaskRequest{}
	var nameParts []string

	for i := 0; i < len(rest); i++ {
		switch strings.ToLower(rest[i]) {
		case "--assign", "-a":
			if i+1 >= len(rest) {
				return req, "", fmt.Errorf("missing value for --assign")
			}
			i++
			username := strings.TrimPrefix(rest[i], "@")
			user, appErr := p.API.GetUserByUsername(username)
			if appErr != nil {
				return req, "", fmt.Errorf("unknown user @%s", username)
			}
			client, err := p.getClickUpClient()
			if err != nil {
				return req, "", err
			}
			clickupID, err := p.findClickUpUserByEmail(client, user.Email)
			if err != nil {
				return req, "", err
			}
			req.Assignees = append(req.Assignees, clickupID)
		case "--due", "-d":
			if i+1 >= len(rest) {
				return req, "", fmt.Errorf("missing value for --due")
			}
			i++
			due, err := dueDateToMillis(rest[i])
			if err != nil {
				return req, "", err
			}
			req.DueDate = due
		case "--priority", "-p":
			if i+1 >= len(rest) {
				return req, "", fmt.Errorf("missing value for --priority")
			}
			i++
			req.Priority = priorityToInt(rest[i])
		default:
			nameParts = append(nameParts, rest[i])
		}
	}

	return req, strings.Join(nameParts, " "), nil
}

func (p *Plugin) handleAssignCommand(args *model.CommandArgs, rest []string) (*model.CommandResponse, *model.AppError) {
	if len(rest) < 2 {
		return p.ephemeral(args.UserId, args.ChannelId, "Usage: `/clickup assign <task_id> @username`"), nil
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	taskID := parseTaskID(rest[0])
	username := strings.TrimPrefix(rest[1], "@")
	user, appErr := p.API.GetUserByUsername(username)
	if appErr != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Unknown user @"+username), nil
	}

	clickupID, err := p.findClickUpUserByEmail(client, user.Email)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	task, err := client.UpdateTask(taskID, map[string]interface{}{
		"assignees": map[string]interface{}{
			"add": []int{clickupID},
		},
	})
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to assign: "+err.Error()), nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("Assigned **[ %s ](%s)** to @%s", task.Name, task.URL, username),
	}, nil
}

func (p *Plugin) handleDoneCommand(args *model.CommandArgs, rest []string) (*model.CommandResponse, *model.AppError) {
	if len(rest) < 1 {
		return p.ephemeral(args.UserId, args.ChannelId, "Usage: `/clickup done <task_id>`"), nil
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	taskID := parseTaskID(rest[0])
	task, err := client.UpdateTask(taskID, map[string]interface{}{
		"status": "complete",
	})
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to update task: "+err.Error()), nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("Marked **[ %s ](%s)** as complete.", task.Name, task.URL),
	}, nil
}

func (p *Plugin) handleCommentCommand(args *model.CommandArgs, rest []string) (*model.CommandResponse, *model.AppError) {
	if len(rest) < 2 {
		return p.ephemeral(args.UserId, args.ChannelId, "Usage: `/clickup comment <task_id> <text>`"), nil
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	taskID := parseTaskID(rest[0])
	comment := strings.Join(rest[1:], " ")
	if err := client.AddComment(taskID, comment); err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to add comment: "+err.Error()), nil
	}

	return p.ephemeral(args.UserId, args.ChannelId, "Comment added to task "+taskID+"."), nil
}

func (p *Plugin) handleMyTasksCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	client, err := p.getClickUpClient()
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	user, appErr := p.API.GetUser(args.UserId)
	if appErr != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to load user"), nil
	}

	clickupID, err := p.findClickUpUserByEmail(client, user.Email)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Could not match your Mattermost email to a ClickUp user. "+err.Error()), nil
	}

	listID, err := p.resolveListID(args.ChannelId)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	tasks, err := client.GetListTasks(listID, false)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to fetch tasks: "+err.Error()), nil
	}

	var mine []ClickUpTask
	for _, task := range tasks {
		for _, assignee := range task.Assignees {
			if assignee.ID == clickupID {
				mine = append(mine, task)
				break
			}
		}
	}

	if len(mine) == 0 {
		return p.ephemeral(args.UserId, args.ChannelId, "You have no open tasks in this list."), nil
	}

	var b strings.Builder
	b.WriteString("#### Your tasks\n")
	for _, task := range mine {
		due := formatDueDate(task.DueDate)
		line := fmt.Sprintf("- [%s](%s) — **%s**", task.Name, task.URL, taskStatusLabel(task))
		if due != "" {
			line += " — due " + due
		}
		b.WriteString(line + "\n")
	}

	return p.ephemeral(args.UserId, args.ChannelId, b.String()), nil
}

func (p *Plugin) openCreateTaskDialog(args *model.CommandArgs) *model.CommandResponse {
	if args.TriggerId == "" {
		return p.ephemeral(args.UserId, args.ChannelId, "Usage: `/clickup task <name>` or `/clickup create`")
	}

	dialog := model.Dialog{
		Title:       "Create ClickUp Task",
		CallbackId:  "create-task",
		SubmitLabel: "Create",
		Elements: []model.DialogElement{
			{
				DisplayName: "Task name",
				Name:        "name",
				Type:        "text",
				Placeholder: "What needs to be done?",
				Optional:    false,
			},
			{
				DisplayName: "Description",
				Name:        "description",
				Type:        "textarea",
				Placeholder: "Details, links, acceptance criteria...",
				Optional:    true,
			},
			{
				DisplayName: "Assignee (Mattermost username)",
				Name:        "assignee",
				Type:        "text",
				Placeholder: "username (without @)",
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

	req := model.OpenDialogRequest{
		TriggerId: args.TriggerId,
		URL:       p.getPluginURL() + "/dialog/create-task",
		Dialog:    dialog,
	}

	if appErr := p.API.OpenInteractiveDialog(req); appErr != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to open dialog: "+appErr.Error())
	}

	return p.ephemeral(args.UserId, args.ChannelId, "")
}

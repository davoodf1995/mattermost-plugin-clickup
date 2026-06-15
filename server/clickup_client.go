package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ClickUpClient struct {
	token  string
	client *http.Client
}

func NewClickUpClient(token string) *ClickUpClient {
	return &ClickUpClient{
		token:  token,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type ClickUpTask struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      ClickUpStatus     `json:"status"`
	URL         string            `json:"url"`
	DueDate     *json.Number      `json:"due_date"`
	Assignees   []ClickUpAssignee `json:"assignees"`
	Priority    *ClickUpPriority  `json:"priority"`
	List        ClickUpListRef    `json:"list"`
}

type ClickUpStatus struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}

type ClickUpAssignee struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	ProfilePicture string `json:"profilePicture"`
}

type ClickUpPriority struct {
	Priority string `json:"priority"`
}

type ClickUpListRef struct {
	ID string `json:"id"`
}

type ClickUpMember struct {
	User ClickUpAssignee `json:"user"`
}

type ClickUpWebhookPayload struct {
	Event     string          `json:"event"`
	WebhookID string          `json:"webhook_id"`
	TaskID    string          `json:"task_id"`
	History   []ClickUpChange `json:"history_items"`
}

type ClickUpChange struct {
	Field string      `json:"field"`
	User  ClickUpUser `json:"user"`
	After interface{} `json:"after"`
}

type ClickUpUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (c *ClickUpClient) request(method, path string, body interface{}, result interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, clickUpAPIBase+path, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("clickup API %s %s: %s", method, path, string(respBody))
	}

	if result == nil {
		return nil
	}

	return json.Unmarshal(respBody, result)
}

func (c *ClickUpClient) GetTask(taskID string) (*ClickUpTask, error) {
	var task ClickUpTask
	if err := c.request(http.MethodGet, "/task/"+taskID, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *ClickUpClient) GetListTasks(listID string, includeClosed bool) ([]ClickUpTask, error) {
	path := fmt.Sprintf("/list/%s/task?include_closed=%t&subtasks=true", listID, includeClosed)
	var result struct {
		Tasks []ClickUpTask `json:"tasks"`
	}
	if err := c.request(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

func (c *ClickUpClient) GetViewTasks(viewID string, includeClosed bool) ([]ClickUpTask, error) {
	path := fmt.Sprintf("/view/%s/task?include_closed=%t", viewID, includeClosed)
	var result struct {
		Tasks []ClickUpTask `json:"tasks"`
	}
	if err := c.request(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

func (c *ClickUpClient) GetTasks(listID, viewID string, includeClosed bool) ([]ClickUpTask, error) {
	if viewID != "" {
		return c.GetViewTasks(viewID, includeClosed)
	}
	return c.GetListTasks(listID, includeClosed)
}

type CreateTaskRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Assignees   []int    `json:"assignees,omitempty"`
	DueDate     *int64   `json:"due_date,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (c *ClickUpClient) CreateTask(listID string, req CreateTaskRequest) (*ClickUpTask, error) {
	var task ClickUpTask
	if err := c.request(http.MethodPost, "/list/"+listID+"/task", req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *ClickUpClient) UpdateTask(taskID string, updates map[string]interface{}) (*ClickUpTask, error) {
	var task ClickUpTask
	if err := c.request(http.MethodPut, "/task/"+taskID, updates, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *ClickUpClient) AddComment(taskID, text string) error {
	body := map[string]string{"comment_text": text}
	return c.request(http.MethodPost, "/task/"+taskID+"/comment", body, nil)
}

func (c *ClickUpClient) GetTeamMembers(teamID string) ([]ClickUpMember, error) {
	var membersResult struct {
		Members []ClickUpMember `json:"members"`
	}
	if err := c.request(http.MethodGet, "/team/"+teamID+"/member", nil, &membersResult); err == nil && len(membersResult.Members) > 0 {
		return membersResult.Members, nil
	}

	// Fallback for older API shapes: members nested under team.
	var teamResult struct {
		Team struct {
			Members []ClickUpMember `json:"members"`
		} `json:"team"`
	}
	if err := c.request(http.MethodGet, "/team/"+teamID, nil, &teamResult); err != nil {
		return nil, err
	}
	if len(teamResult.Team.Members) > 0 {
		return teamResult.Team.Members, nil
	}

	if len(membersResult.Members) > 0 {
		return membersResult.Members, nil
	}

	return nil, fmt.Errorf("no members returned for ClickUp team %s — verify Team ID in plugin settings", teamID)
}

func (c *ClickUpClient) CreateWebhook(teamID, endpoint, secret string) (string, error) {
	body := map[string]interface{}{
		"endpoint": endpoint,
		"events": []string{
			"taskCreated",
			"taskUpdated",
			"taskStatusUpdated",
			"taskAssigneeUpdated",
			"taskDueDateUpdated",
			"taskCommentPosted",
			"taskPriorityUpdated",
		},
		"secret": secret,
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := c.request(http.MethodPost, "/team/"+teamID+"/webhook", body, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *ClickUpClient) DeleteWebhook(webhookID string) error {
	return c.request(http.MethodDelete, "/webhook/"+webhookID, nil, nil)
}

func parseTaskID(input string) string {
	input = strings.TrimSpace(input)
	if strings.Contains(input, "clickup.com/") {
		parts := strings.Split(strings.TrimRight(input, "/"), "/")
		return parts[len(parts)-1]
	}
	return input
}

func priorityToInt(priority string) *int {
	switch strings.ToLower(priority) {
	case "urgent":
		v := 1
		return &v
	case "high":
		v := 2
		return &v
	case "normal":
		v := 3
		return &v
	case "low":
		v := 4
		return &v
	default:
		return nil
	}
}

func dueDateToMillis(date string) (*int64, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	ms := t.UnixMilli()
	return &ms, nil
}

func formatDueDate(due *json.Number) string {
	if due == nil {
		return ""
	}
	ms, err := due.Int64()
	if err != nil {
		return ""
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04")
}

func taskStatusLabel(task ClickUpTask) string {
	if task.Status.Status != "" {
		return task.Status.Status
	}
	return "open"
}

func assigneeNames(task ClickUpTask) string {
	if len(task.Assignees) == 0 {
		return "unassigned"
	}
	names := make([]string, 0, len(task.Assignees))
	for _, a := range task.Assignees {
		if a.Username != "" {
			names = append(names, a.Username)
		} else if a.Email != "" {
			names = append(names, a.Email)
		} else {
			names = append(names, strconv.Itoa(a.ID))
		}
	}
	return strings.Join(names, ", ")
}

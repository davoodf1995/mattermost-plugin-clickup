package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	clickUpListURLPattern = regexp.MustCompile(`/v/li/(\d+)`)
	clickUpViewURLPattern = regexp.MustCompile(`/v/l/([^/?#]+)`)
)

const (
	clickUpParentTeam   = 7
	clickUpParentSpace  = 4
	clickUpParentFolder = 5
	clickUpParentList   = 6
)

type ClickUpView struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Parent ClickUpViewParent `json:"parent"`
}

type ClickUpViewParent struct {
	ID   interface{}   `json:"id"`
	Type clickUpParentType `json:"type"`
}

type clickUpParentType int

func (t *clickUpParentType) UnmarshalJSON(data []byte) error {
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*t = clickUpParentType(i)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		parsed, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		*t = clickUpParentType(parsed)
		return nil
	}
	return fmt.Errorf("invalid parent type: %s", string(data))
}

type ClickUpList struct {
	ID   clickUpID `json:"id"`
	Name string    `json:"name"`
}

type clickUpID string

func (id clickUpID) String() string {
	return string(id)
}

func (id *clickUpID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*id = clickUpID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*id = clickUpID(n.String())
		return nil
	}
	var i int64
	if err := json.Unmarshal(data, &i); err == nil {
		*id = clickUpID(fmt.Sprintf("%d", i))
		return nil
	}
	return fmt.Errorf("invalid ClickUp id: %s", string(data))
}

func parseClickUpReference(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if strings.Contains(input, "clickup.com") {
		if match := clickUpViewURLPattern.FindStringSubmatch(input); len(match) == 2 {
			return match[1]
		}
		if match := clickUpListURLPattern.FindStringSubmatch(input); len(match) == 2 {
			return match[1]
		}
	}

	return input
}

func looksLikeViewID(id string) bool {
	return strings.Contains(id, "-")
}

func parentIDString(parent ClickUpViewParent) string {
	switch v := parent.ID.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", parent.ID)
	}
}

func parseLinkRestFlags(rest []string) (ref, listIDOverride, listName string) {
	if len(rest) == 0 {
		return "", "", ""
	}
	ref = rest[0]
	var extras []string
	for i := 1; i < len(rest); i++ {
		token := rest[i]
		lower := strings.ToLower(token)
		if (lower == "--list_id" || lower == "--list-id") && i+1 < len(rest) {
			listIDOverride = rest[i+1]
			i++
			continue
		}
		extras = append(extras, token)
	}
	if listIDOverride == "" && len(extras) == 1 && isNumericListID(extras[0]) {
		listIDOverride = extras[0]
	} else if len(extras) > 0 {
		listName = strings.Join(extras, " ")
	}
	return ref, listIDOverride, listName
}

func linkCommandArgs(command string) []string {
	ref, tail := parseLinkCommandInput(command)
	if ref == "" {
		return nil
	}
	return append([]string{ref}, tail...)
}

func parseLinkCommandInput(command string) (reference string, tail []string) {
	command = strings.TrimSpace(command)
	command = strings.TrimPrefix(command, "/")
	if strings.HasPrefix(strings.ToLower(command), commandTrigger) {
		command = strings.TrimSpace(command[len(commandTrigger):])
	}
	if !strings.HasPrefix(strings.ToLower(command), "link") {
		return "", nil
	}
	command = strings.TrimSpace(command[4:])
	command = strings.Trim(command, "<>")
	if command == "" {
		return "", nil
	}

	if idx := strings.Index(command, "clickup.com"); idx >= 0 {
		start := idx
		for start > 0 && command[start-1] != ' ' && command[start-1] != '\t' {
			start--
		}
		end := idx + len("clickup.com")
		for end < len(command) && command[end] != ' ' && command[end] != '\t' {
			end++
		}
		reference = command[start:end]
		rest := strings.TrimSpace(command[end:])
		if rest != "" {
			tail = strings.Fields(rest)
		}
		return reference, tail
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

func (c *ClickUpClient) GetView(viewID string) (*ClickUpView, error) {
	var raw json.RawMessage
	if err := c.request(http.MethodGet, "/view/"+viewID, nil, &raw); err != nil {
		return nil, err
	}

	var wrapped struct {
		View ClickUpView `json:"view"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.View.ID != "" {
		return &wrapped.View, nil
	}

	var direct ClickUpView
	if err := json.Unmarshal(raw, &direct); err != nil {
		return nil, err
	}
	if direct.ID == "" {
		return nil, fmt.Errorf("view %s not found", viewID)
	}
	return &direct, nil
}

func (c *ClickUpClient) GetList(listID string) (*ClickUpList, error) {
	var list ClickUpList
	if err := c.request(http.MethodGet, "/list/"+listID, nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

func (c *ClickUpClient) GetFolderLists(folderID string) ([]ClickUpList, error) {
	var result struct {
		Lists []ClickUpList `json:"lists"`
	}
	path := fmt.Sprintf("/folder/%s/list?archived=false", folderID)
	if err := c.request(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.Lists, nil
}

func (c *ClickUpClient) GetSpaceLists(spaceID string) ([]ClickUpList, error) {
	var result struct {
		Lists []ClickUpList `json:"lists"`
	}
	path := fmt.Sprintf("/space/%s/list?archived=false", spaceID)
	if err := c.request(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.Lists, nil
}

func firstListID(lists []ClickUpList) (string, error) {
	if len(lists) == 0 {
		return "", fmt.Errorf("no lists found")
	}
	return lists[0].ID.String(), nil
}

func (c *ClickUpClient) resolveListIDFromViewParent(parent ClickUpViewParent) (string, error) {
	id := parentIDString(parent)
	if id == "" {
		return "", fmt.Errorf("view has no parent")
	}

	switch int(parent.Type) {
	case clickUpParentList, 0:
		return id, nil
	case clickUpParentFolder:
		lists, err := c.GetFolderLists(id)
		if err != nil {
			return "", fmt.Errorf("could not load lists for folder %s: %w", id, err)
		}
		listID, err := firstListID(lists)
		if err != nil {
			return "", fmt.Errorf("folder %s has no lists to create tasks in", id)
		}
		return listID, nil
	case clickUpParentSpace:
		lists, err := c.GetSpaceLists(id)
		if err != nil {
			return "", fmt.Errorf("could not load lists for space %s: %w", id, err)
		}
		listID, err := firstListID(lists)
		if err != nil {
			return "", fmt.Errorf("space %s has no lists to create tasks in", id)
		}
		return listID, nil
	case clickUpParentTeam:
		return "", nil
	default:
		return "", fmt.Errorf("unsupported ClickUp view parent type %d", parent.Type)
	}
}

// ResolveReference turns a ClickUp list URL, view URL, list ID, or view ID into list/view IDs for the API.
func (c *ClickUpClient) ResolveReference(input string) (listID, viewID, displayName string, err error) {
	ref := parseClickUpReference(input)
	if ref == "" {
		return "", "", "", fmt.Errorf("empty ClickUp list or view reference")
	}

	if looksLikeViewID(ref) {
		view, err := c.GetView(ref)
		if err != nil {
			return "", "", "", fmt.Errorf("could not resolve ClickUp view %s: %w", ref, err)
		}

		listID, err := c.resolveListIDFromViewParent(view.Parent)
		if err != nil {
			return "", "", "", err
		}

		return listID, ref, view.Name, nil
	}

	list, err := c.GetList(ref)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid ClickUp list ID %s: %w", ref, err)
	}

	return ref, "", list.Name, nil
}

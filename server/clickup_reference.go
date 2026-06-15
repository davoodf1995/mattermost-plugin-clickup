package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var (
	clickUpListURLPattern = regexp.MustCompile(`/v/li/(\d+)`)
	clickUpViewURLPattern = regexp.MustCompile(`/v/l/([^/?#]+)`)
)

type ClickUpView struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Parent ClickUpViewParent `json:"parent"`
}

type ClickUpViewParent struct {
	ID   interface{} `json:"id"` // API returns string or number
	Type int         `json:"type"`
}

type ClickUpList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

func (c *ClickUpClient) GetView(viewID string) (*ClickUpView, error) {
	var result struct {
		View ClickUpView `json:"view"`
	}
	if err := c.request(http.MethodGet, "/view/"+viewID, nil, &result); err != nil {
		return nil, err
	}
	if result.View.ID == "" {
		return nil, fmt.Errorf("view %s not found", viewID)
	}
	return &result.View, nil
}

func (c *ClickUpClient) GetList(listID string) (*ClickUpList, error) {
	var list ClickUpList
	if err := c.request(http.MethodGet, "/list/"+listID, nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
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

		listID, err := listIDFromViewParent(view.Parent)
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

func listIDFromViewParent(parent ClickUpViewParent) (string, error) {
	id := parentIDString(parent)
	if id == "" {
		return "", fmt.Errorf("view has no parent list")
	}

	// parent.type 6 = List per ClickUp API docs
	if parent.Type != 0 && parent.Type != 6 {
		return "", fmt.Errorf("this view is not attached to a list (parent type %d). Open the List in ClickUp and copy its URL", parent.Type)
	}

	return id, nil
}

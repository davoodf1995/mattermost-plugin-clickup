package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
)

func TestHandleLinkCommandFolderView(t *testing.T) {
	viewBody := map[string]any{
		"view": map[string]any{
			"id":   "5-19524559-1",
			"name": "My Folder View",
			"type": "list",
			"parent": map[string]any{
				"id":   19524559,
				"type": 5,
			},
		},
	}
	listsBody := map[string]any{
		"lists": []map[string]any{
			{"id": 901234567890, "name": "Tasks"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/view/5-19524559-1":
			_ = json.NewEncoder(w).Encode(viewBody)
		case "/api/v2/folder/19524559/list":
			_ = json.NewEncoder(w).Encode(listsBody)
		default:
			t.Logf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	api := &plugintest.API{}
	api.On("KVSet", mock.Anything, mock.Anything).Return((*model.AppError)(nil))

	p := &Plugin{}
	p.API = api
	p.setConfiguration(&configuration{ClickUpAPIToken: "pk_test"})
	p.clickup = &ClickUpClient{
		token:   "pk_test",
		client:  srv.Client(),
		baseURL: srv.URL + "/api/v2",
	}

	args := &model.CommandArgs{
		ChannelId: "channel1",
		UserId:    "user1",
		Command:   "/clickup link https://app.clickup.com/2678792/v/l/5-19524559-1",
	}

	resp, appErr := p.handleLinkCommand(args, linkCommandArgs(args.Command))
	if appErr != nil {
		t.Fatalf("appErr: %v", appErr)
	}
	if resp == nil || resp.Text == "" {
		t.Fatalf("empty response: %+v", resp)
	}
	t.Log(resp.Text)
}

func TestResolveReferenceFlatViewJSON(t *testing.T) {
	flatBody := `{
		"id": "5-19524559-1",
		"name": "Flat View",
		"type": "list",
		"parent": {"id": 19524559, "type": 5}
	}`
	listsBody := `{"lists":[{"id":"901234567890","name":"Tasks"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/view/5-19524559-1":
			w.Write([]byte(flatBody))
		case "/api/v2/folder/19524559/list":
			w.Write([]byte(listsBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &ClickUpClient{
		token:   "pk_test",
		client:  srv.Client(),
		baseURL: srv.URL + "/api/v2",
	}

	listID, viewID, name, err := client.ResolveReference("5-19524559-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if viewID != "5-19524559-1" {
		t.Fatalf("viewID=%q", viewID)
	}
	if listID != "901234567890" {
		t.Fatalf("listID=%q", listID)
	}
	if name != "Flat View" {
		t.Fatalf("name=%q", name)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

type channelLink struct {
	ListID   string `json:"list_id"`
	ListName string `json:"list_name,omitempty"`
}

func (p *Plugin) getChannelLink(channelID string) (*channelLink, error) {
	data, appErr := p.API.KVGet(kvChannelListPrefix + channelID)
	if appErr != nil || len(data) == 0 {
		return nil, errNoLinkedList
	}

	var link channelLink
	if err := json.Unmarshal(data, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

func (p *Plugin) setChannelLink(channelID, listID, listName string) error {
	link := channelLink{ListID: listID, ListName: listName}
	data, err := json.Marshal(link)
	if err != nil {
		return err
	}
	return p.API.KVSet(kvChannelListPrefix+channelID, data)
}

func (p *Plugin) removeChannelLink(channelID string) error {
	return p.API.KVDelete(kvChannelListPrefix + channelID)
}

func (p *Plugin) resolveListID(channelID string) (string, error) {
	link, err := p.getChannelLink(channelID)
	if err == nil && link.ListID != "" {
		return link.ListID, nil
	}

	config := p.getConfiguration()
	if config.DefaultListID != "" {
		return config.DefaultListID, nil
	}

	return "", errNoLinkedList
}

func (p *Plugin) findClickUpUserByEmail(client *ClickUpClient, email string) (int, error) {
	config := p.getConfiguration()
	if config.ClickUpTeamID == "" {
		return 0, fmt.Errorf("ClickUp team ID is not configured")
	}

	members, err := client.GetTeamMembers(config.ClickUpTeamID)
	if err != nil {
		return 0, err
	}

	email = strings.ToLower(strings.TrimSpace(email))
	for _, member := range members {
		if strings.ToLower(member.User.Email) == email {
			return member.User.ID, nil
		}
	}

	return 0, fmt.Errorf("no ClickUp user found for %s", email)
}

func (p *Plugin) findMattermostUserByEmail(email string) (*model.User, error) {
	user, appErr := p.API.GetUserByEmail(email)
	if appErr == nil && user != nil {
		return user, nil
	}

	mmUsers, appErr := p.API.SearchUsers(&model.UserSearch{
		Term: email,
	})
	if appErr != nil {
		return nil, appErr
	}

	email = strings.ToLower(strings.TrimSpace(email))
	for _, u := range mmUsers {
		if strings.ToLower(u.Email) == email {
			return u, nil
		}
	}

	return nil, fmt.Errorf("no Mattermost user found for %s", email)
}

func (p *Plugin) postToChannel(channelID, message string) {
	if p.botUserID == "" {
		return
	}

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelID,
		Message:   message,
	}
	p.API.CreatePost(post)
}

func (p *Plugin) ephemeral(userID, channelID, message string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         message,
	}
}

func (p *Plugin) findChannelsForList(listID string) []string {
	keys := p.listAllKVKeys()

	var channels []string
	for _, key := range keys {
		if !strings.HasPrefix(key, kvChannelListPrefix) {
			continue
		}
		link, err := p.getChannelLink(strings.TrimPrefix(key, kvChannelListPrefix))
		if err == nil && link.ListID == listID {
			channels = append(channels, strings.TrimPrefix(key, kvChannelListPrefix))
		}
	}
	return channels
}

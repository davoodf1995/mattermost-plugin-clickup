package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

type ClickUpSpace struct {
	ID   clickUpID `json:"id"`
	Name string    `json:"name"`
}

type ClickUpFolder struct {
	ID   clickUpID `json:"id"`
	Name string    `json:"name"`
}

func (c *ClickUpClient) GetTeamSpaces(teamID string) ([]ClickUpSpace, error) {
	var result struct {
		Spaces []ClickUpSpace `json:"spaces"`
	}
	if err := c.request(http.MethodGet, "/team/"+teamID+"/space?archived=false", nil, &result); err != nil {
		return nil, err
	}
	return result.Spaces, nil
}

func (c *ClickUpClient) GetSpaceFolders(spaceID string) ([]ClickUpFolder, error) {
	var result struct {
		Folders []ClickUpFolder `json:"folders"`
	}
	if err := c.request(http.MethodGet, "/space/"+spaceID+"/folder?archived=false", nil, &result); err != nil {
		return nil, err
	}
	return result.Folders, nil
}

func (p *Plugin) handleListsCommand(args *model.CommandArgs, rest []string) (*model.CommandResponse, *model.AppError) {
	config := p.getConfiguration()
	if config.ClickUpTeamID == "" {
		return p.ephemeral(args.UserId, args.ChannelId, "Set **ClickUp Team ID** in plugin settings first."), nil
	}

	client, err := p.getClickUpClient()
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, err.Error()), nil
	}

	if len(rest) == 0 {
		return p.formatSpacesList(args, client, config.ClickUpTeamID)
	}

	arg := strings.Join(rest, " ")
	lower := strings.ToLower(arg)

	if strings.HasPrefix(lower, "folder ") {
		return p.formatFolderLists(args, client, strings.TrimSpace(arg[7:]))
	}
	if strings.HasPrefix(lower, "space ") {
		return p.formatSpaceContents(args, client, strings.TrimSpace(arg[6:]))
	}
	if isNumericListID(arg) {
		if resp, handled := p.tryFormatSpaceOrFolder(args, client, arg); handled {
			return resp, nil
		}
	}

	if resp, handled := p.tryFormatSpaceOrFolder(args, client, arg); handled {
		return resp, nil
	}

	return p.formatSpacesSearch(args, client, config.ClickUpTeamID, arg)
}

func (p *Plugin) tryFormatSpaceOrFolder(args *model.CommandArgs, client *ClickUpClient, id string) (*model.CommandResponse, bool) {
	if resp, err := p.formatSpaceContents(args, client, id); err == nil && resp != nil && !strings.Contains(resp.Text, "Failed") {
		return resp, true
	}
	if resp, err := p.formatFolderLists(args, client, id); err == nil && resp != nil && !strings.Contains(resp.Text, "Failed") {
		return resp, true
	}
	return nil, false
}

func (p *Plugin) formatSpacesList(args *model.CommandArgs, client *ClickUpClient, teamID string) (*model.CommandResponse, *model.AppError) {
	spaces, err := client.GetTeamSpaces(teamID)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to load spaces: "+err.Error()), nil
	}

	var b strings.Builder
	b.WriteString("#### ClickUp Spaces\n")
	b.WriteString("Run `/clickup lists space <id>` to see folders and lists.\n\n")
	for _, space := range spaces {
		b.WriteString(fmt.Sprintf("- **%s** — space id `%s`\n", space.Name, space.ID.String()))
	}
	b.WriteString("\nTip: `/clickup lists OMEST` searches by name. `/clickup lists folder <id>` shows lists in a folder.")
	return p.ephemeral(args.UserId, args.ChannelId, b.String()), nil
}

func (p *Plugin) formatSpaceContents(args *model.CommandArgs, client *ClickUpClient, spaceID string) (*model.CommandResponse, *model.AppError) {
	folders, err := client.GetSpaceFolders(spaceID)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to load space "+spaceID+": "+err.Error()), nil
	}

	lists, err := client.GetSpaceLists(spaceID)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to load lists for space "+spaceID+": "+err.Error()), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("#### Space `%s`\n\n", spaceID))
	if len(folders) > 0 {
		b.WriteString("**Folders**\n")
		for _, folder := range folders {
			b.WriteString(fmt.Sprintf("- **%s** — folder id `%s` (run `/clickup lists folder %s`)\n",
				folder.Name, folder.ID.String(), folder.ID.String()))
		}
		b.WriteString("\n")
	}
	if len(lists) > 0 {
		b.WriteString("**Lists (no folder)**\n")
		for _, list := range lists {
			b.WriteString(fmt.Sprintf("- **%s** — list id `%s`\n", list.Name, list.ID.String()))
		}
	}
	if len(folders) == 0 && len(lists) == 0 {
		b.WriteString("No folders or lists found in this space.")
	}
	return p.ephemeral(args.UserId, args.ChannelId, b.String()), nil
}

func (p *Plugin) formatFolderLists(args *model.CommandArgs, client *ClickUpClient, folderID string) (*model.CommandResponse, *model.AppError) {
	lists, err := client.GetFolderLists(folderID)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to load folder "+folderID+": "+err.Error()), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("#### Folder `%s`\n\n", folderID))
	if len(lists) == 0 {
		b.WriteString("No lists in this folder.")
	} else {
		for _, list := range lists {
			b.WriteString(fmt.Sprintf("- **%s** — list id `%s`\n", list.Name, list.ID.String()))
		}
		b.WriteString("\nLink with:\n`/clickup link https://app.clickup.com/.../v/li/LIST_ID` or `/clickup link VIEW_URL LIST_ID`")
	}
	return p.ephemeral(args.UserId, args.ChannelId, b.String()), nil
}

func (p *Plugin) formatSpacesSearch(args *model.CommandArgs, client *ClickUpClient, teamID, query string) (*model.CommandResponse, *model.AppError) {
	spaces, err := client.GetTeamSpaces(teamID)
	if err != nil {
		return p.ephemeral(args.UserId, args.ChannelId, "Failed to load spaces: "+err.Error()), nil
	}

	query = strings.ToLower(strings.TrimSpace(query))
	var b strings.Builder
	b.WriteString(fmt.Sprintf("#### Search: %s\n\n", query))
	found := 0

	for _, space := range spaces {
		if !strings.Contains(strings.ToLower(space.Name), query) {
			continue
		}
		found++
		b.WriteString(fmt.Sprintf("**Space: %s** (`%s`)\n", space.Name, space.ID.String()))
		p.appendSpaceMatches(&b, client, space.ID.String(), query)
		b.WriteString("\n")
	}

	for _, space := range spaces {
		folders, err := client.GetSpaceFolders(space.ID.String())
		if err != nil {
			continue
		}
		for _, folder := range folders {
			if !strings.Contains(strings.ToLower(folder.Name), query) {
				continue
			}
			found++
			b.WriteString(fmt.Sprintf("**%s / %s** — folder `%s`\n", space.Name, folder.Name, folder.ID.String()))
			lists, _ := client.GetFolderLists(folder.ID.String())
			for _, list := range lists {
				b.WriteString(fmt.Sprintf("  - **%s** — list id `%s`\n", list.Name, list.ID.String()))
			}
			b.WriteString("\n")
		}
	}

	if found == 0 {
		b.WriteString("No matches. Try `/clickup lists` to browse all spaces.")
	}
	return p.ephemeral(args.UserId, args.ChannelId, b.String()), nil
}

func (p *Plugin) appendSpaceMatches(b *strings.Builder, client *ClickUpClient, spaceID, query string) {
	folders, _ := client.GetSpaceFolders(spaceID)
	for _, folder := range folders {
		lists, _ := client.GetFolderLists(folder.ID.String())
		for _, list := range lists {
			if strings.Contains(strings.ToLower(list.Name), query) || strings.Contains(strings.ToLower(folder.Name), query) {
				b.WriteString(fmt.Sprintf("- **%s / %s** — list id `%s`\n", folder.Name, list.Name, list.ID.String()))
			}
		}
	}
	lists, _ := client.GetSpaceLists(spaceID)
	for _, list := range lists {
		if strings.Contains(strings.ToLower(list.Name), query) {
			b.WriteString(fmt.Sprintf("- **%s** — list id `%s`\n", list.Name, list.ID.String()))
		}
	}
}

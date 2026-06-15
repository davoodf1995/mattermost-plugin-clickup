package main

import "github.com/mattermost/mattermost/server/public/model"

const commandDescription = "Available commands: link, unlink, tasks, create, task, assign, done, comment, my, help"

func getAutocompleteData() *model.AutocompleteData {
	clickup := model.NewAutocompleteData(commandTrigger, "[command]", commandDescription)

	link := model.NewAutocompleteData("link", "<list_url_or_id> [name]", "Link this channel to a ClickUp list (URL or ID)")
	link.AddNamedTextArgument("list_id", "ClickUp list ID (numeric)", "901234567890", "", true)
	link.AddNamedTextArgument("name", "Optional list label", "Sprint Backlog", "", false)
	clickup.AddCommand(link)

	unlink := model.NewAutocompleteData("unlink", "", "Remove the ClickUp list link from this channel")
	clickup.AddCommand(unlink)

	tasks := model.NewAutocompleteData("tasks", "", "Show open tasks for the linked list")
	clickup.AddCommand(tasks)

	create := model.NewAutocompleteData("create", "", "Open the task creation dialog")
	clickup.AddCommand(create)

	task := model.NewAutocompleteData("task", "<name>", "Create a task quickly from the channel")
	task.AddTextArgument("Task title", "Fix login bug", "")
	clickup.AddCommand(task)

	assign := model.NewAutocompleteData("assign", "<task_id> @user", "Assign a ClickUp task to a teammate")
	assign.AddTextArgument("ClickUp task ID or URL", "abc123", "")
	assign.AddTextArgument("Mattermost username", "john", "")
	clickup.AddCommand(assign)

	done := model.NewAutocompleteData("done", "<task_id>", "Mark a ClickUp task as complete")
	done.AddTextArgument("ClickUp task ID or URL", "abc123", "")
	clickup.AddCommand(done)

	comment := model.NewAutocompleteData("comment", "<task_id> <text>", "Add a comment to a ClickUp task")
	comment.AddTextArgument("ClickUp task ID or URL", "abc123", "")
	comment.AddTextArgument("Comment text", "Updated in Mattermost", "")
	clickup.AddCommand(comment)

	my := model.NewAutocompleteData("my", "", "Show tasks assigned to you")
	clickup.AddCommand(my)

	help := model.NewAutocompleteData("help", "", "Show ClickUp slash command help")
	clickup.AddCommand(help)

	return clickup
}

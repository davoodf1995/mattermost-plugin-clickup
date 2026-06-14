# Mattermost ClickUp Plugin

Mattermost plugin for [ClickUp](https://clickup.com) integration, maintained by [devmatika](https://github.com/devmatika).

Connect Mattermost channels to ClickUp lists, create and manage tasks, get real-time notifications, and receive due-date reminders — without leaving your chat.

**Requires Mattermost Server v11.0.0 or later.**

## Features

- **Channel ↔ List linking** — map each Mattermost channel to a ClickUp list
- **Create tasks** — slash commands, interactive dialog, or from any message
- **Assign tasks** — match Mattermost users to ClickUp members by email
- **Task panel** — view open tasks in the right-hand sidebar (app bar / channel header)
- **Webhooks** — ClickUp events posted to linked channels (created, assigned, status, due date, comments)
- **Due-date reminders** — DM and channel alerts before tasks are due
- **Mobile-friendly** — slash commands and dialogs work on mobile ([Mattermost mobile plugin docs](https://developers.mattermost.com/integrate/plugins/components/mobile/))

## Setup

### 1. ClickUp credentials

1. In ClickUp go to **Settings → Apps** and create a **Personal API Token** (`pk_...`).
2. Find your **Team ID** from the URL: `https://app.clickup.com/{team_id}/...`
3. Find a **List ID** from the list URL or API.

### 2. Install plugin

1. Build: `make dist`
2. Upload `dist/com.mattermost.clickup-1.0.0.tar.gz` via **System Console → Plugins**
3. Enable the plugin and configure:
   - **ClickUp API Token**
   - **ClickUp Team ID**
   - **Default List ID** (optional fallback)
   - **Enable Due-Date Reminders**

### 3. Link a channel

```
/clickup link 123456789 My Sprint List
```

## Commands

| Command | Description |
|---------|-------------|
| `/clickup help` | Show all commands |
| `/clickup link <list_id> [name]` | Link channel to a ClickUp list |
| `/clickup unlink` | Remove channel link |
| `/clickup tasks` | List open tasks |
| `/clickup task <name>` | Create a task |
| `/clickup create` | Open task creation dialog |
| `/clickup assign <task_id> @user` | Assign task |
| `/clickup done <task_id>` | Mark task complete |
| `/clickup comment <task_id> <text>` | Add comment |
| `/clickup my` | Your assigned tasks |

### Task options

```
/clickup task Fix login bug --assign @john --due 2026-06-20 --priority high
```

## UI shortcuts

- **App bar** — ClickUp icon opens the task panel
- **Channel header** — checklist icon opens tasks for this channel
- **Message menu (⋯)** — **Create ClickUp Task** from any message

## User matching

Assignees are matched by **email** between Mattermost and ClickUp workspace members. Ensure users share the same email in both systems.

## Development

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make dist
make deploy
```

## Support

**[Donate via NOWPayments](https://nowpayments.io/donation/davood)**

## License

MIT — see [LICENSE](LICENSE)

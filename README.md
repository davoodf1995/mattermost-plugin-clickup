# Mattermost ClickUp Plugin

Mattermost plugin for [ClickUp](https://clickup.com) integration, maintained by [devmatika](https://github.com/devmatika).

Connect Mattermost channels to ClickUp lists, create and manage tasks, get real-time notifications, and receive due-date reminders — without leaving your chat.

**Requires Mattermost Server v11.0.0 or later.**

## Features

- **Channel ↔ List linking** — map each Mattermost channel to a ClickUp list
- **Slash command autocomplete** — nested subcommands like `/clickup create`, `/clickup tasks`, …
- **Create tasks** — slash commands, interactive dialog, or from any message
- **Assign tasks** — match Mattermost users to ClickUp members by email
- **Task panel** — ClickUp icon in app bar and channel header
- **Webhooks** — ClickUp events posted to linked channels
- **Due-date reminders** — DM and channel alerts before tasks are due
- **Mobile-friendly** — slash commands and dialogs work on mobile

## Where to get ClickUp credentials

### 1. ClickUp API Token (`pk_...`)

1. Open [ClickUp](https://app.clickup.com)
2. Click your **avatar** (bottom-left) → **Settings**
3. Go to **Apps** (or open [Settings → Apps](https://app.clickup.com/settings/apps))
4. Scroll to **API Token** → click **Generate**
5. Copy the token (starts with `pk_`)
6. Paste it in **System Console → Plugins → ClickUp → ClickUp API Token**

### 2. ClickUp Team ID

Your Team ID is the **first number** in the ClickUp URL after you open a workspace:

```
https://app.clickup.com/12345678/home
                          ^^^^^^^^
                          Team ID = 12345678
```

Paste `12345678` in **ClickUp Team ID** in plugin settings.

### 3. Default List ID (optional)

1. Open the ClickUp **List** you want to use in the browser
2. Look at the URL:

```
https://app.clickup.com/12345678/v/li/901234567890
                                    ^^^^^^^^^^^^^
                                    List ID = 901234567890
```

3. Paste the List ID in **Default List ID** (used when a channel is not linked)
4. Or link per-channel with `/clickup link 901234567890 Sprint Backlog`

## Build and install

```bash
make dist
```

Upload **`dist/com.mattermost.clickup-1.0.0.tar.gz`** via **System Console → Plugins → Upload**.

This file is a **compiled plugin bundle** (binaries + webapp + `plugin.json`), not source code.

### GitHub Releases (important)

GitHub always adds **Source code (zip/tar.gz)** to every release. That archive is the **git repository** and cannot be uploaded to Mattermost.

You need the **`com.mattermost.clickup-*.tar.gz`** from `make dist`, which contains:

```
com.mattermost.clickup/
  plugin.json
  server/dist/plugin-linux-amd64
  server/dist/plugin-linux-arm64
  ...
  webapp/dist/main.js
  assets/
  public/
```

**Manual release:** create a tag → attach `dist/com.mattermost.clickup-*.tar.gz` under **Assets**.

**Automatic release:** push a version tag and CI uploads the bundle:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The [release workflow](.github/workflows/release.yml) runs `make dist` and attaches the correct `.tar.gz` to the GitHub release.

## Setup

## Slash commands

Type `/clickup` and Mattermost shows nested subcommands with the ClickUp icon (like the [Google Drive plugin](https://github.com/mattermost/mattermost-plugin-google-drive)).

| Command | Description |
|---------|-------------|
| `/clickup help` | Show all commands |
| `/clickup link <list_id> [name]` | Link channel to a ClickUp list |
| `/clickup unlink` | Remove channel link |
| `/clickup tasks` | List open tasks |
| `/clickup create` | Open task creation dialog |
| `/clickup task <name>` | Create a task |
| `/clickup assign <task_id> @user` | Assign task |
| `/clickup done <task_id>` | Mark task complete |
| `/clickup comment <task_id> <text>` | Add comment |
| `/clickup my` | Your assigned tasks |

### Task options

```
/clickup task Fix login --assign @john --due 2026-06-20 --priority high
```

## UI shortcuts

- **App bar** — ClickUp logo opens the task panel
- **Channel header** — ClickUp logo opens tasks for this channel
- **Message menu (⋯)** — **Create ClickUp Task**

## User matching

Assignees are matched by **email** between Mattermost and ClickUp. Ensure users share the same email in both systems.

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

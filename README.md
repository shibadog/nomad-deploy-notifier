# nomad-deploy-notifier
send nomad deployment messages to slack

# Notes:
Requires golang 1.15 or newer - older distributions may need an update. Check golang version: `go version`.

# Install:
```
cd cmd/bot
go get
go build
```

# Run:
SLACK_TOKEN=XXXX SLACK_CHANNEL=channel_name_here ./bot 

SLACK_TOKEN should start with xoxb-, from the field Bot User OAuth Access Token found on the OAuth & Permissions page of your custom Slack app.

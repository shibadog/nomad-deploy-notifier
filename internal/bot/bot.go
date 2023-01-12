package bot

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/slack-go/slack"
)

type Config struct {
	Token   string
	Channel string
}

type Bot struct {
	mu           sync.Mutex
	chanID       string
	nomadAddress string
	api          *slack.Client
	deploys      map[string]string
	L            hclog.Logger
}

func NewBot(cfg Config, nomadAddress string) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("no token provided")
	}

	api := slack.New(cfg.Token)

	bot := &Bot{
		api:          api,
		nomadAddress: nomadAddress,
		chanID:       cfg.Channel,
		deploys:      make(map[string]string),
	}

	return bot, nil
}

func (b *Bot) UpsertDeployMsg(deploy api.Deployment) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	ts, ok := b.deploys[deploy.ID]
	if !ok {
		return b.initialDeployMsg(deploy)
	}
	// b.L.Debug("Existing deployment found, updating status", "slack ts", ts)

	attachments := b.DefaultAttachments(deploy)
	opts := []slack.MsgOption{slack.MsgOptionAttachments(attachments...)}
	opts = append(opts, DefaultDeployMsgOpts()...)

	_, ts, _, err := b.api.UpdateMessage(b.chanID, ts, opts...)
	if err != nil {
		return err
	}
	b.deploys[deploy.ID] = ts

	return nil
}

func (b *Bot) initialDeployMsg(deploy api.Deployment) error {
	attachments := b.DefaultAttachments(deploy)

	opts := []slack.MsgOption{slack.MsgOptionAttachments(attachments...)}
	opts = append(opts, DefaultDeployMsgOpts()...)

	_, ts, err := b.api.PostMessage(b.chanID, opts...)
	if err != nil {
		return err
	}
	b.deploys[deploy.ID] = ts
	return nil
}

func DefaultDeployMsgOpts() []slack.MsgOption {
	return []slack.MsgOption{
		slack.MsgOptionAsUser(true),
	}
}

func (b *Bot) DefaultAttachments(deploy api.Deployment) []slack.Attachment {
	var actions []slack.AttachmentAction
	if deploy.StatusDescription == "Deployment is running but requires manual promotion" {
		actions = []slack.AttachmentAction{
			{
				Name: "promote",
				Text: "Promote :heavy_check_mark:",
				Type: "button",
			},
			{
				Name:  "fail",
				Text:  "Fail :boom:",
				Style: "danger",
				Type:  "button",
				Confirm: &slack.ConfirmationField{
					Title:       "あってる?",
					Text:        ":nomad-sad: :nomad-sad: :nomad-sad: :nomad-sad: :nomad-sad:",
					OkText:      "Fail",
					DismissText: "Woops!",
				},
			},
		}
	}
	var fields []slack.AttachmentField
	for tgn, tg := range deploy.TaskGroups {
		field := slack.AttachmentField{
			Title: fmt.Sprintf("Task Group: %s", tgn),
			Value: fmt.Sprintf("Healthy: %d, 配置数: %d, カナリア: %d", tg.HealthyAllocs, tg.PlacedAllocs, tg.DesiredCanaries),
		}
		fields = append(fields, field)
	}
	return []slack.Attachment{
		{
			Fallback:   "deployment update",
			Color:      colorForStatus(deploy.Status),
			AuthorName: fmt.Sprintf("%sのデプロイで更新がありました。", deploy.JobID),
			AuthorLink: fmt.Sprintf("%s/ui/jobs/%s/deployments", b.nomadAddress, deploy.JobID),
			Title:      jpMessageFoStatusDescription(deploy.StatusDescription),
			TitleLink:  fmt.Sprintf("%s/ui/jobs/%s/deployments", b.nomadAddress, deploy.JobID),
			Fields:     fields,
			Footer:     fmt.Sprintf("Deploy ID: %s", deploy.ID),
			Ts:         json.Number(fmt.Sprintf("%d", time.Now().Unix())),
			Actions:    actions,
		},
	}
}

func colorForStatus(status string) string {
	switch status {
	case "failed":
		return "#dd4e58"
	case "running":
		return "#1daeff"
	case "successful":
		return "#36a64f"
	default:
		return "#D3D3D3"
	}
}

func jpMessageFoStatusDescription(statusDescription string) string {
	switch statusDescription {
	case "Deployment completed successfully":
		return "デプロイメントが正常に完了しました。"
	case "Failed due to progress deadline":
		return "タイムアウトしたため失敗しました。"
	case "Deployment is running":
		return "デプロイメントが開始されました。"
	default:
		return statusDescription
	}
}

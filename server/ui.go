package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v5/model"
)

func uiHelper(title, text, command, yesLabel, rootId string) (string, []*model.SlackAttachment) {
	return "", []*model.SlackAttachment{
		{
			Color: "#ff0000",
			Title: title,
			Text:  text,
			Actions: []*model.PostAction{
				{
					Type:  model.POST_ACTION_TYPE_BUTTON,
					Name:  "Cancel",
					Style: "primary",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/v1/echo", PLUGIN_ID),
						Context: map[string]interface{}{
							"message": fmt.Sprintf("Canceled command `%s`.", command),
						},
					},
				},
				{
					Type:  model.POST_ACTION_TYPE_BUTTON,
					Name:  yesLabel,
					Style: "danger",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/v1/confirm", PLUGIN_ID),
						Context: map[string]interface{}{
							"command": command,
							"root_id": rootId,
						},
					},
				},
			},
		},
	}
}

func uiConfirmation(text, command, rootId string) (string, []*model.SlackAttachment) {
	return uiHelper("Warning", text, command, "Yes", rootId)
}

func uiError(text, command, rootId string) (string, []*model.SlackAttachment) {
	return uiHelper("Error", text, command, "Try again", rootId)
}

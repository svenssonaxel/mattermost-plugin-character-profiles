package main

import (
	"regexp"
	"sync"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) MessageWillBePosted(_ *plugin.Context, post *model.Post) (*model.Post, string) {
	return p.NpcPost(post)
}

func (p *Plugin) MessageWillBeUpdated(_ *plugin.Context, newPost *model.Post, _ *model.Post) (*model.Post, string) {
	return p.NpcPost(newPost)
}

func (p *Plugin) NpcPost(post *model.Post) (*model.Post, string) {
	// Only touch posts created by users
	if post.IsSystemMessage() || post.UserId == "" {
		return nil, ""
	}
	// Check if the post message matches the regex pattern
	pattern := regexp.MustCompile(`(?s)^npc ([a-z]+): ?(.*)$`)
	matches := pattern.FindStringSubmatch(post.Message)
	ret := post.Clone()
	if len(matches) == 3 {
		// This is an NPC post, so override the displayed user name and message
		ret.AddProp("override_username", matches[1])
		ret.AddProp("from_webhook", "true") // Unfortunately we need to pretend this is from a bot, or the username won't get overridden.
		//ret.Props["override_icon_url"] = "https://example.com/my-custom-bot-icon.png"
		ret.MessageSource = ret.Message
		ret.Message = matches[2]
		return ret, ""
	} else {
		// This is a user post, so make sure to use the defaults.
		ret.DelProp("override_username")
		ret.AddProp("from_webhook", "false")
		return ret, ""
	}
}

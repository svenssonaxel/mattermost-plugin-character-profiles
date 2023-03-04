package main

import (
	"regexp"
	"sync"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
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
	return p.ProfiledPost(post)
}

func (p *Plugin) MessageWillBeUpdated(_ *plugin.Context, newPost *model.Post, _ *model.Post) (*model.Post, string) {
	return p.ProfiledPost(newPost)
}

func (p *Plugin) ProfiledPost(post *model.Post) (*model.Post, string) {
	userId := post.UserId
	// Only touch posts created by users
	if post.IsSystemMessage() || post.UserId == "" {
		return nil, ""
	}
	// Clone before altering
	ret := post.Clone()

	// Check if the post message matches the regex pattern
	matches := regexp.MustCompile(`(?s)^([a-z]+):[ \n](.*)$`).FindStringSubmatch(post.Message)
	if len(matches) == 3 {
		// This might be a one-off post.
		profileId := matches[1]
		actualMessage := matches[2]
		if isMe(profileId) {
			ret.Message = actualMessage
			return mePost(ret), ""
		}
		profile, err := p.getProfile(userId, profileId)
		// todo: Distinguish between not found and other errors.
		if err == nil && profile != nil {
			// We found a matching profile, so this is an actual one-off post.
			ret.Message = actualMessage
			return profilePost(ret, profile), ""
		}
	}

	// Check if a profile identifier is already set. This is to not randomly change profile when editing a post.
	oldProfileIdentifier, opiOk := ret.Props["profile_identifier"]
	if opiOk {
		oldProfileIdentifierStr, ok := oldProfileIdentifier.(string)
		if ok {
			profile, err := p.getProfile(userId, oldProfileIdentifierStr)
			// todo: Distinguish between not found and other errors.
			if err == nil && profile != nil {
				// We found a matching profile, so let's update the post with the current settings.
				return profilePost(ret, profile), ""
			} else {
				// We didn't find a matching profile but we can't change it, so let it be as it is.
				return nil, ""
			}
		}
	}

	// This is a new post that should be sent using the default character profile.
	channelId := post.ChannelId
	profileId, err := p.getDefaultProfileIdentifier(userId, channelId)
	if err == nil && !isMe(profileId) {
		profile, err := p.getProfile(userId, profileId)
		// todo: Distinguish between not found and other errors.
		if err == nil && profile != nil {
			// We found a matching profile, so let's apply it to the post.
			return profilePost(ret, profile), ""
		}
	}

	// The default is to send as yourself.
	return mePost(ret), ""
}

func mePost(post *model.Post) *model.Post {
	post.AddProp("profile_identifier", "myself")
	post.AddProp("override_username", nil)
	post.AddProp("override_icon_url", nil)
	post.AddProp("from_webhook", nil)
	return post
}

func profilePost(post *model.Post, profile *Profile) *model.Post {
	post.AddProp("profile_identifier", profile.Identifier)
	post.AddProp("override_username", profile.Name)
	post.AddProp("override_icon_url", profile.IconUrl)
	post.AddProp("from_webhook", "true") // Unfortunately we need to pretend this is from a bot, or the username won't get overridden.
	return post
}

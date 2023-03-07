package main

import (
	"net/http"
	"path/filepath"
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
	return p.ProfiledPost(post, false)
}

func (p *Plugin) MessageWillBeUpdated(_ *plugin.Context, newPost *model.Post, _ *model.Post) (*model.Post, string) {
	return p.ProfiledPost(newPost, true)
}

func (p *Plugin) ProfiledPost(post *model.Post, isedited bool) (*model.Post, string) {
	// Shouldn't really happen.
	if post == nil {
		return nil, ""
	}
	userId := post.UserId
	// Only touch posts created by users
	if post.IsSystemMessage() || post.UserId == "" {
		return nil, ""
	}
	// Clone before altering
	ret := post.Clone()

	// Handle one-off profiled posts
	matches := regexp.MustCompile(`(?s)^([a-z]+):[ \n](.*)$`).FindStringSubmatch(post.Message)
	if len(matches) == 3 {
		// This might be a one-off post.
		profileId := matches[1]
		actualMessage := matches[2]
		if isMe(profileId) {
			ret.Message = actualMessage
			return mePost(ret)
		}
		profile, err := p.getProfile(userId, profileId, true)
		if err == nil && profile != nil {
			// We found a matching profile, so this is an actual one-off post.
			ret.Message = actualMessage
			return profilePost(ret, profile)
		}
	}

	// Handle edited posts that are not one-off.
	oldProfileIdentifier, opiOk := ret.Props["profile_identifier"]
	if opiOk {
		oldProfileIdentifierStr, ok := oldProfileIdentifier.(string)
		if ok {
			profile, err := p.getProfile(userId, oldProfileIdentifierStr, true)
			if err == nil && profile != nil {
				// We found a matching profile, so let's update the post with the current settings.
				return profilePost(ret, profile)
			}
		}
	}
	if isedited {
		// We didn't find a matching profile but we can't change it, so let it be as it is.
		return nil, ""
	}

	// Handle new posts affected by channel default profile
	channelId := post.ChannelId
	profileId, err := p.getDefaultProfileIdentifier(userId, channelId)
	if err == nil && !isMe(profileId) {
		profile, err := p.getProfile(userId, profileId, true)
		if err == nil && profile != nil {
			// We found a matching profile, so let's apply it to the post.
			return profilePost(ret, profile)
		}
	}

	// The default is to send as yourself.
	return mePost(ret)
}

func mePost(post *model.Post) (*model.Post, string) {
	post.AddProp("profile_identifier", "myself")
	post.AddProp("override_username", nil)
	post.AddProp("override_icon_url", nil)
	post.AddProp("from_webhook", nil)
	return post, ""
}

func profilePost(post *model.Post, profile *Profile) (*model.Post, string) {
	// Send a normal message with the selected profile
	post.AddProp("profile_identifier", profile.Identifier)
	post.AddProp("override_username", profile.Name)
	post.AddProp("override_icon_url", profileIconUrl(profile.PictureFileId, false))
	post.AddProp("from_webhook", "true") // Unfortunately we need to pretend this is from a bot, or the username won't get overridden.
	return post, ""
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		p.API.LogWarn("Failed to get bundle path", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filepath.Join(bundlePath, "assets", r.URL.Path))
}

func profileIconUrl(fileId string, thumbnail bool) string {
	if fileId == "" {
		if thumbnail {
			return "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/character-thumbnail.jpeg"
		}
		return "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/character.png"
	}
	if thumbnail {
		return "/api/v4/files/" + fileId + "/thumbnail"
	}
	return "/api/v4/files/" + fileId
}

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

	// Mockable backend, the only thing passed to non-glue code.
	backend *Backend
}

func getRealBackendFromPlugin(p *Plugin) (Backend, *model.AppError) {
	backend := BackendImpl{}
	maybeSiteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if maybeSiteURL == nil {
		return backend, model.NewAppError("backendFromPlugin", "Cannot get Site URL", nil, "", http.StatusInternalServerError)
	}
	backend.SiteURL = *maybeSiteURL
	if p.API == nil {
		return backend, model.NewAppError("backendFromPlugin", "Cannot get API", nil, "", http.StatusInternalServerError)
	}
	backend.API = p.API
	return backend, nil
}

func (p *Plugin) OnActivate() error {
	backend, beErr := getRealBackendFromPlugin(p)
	if beErr != nil {
		return beErr
	}
	p.backend = &backend
	err := p.API.RegisterCommand(&model.Command{
		Trigger:          "character",
		Description:      "Become a nomad of names, a litany of labels, to master monikers and fabricate fables.",
		DisplayName:      "Character profiles",
		AutoComplete:     true,
		AutoCompleteDesc: "Try `/character help` to become a nomad of names, a litany of labels, to master monikers and fabricate fables.",
		AutoCompleteHint: "haddock=Captain Haddock",
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) MessageWillBePosted(_ *plugin.Context, post *model.Post) (*model.Post, string) {
	if p.backend == nil {
		return nil, "Backend not initialized"
	}
	return ProfiledPost(*p.backend, post, false)
}

func (p *Plugin) MessageWillBeUpdated(_ *plugin.Context, newPost *model.Post, _ *model.Post) (*model.Post, string) {
	if p.backend == nil {
		return nil, "Backend not initialized"
	}
	return ProfiledPost(*p.backend, newPost, true)
}

func ProfiledPost(be Backend, post *model.Post, isedited bool) (*model.Post, string) {
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
		profile, err := getProfile(be, userId, profileId, PROFILE_CHARACTER|PROFILE_ME)
		if err == nil && profile != nil {
			// We found a matching profile, so this is an actual one-off post.
			ret.Message = actualMessage
			return profilePost(be, ret, *profile)
		}
	}

	// Handle edited posts that are not one-off.
	oldProfileIdentifier, opiOk := ret.Props["profile_identifier"]
	if opiOk {
		oldProfileIdentifierStr, ok := oldProfileIdentifier.(string)
		if ok {
			profile, err := getProfile(be, userId, oldProfileIdentifierStr, PROFILE_CHARACTER)
			if err == nil && profile != nil {
				// We found a matching profile, so let's update the post with the current settings.
				return profilePost(be, ret, *profile)
			}
		}
	}
	if isedited {
		// We didn't find a matching profile but we can't change it, so let it be as it is.
		return nil, ""
	}

	// Handle new posts
	channelId := post.ChannelId
	profileId, err := getDefaultProfileIdentifier(be, userId, channelId)
	if err == nil {
		profile, err := getProfile(be, userId, profileId, PROFILE_CHARACTER|PROFILE_ME)
		if err == nil && profile != nil {
			// We found a matching profile, so let's apply it to the post.
			return profilePost(be, ret, *profile)
		}
	}

	// This shouldn't happen, but if it does let's not make a fuss.
	return nil, ""
}

func profilePost(be Backend, post *model.Post, profile Profile) (*model.Post, string) {
	// Send a normal message with the selected profile
	switch profile.Status {
	case PROFILE_ME:
		post.AddProp("profile_identifier", "myself")
		post.AddProp("override_username", nil)
		post.AddProp("override_icon_url", nil)
		post.AddProp("from_webhook", nil)
		return post, ""
	case PROFILE_CHARACTER:
		post.AddProp("profile_identifier", profile.Identifier)
		post.AddProp("override_username", profile.Name)
		post.AddProp("override_icon_url", profileIconUrl(be, profile, false))
		post.AddProp("from_webhook", "true") // Unfortunately we need to pretend this is from a bot, or the username won't get overridden.
		return post, ""
	default:
		return nil, "Invalid profile status"
	}
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

func profileIconUrl(be Backend, profile Profile, thumbnail bool) string {
	siteURL := be.GetSiteURL()
	if profile.Status == PROFILE_CHARACTER {
		fileId := profile.PictureFileId
		if fileId == "" {
			if thumbnail {
				return siteURL + "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/character-thumbnail.jpeg"
			}
			return siteURL + "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/character.png"
		}
		if thumbnail {
			return siteURL + "/api/v4/files/" + fileId + "/thumbnail"
		}
		return siteURL + "/api/v4/files/" + fileId
	}
	if profile.Status == PROFILE_ME {
		return siteURL + "/api/v4/users/" + profile.UserId + "/image" // todo how to get thumbnail?
	}
	if thumbnail {
		return siteURL + "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/no-sign-thumbnail.jpg"
	}
	return siteURL + "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/no-sign.jpg"
}

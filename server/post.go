package main

import (
	"regexp"

	"github.com/mattermost/mattermost-server/v5/model"
)

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

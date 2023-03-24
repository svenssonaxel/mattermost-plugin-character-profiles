package main

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/mattermost/mattermost-server/v5/model"
)

// RegisterPost adds a post to the corresponding id set.
func RegisterPost(be Backend, post *model.Post) *model.AppError {
	if post == nil {
		return appError("Post is nil", nil)
	}
	profileId := ""
	profileIdRaw, ok := post.Props["profile_identifier"]
	if ok {
		profileId, ok = profileIdRaw.(string)
		if !ok {
			profileId = ""
		}
	}
	if profileId != "" {
		key := getIdsetKey(post.UserId, profileId)
		addErr := IdsetInsert(be, key, post.Id)
		if addErr != nil {
			return addErr
		}
	}
	return nil
}

// ProfiledPost decides which profile to apply to the given post based on its
// Message, Props and whether it's edited. It returns the post with the profile
// applied, potentially with a prefix removed from the message.
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

// profilePost returns a post with the given profile applied.
func profilePost(be Backend, post *model.Post, profile Profile) (*model.Post, string) {
	// Send a normal message with the selected profile
	switch profile.Status {
	case PROFILE_ME:
		post.AddProp("profile_identifier", nil)
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

// Handle id sets

func getIdsetKey(userId, profileId string) string {
	return fmt.Sprintf("profiledpost_%s_%s", userId, profileId)
}

func updatePostsUsingProfile(be Backend, userId, profileId string) *model.AppError {
	pre := fmt.Sprintf("updatePostsUsingProfile(%s, %s): ", userId, profileId)
	profile, err := getProfile(be, userId, profileId, PROFILE_CHARACTER)
	if err != nil {
		return appErrorPre(pre, err)
	}
	key := getIdsetKey(userId, profileId)
	err = IdsetIter(be, key, "", 0, func(postId string) *model.AppError {
		post, err := GetPostIfExists(be, postId)
		if err != nil {
			return err
		}
		if post == nil {
			// API did not return a post, so it must have been deleted. That's fine,
			// we'll just ignore it.
			return nil
		}
		if post.UserId != userId {
			return appError(fmt.Sprintf("Found post with userId \"%s\" but expected \"%s\"", post.UserId, userId), nil)
		}
		profileIdOfPost, ok := post.Props["profile_identifier"]
		if !ok {
			return appError(fmt.Sprintf("Post \"%s\" has no profile_identifier", postId), nil)
		}
		if profileIdOfPost == nil {
			profileIdOfPost = ""
		}
		profileIdOfPostStr, ok := profileIdOfPost.(string)
		if !ok {
			return appError(fmt.Sprintf("Post \"%s\" has a profile_identifier that is not null or string", postId), nil)
		}
		if profileIdOfPostStr != profileId {
			// This post is not using the profile we're updating, so skip it. This can
			// happen if the post was edited to use a different profile.
			return nil
		}
		profiledPost, errStr := profilePost(be, post, *profile)
		if errStr != "" {
			return appError(errStr, nil)
		}
		_, err = be.UpdatePost(profiledPost)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return appErrorPre(pre, err)
	}
	return nil
}

// GetPostIfExists returns the post with the given id, or nil if it does not
// exist. This differs from be.GetPost, which returns an error if the post does
// not exist.
func GetPostIfExists(be Backend, postId string) (*model.Post, *model.AppError) {
	post, err := be.GetPost(postId)
	if err != nil && err.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, appErrorPre("GetPostIfExists: ", err)
	}
	return post, nil
}

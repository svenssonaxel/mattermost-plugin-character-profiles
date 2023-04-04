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
		return appError("Message is nil", nil)
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
	ret := DeepClonePost(post)

	// Handle one-off profiled posts
	matches := regexp.MustCompile(`(?s)^([a-z]+):[ \n](.*)$`).FindStringSubmatch(post.Message)
	if matches != nil {
		// This might be a one-off post.
		profileId := matches[1]
		actualMessage := matches[2]
		profile, err := GetProfile(be, userId, profileId, PROFILE_CHARACTER|PROFILE_ME)
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
			profile, err := GetProfile(be, userId, oldProfileIdentifierStr, PROFILE_CHARACTER)
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
		profile, err := GetProfile(be, userId, profileId, PROFILE_CHARACTER|PROFILE_ME)
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

// updatePostsForProfile updates all posts of the given user that use the given
// profile identifier to use the new profile identifier. Provide an empty string
// for newProfileId to remove the profile identifier from the posts and instead
// use the default profile. Provide the same value for oldProfileId and
// newProfileId to update the display name and icon of the posts.
func updatePostsForProfile(be Backend, userId, oldProfileId, newProfileId string) *model.AppError {
	pre := fmt.Sprintf("updatePostsUsingProfile(%s, %s, %s)", userId, oldProfileId, newProfileId)
	if IsMe(oldProfileId) {
		return appError(pre+"Cannot update message that are using the user's real profile.", nil)
	}
	newProfile, err := GetProfile(be, userId, newProfileId, PROFILE_CHARACTER|PROFILE_ME)
	if err != nil {
		return appErrorPre(pre, err)
	}
	oldKey := getIdsetKey(userId, oldProfileId)
	err = IdsetIter(be, oldKey, "", 0, func(postId string) *model.AppError {
		post, gpErr := GetPostIfExists(be, postId)
		if gpErr != nil {
			return gpErr
		}
		if post == nil {
			// API did not return a post, so it must have been deleted. That's fine,
			// we'll just ignore it.
			return nil
		}
		if post.UserId != userId {
			return appError(fmt.Sprintf("Found message with userId \"%s\" but expected \"%s\"", post.UserId, userId), nil)
		}
		profileIdOfPost, ok := post.Props["profile_identifier"]
		if !ok {
			return appError(fmt.Sprintf("Message \"%s\" has no profile_identifier", postId), nil)
		}
		if profileIdOfPost == nil {
			profileIdOfPost = ""
		}
		profileIdOfPostStr, ok := profileIdOfPost.(string)
		if !ok {
			return appError(fmt.Sprintf("Message \"%s\" has a profile_identifier that is not null or string", postId), nil)
		}
		if profileIdOfPostStr != oldProfileId {
			// This post is not using the profile we're updating, so skip it. This can
			// happen if the post was edited to use a different profile.
			return nil
		}
		profiledPost, errStr := profilePost(be, DeepClonePost(post), *newProfile)
		if errStr != "" {
			return appError(errStr, nil)
		}
		// If post is unchanged, don't update it.
		if profiledPost.Message == post.Message &&
			profiledPost.Props["profile_identifier"] == post.Props["profile_identifier"] &&
			profiledPost.Props["override_username"] == post.Props["override_username"] &&
			profiledPost.Props["override_icon_url"] == post.Props["override_icon_url"] &&
			profiledPost.Props["from_webhook"] == post.Props["from_webhook"] {
			return nil
		}
		// Update the post. This will also insert it into the new idset.
		_, err = be.UpdatePost(profiledPost)
		if err != nil {
			return err
		}
		if profiledPost.Props["profile_identifier"] != oldProfileId {
			// The post was updated to use a different profile, so remove it from the
			// old idset. This is ok to do inside the iteration; we will not miss any
			// unrelated ids.
			err = IdsetRemove(be, oldKey, postId)
			if err != nil {
				return err
			}
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

// DeepClonePost returns a clone of the given post that is safe to modify. This
// is under the assumption that only Props and shallow fields are modified.
func DeepClonePost(post *model.Post) *model.Post {
	ret := post.Clone()
	ret.Props = make(map[string]interface{})
	for k, v := range post.Props {
		ret.Props[k] = v
	}
	return ret
}

// countPostsForProfile returns the number of posts that use the given profile
func countPostsForProfile(be Backend, userId, profileId string) (int, *model.AppError) {
	pre := fmt.Sprintf("countPostsForProfile(%s, %s): ", userId, profileId)
	if IsMe(profileId) {
		return 0, appError(pre+"Cannot count messages that are using the user's real profile.", nil)
	}
	key := getIdsetKey(userId, profileId)
	count := 0
	err := IdsetIter(be, key, "", 0, func(postId string) *model.AppError {
		post, err := GetPostIfExists(be, postId)
		if err != nil {
			return err
		}
		if post == nil {
			// API did not return a post, so it must have been deleted. That's fine,
			// we'll just ignore it.
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		return 0, appErrorPre(pre, err)
	}
	return count, nil
}

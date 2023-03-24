package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	perPage = 50
)

const (
	PROFILE_CHARACTER   = 0x1
	PROFILE_ME          = 0x2
	PROFILE_CORRUPT     = 0x4
	PROFILE_NONEXISTENT = 0x8
)

type Profile struct {
	UserId          string          `json:"-"`           // not stored
	Identifier      string          `json:"-"`           // not stored
	Name            string          `json:"displayName"` // todo rename to DisplayName
	PictureFileId   string          `json:"pictureFile"`
	PictureFileInfo *model.FileInfo `json:"-"`          // not stored
	PicturePost     *model.Post     `json:"-"`          // not stored
	Status          int             `json:"-"`          // not stored. Can be any of PROFILE_*.
	Error           *model.AppError `json:"-"`          // not stored. Must be set if Status == PROFILE_NONEXISTENT || Status == PROFILE_CORRUPTED.
	RequestKey      string          `json:"requestKey"` // Used to authorize HTTP requests for the profile picture, as well as force a cache miss.
}

func populateProfile(be Backend, profile *Profile) *model.AppError {
	if profile.PictureFileId == "" {
		return nil
	}
	pre := fmt.Sprintf("Failed to populate profile `%s`: ", profile.Identifier)
	var err *model.AppError
	if profile.PictureFileInfo == nil {
		profile.PictureFileInfo, err = be.GetFileInfo(profile.PictureFileId)
		if err != nil {
			return appErrorPre(pre, err)
		}
	}
	if profile.PictureFileInfo == nil {
		return appError(pre+"Could not find information about the profile picture file.", nil)
	}
	if profile.PicturePost == nil {
		profile.PicturePost, err = GetPostIfExists(be, profile.PictureFileInfo.PostId)
		if err != nil {
			return appErrorPre(pre, err)
		}
		if profile.PicturePost == nil {
			return appError(pre+"The post supposedly holding the profile picture could not be found, perhaps it's deleted.", nil)
		}
	}
	return nil
}

func (profile *Profile) validate(profileId string) *model.AppError {
	pre := fmt.Sprintf("Failed validating profile `%s`: ", profileId)
	if profile == nil {
		return appError(pre+"Profile is nil.", nil)
	}
	if profile.Identifier != profileId {
		return appError(pre+"Identifier mismatch.", nil)
	}
	matches := regexp.MustCompile(`^[a-z]{1,60}$`).FindStringSubmatch(profile.Identifier)
	if len(matches) != 1 {
		return appError(pre+"Identifier must be 1-60 lowercase letters a-z.", nil)
	}
	matches = regexp.MustCompile("^[^-*|`>#*_.~[\\]]{1,200}$").FindStringSubmatch(profile.Name)
	if len(matches) != 1 {
		return appError(pre+"Display name must be 1-200 characters and must not contain format control characters.", nil)
	}
	if profile.PictureFileId == "" {
		if profile.PictureFileInfo != nil {
			return appError(pre+"PictureFileInfo has a value despite no PictureFileId.", nil)
		}
		if profile.PicturePost != nil {
			return appError(pre+"PicturePost has a value despite no PictureFileId.", nil)
		}
		if profile.RequestKey != "" {
			return appError(pre+"RequestKey has a value despite no PictureFileId.", nil)
		}
	} else {
		file := profile.PictureFileInfo
		if !file.IsImage() {
			return appError(fmt.Sprintf("%sThe file \"%s\" is not recognized as an image file.", pre, file.Name), nil)
		}
		ext := file.Extension
		matches := regexp.MustCompile(`(?i)^(jpe?g|png)$`).FindStringSubmatch(ext)
		if len(matches) == 0 {
			return appError(fmt.Sprintf("%sThe file extension \"%s\" is not valid for a profile picture. Only .JPG, .JPEG and .PNG are acceptable.", pre, ext), nil)
		}
		err := file.IsValid()
		if err != nil {
			return appErrorPre(pre, err)
		}
		post := profile.PicturePost
		if post == nil {
			return appError(pre+"The post supposedly holding the profile picture is nil.", nil)
		}
		if post.DeleteAt != 0 {
			// This probably can't happen because the API doesn't return deleted posts.
			return appError(pre+"The post supposedly holding the profile picture is deleted.", nil)
		}
		if len(post.FileIds) != 1 {
			return appError(pre+"The post supposedly holding the profile picture does not have exactly 1 file.", nil)
		}
		if post.FileIds[0] != profile.PictureFileId {
			return appError(pre+"The post supposedly holding the profile picture does not hold the expected file.", nil)
		}
		if profile.RequestKey == "" {
			return appError(pre+"RequestKey is empty despite PictureFileId being set.", nil)
		}
	}
	switch profile.Status {
	case PROFILE_CHARACTER:
		if profile.Error != nil {
			return appError(pre+"Error is set despite status being PROFILE_CHARACTER.", nil)
		}
		if isMe(profile.Identifier) {
			return appError(pre+"Identifier indicates real profile despite status being PROFILE_CHARACTER.", nil)
		}
		break
	case PROFILE_ME:
		if profile.Error != nil {
			return appError(pre+"Error is set despite status being PROFILE_ME.", nil)
		}
		if !isMe(profile.Identifier) {
			return appError(pre+"Identifier does not indicate real profile despite status being PROFILE_ME.", nil)
		}
		break
	default:
		return appError(pre+"Status is not PROFILE_CHARACTER or PROFILE_ME.", nil)
	}
	return nil
}

// EncodeToByte returns a profile as a byte array
func (profile *Profile) EncodeToByte() []byte {
	b, _ := json.Marshal(profile)
	return b
}

// DecodeProfileFromByte tries to create a Profile from a byte array
func DecodeProfileFromByte(b []byte) (*Profile, *model.AppError) {
	profile := Profile{}
	err := json.Unmarshal(b, &profile)
	if err != nil {
		return nil, appError("Failed to decode profile", err)
	}
	return &profile, nil
}

func getProfileKey(userId, profileId string) string {
	return fmt.Sprintf("profile_%s_%s", userId, profileId)
}

func profileExists(be Backend, userId, profileId string) (bool, *model.AppError) {
	return StrsetHas(be, ProfileIdsKey(userId), profileId)
}

func GetProfile(be Backend, userId, profileId string, accepted int) (*Profile, *model.AppError) {
	// Handle the real profile
	if isMe(profileId) {
		if accepted&PROFILE_ME != 0 {
			user, err := be.GetUser(userId)
			if err != nil {
				return nil, err
			}
			if user == nil {
				return nil, appError("Could not fetch user.", nil)
			}
			return &Profile{
				UserId:     userId,
				Identifier: profileId,
				Name:       user.GetDisplayName(model.SHOW_FULLNAME), // Todo options are: model.SHOW_USERNAME, model.SHOW_FULLNAME, model.SHOW_NICKNAME_FULLNAME
				Status:     PROFILE_ME,
			}, nil
		} else {
			return nil, appError(fmt.Sprintf("Profile identifier `%s` refers to the real profile.", profileId), nil)
		}
	}

	// Try to fetch profile
	b, err := be.KVGet(getProfileKey(userId, profileId))
	if err != nil {
		return nil, err
	}

	// Here, we could call `profileExists` to check if the profile exists in the
	// id list, but that would require an extra database call and is not
	// necessary.

	// Handle nonexistent profile
	if b == nil {
		nonexistentErr := appError(fmt.Sprintf("Profile `%s` does not exist.", profileId), nil)
		if accepted&PROFILE_NONEXISTENT != 0 {
			return &Profile{
				UserId:     userId,
				Identifier: profileId,
				Status:     PROFILE_NONEXISTENT,
				Error:      nonexistentErr,
			}, nil
		} else {
			return nil, nonexistentErr
		}
	}

	// Decode
	profile, corruptionErr := DecodeProfileFromByte(b)
	// Handle character and corrupt profile
	if corruptionErr == nil && profile == nil {
		corruptionErr = appError(fmt.Sprintf("Profile `%s` failed to decode and needs to be recreated.", profileId), nil)
	}
	if profile == nil {
		profile = &Profile{}
	}
	profile.UserId = userId
	profile.Identifier = profileId
	profile.Status = PROFILE_CHARACTER
	populateErr := populateProfile(be, profile)
	corruptionPre := fmt.Sprintf("Profile `%s` is corrupt and needs to be recreated: ", profileId)
	if populateErr != nil {
		corruptionErr = appErrorPre(corruptionPre, populateErr)
	}
	if corruptionErr == nil {
		validateErr := profile.validate(profileId)
		if validateErr != nil {
			corruptionErr = appErrorPre(corruptionPre, validateErr)
		}
	}
	if corruptionErr != nil {
		if accepted&PROFILE_CORRUPT != 0 {
			profile.Status = PROFILE_CORRUPT
			profile.Error = corruptionErr
			return profile, nil
		} else {
			return nil, corruptionErr
		}
	}
	if accepted&PROFILE_CHARACTER != 0 {
		return profile, nil
	} else {
		return nil, appError(fmt.Sprintf("Profile identifier `%s` refers to a character profile.", profileId), nil)
	}
}

func setProfile(be Backend, userId string, profile *Profile) *model.AppError {
	err := populateProfile(be, profile)
	if err != nil {
		return err
	}
	err = profile.validate(profile.Identifier)
	if err != nil {
		return err
	}
	err = be.KVSet(getProfileKey(userId, profile.Identifier), profile.EncodeToByte())
	if err != nil {
		return err
	}
	err = StrsetInsert(be, ProfileIdsKey(userId), profile.Identifier)
	if err != nil {
		return err
	}
	return nil
}

func deleteProfile(be Backend, userId, profileId string) *model.AppError {
	err := StrsetRemove(be, ProfileIdsKey(userId), profileId)
	if err != nil {
		return err
	}
	return be.KVDelete(getProfileKey(userId, profileId))
}

// Get an array of all character profiles, and also the real one.
func listProfiles(be Backend, userId string) ([]Profile, *model.AppError) {
	keys, err := StrsetGet(be, ProfileIdsKey(userId))
	if err != nil {
		return nil, err
	}
	ret := make([]Profile, 0)
	for _, key := range keys {
		profile, err := GetProfile(be, userId, key, PROFILE_CHARACTER|PROFILE_CORRUPT)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *profile)
	}
	profile, err := GetProfile(be, userId, "", PROFILE_ME)
	if err != nil {
		return nil, err
	}
	ret = append(ret, *profile)
	sortProfiles(ret)
	return ret, nil
}

// Sort profiles by identifier, except put the real profile last.
func sortProfiles(profiles []Profile) {
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Status == PROFILE_ME {
			return false
		}
		if profiles[j].Status == PROFILE_ME {
			return true
		}
		return profiles[i].Identifier < profiles[j].Identifier
	})
}

func getDefaultProfileKey(userId, channelId string) string {
	return fmt.Sprintf("defaultprofile_%s_%s", userId, channelId)
}

func removeDefaultProfile(be Backend, userId, channelId string) *model.AppError {
	return be.KVDelete(getDefaultProfileKey(userId, channelId))
}

func getDefaultProfileIdentifier(be Backend, userId, channelId string) (string, *model.AppError) {
	b, err := be.KVGet(getDefaultProfileKey(userId, channelId))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func setDefaultProfileIdentifier(be Backend, userId, channelId, profileId string) (*Profile, *model.AppError) {
	profile, err := GetProfile(be, userId, profileId, PROFILE_CHARACTER)
	if err != nil {
		return nil, err
	}
	err = be.KVSet(getDefaultProfileKey(userId, channelId), []byte(profileId))
	if err != nil {
		return nil, appError("", err)
	}
	return profile, nil
}

func profileIconUrl(be Backend, profile Profile, thumbnail bool) string {
	siteURL := be.GetSiteURL()
	pluginURL := siteURL + "/plugins/com.axelsvensson.mattermost-plugin-character-profiles"
	if profile.Status == PROFILE_CHARACTER {
		fileId := profile.PictureFileId
		userId := profile.UserId
		profileId := profile.Identifier
		if fileId == "" {
			if thumbnail {
				return fmt.Sprintf("%s/static/character-thumbnail.jpeg", pluginURL)
			}
			return fmt.Sprintf("%s/static/character.jpeg", pluginURL)
		}
		if thumbnail {
			return fmt.Sprintf("%s/profile/%s/%s/thumbnail?rk=%s", pluginURL, userId, profileId, profile.RequestKey)
		}
		return fmt.Sprintf("%s/profile/%s/%s?rk=%s", pluginURL, userId, profileId, profile.RequestKey)
	}
	if profile.Status == PROFILE_ME {
		return fmt.Sprintf("%s/api/v4/users/%s/image", siteURL, profile.UserId) // todo how to get thumbnail?
	}
	if thumbnail {
		return fmt.Sprintf("%s/static/no-sign-thumbnail.jpg", pluginURL)
	}
	return fmt.Sprintf("%s/static/no-sign.jpg", pluginURL)
}

func ProfileIdsKey(userId string) string {
	return fmt.Sprintf("profilelist_%s", userId)
}

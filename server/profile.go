package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	perPage = 50
)

type Profile struct {
	Identifier      string          `json:"id"`          // todo rename to Id
	Name            string          `json:"displayName"` // todo rename to DisplayName
	PictureFileId   string          `json:"pictureFile"`
	PictureFileInfo *model.FileInfo `json:"-"` // not stored
	PicturePost     *model.Post     `json:"-"` // not stored
}

func (p *Plugin) populateProfile(profile *Profile) *model.AppError {
	if profile.PictureFileId == "" {
		return nil
	}
	pre := fmt.Sprintf("Failed to populate profile `%s`: ", profile.Identifier)
	var err *model.AppError
	if profile.PictureFileInfo == nil {
		profile.PictureFileInfo, err = p.API.GetFileInfo(profile.PictureFileId)
		if err != nil {
			return appErrorPre(pre, err)
		}
	}
	if profile.PictureFileInfo == nil {
		return appError(pre+"Could not populate PictureFileInfo", nil)
	}
	if profile.PicturePost == nil {
		profile.PicturePost, err = p.API.GetPost(profile.PictureFileInfo.PostId)
		if err != nil {
			return appErrorPre(pre, err)
		}
		if profile.PicturePost == nil {
			return appError(pre+"Could not populate PicturePost", nil)
		}
	}
	return nil
}

func (p *Plugin) validateProfile(profile *Profile, profileId string) *model.AppError {
	pre := fmt.Sprintf("Failed validating profile `%s`: ", profileId)
	var err *model.AppError
	err = p.populateProfile(profile)
	if err != nil {
		return appErrorPre(pre, err)
	}
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
		if post.DeleteAt != 0 {
			return appError(pre+"The post supposedly holding the profile picture is deleted.", nil)
		}
		if len(post.FileIds) != 1 {
			return appError(pre+"The post supposedly holding the profile picture does not have exactly 1 file.", nil)
		}
		if post.FileIds[0] != profile.PictureFileId {
			return appError(pre+"The post supposedly holding the profile picture does not hold the expected file.", nil)
		}
	}
	return nil
}

// EncodeToByte returns a profile as a byte array
func (p *Plugin) EncodeToByte(profile *Profile) []byte {
	b, _ := json.Marshal(profile)
	return b
}

// DecodeProfileFromByte tries to create a Profile from a byte array
func (p *Plugin) DecodeProfileFromByte(b []byte) *Profile {
	profile := Profile{}
	err := json.Unmarshal(b, &profile)
	if err != nil {
		return nil
	}
	return &profile
}

func getProfileKey(userId, profileId string) string {
	return fmt.Sprintf("profile_%s_%s", userId, profileId)
}

func (p *Plugin) profileExists(userId, profileId string) (bool, *model.AppError) {
	b, err := p.API.KVGet(getProfileKey(userId, profileId))
	if err != nil {
		return false, err
	}
	return b != nil, nil
}

func (p *Plugin) getProfile(userId, profileId string, validate bool) (*Profile, *model.AppError) {
	b, err := p.API.KVGet(getProfileKey(userId, profileId))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, appError(fmt.Sprintf("Profile `%s` does not exist.", profileId), nil)
	}
	profile := p.DecodeProfileFromByte(b)
	if profile == nil {
		return nil, appError(fmt.Sprintf("Profile `%s` failed to decode and needs to be recreated.", profileId), nil)
	}
	if validate {
		err = p.validateProfile(profile, profileId)
		if err != nil {
			return nil, appErrorPre(fmt.Sprintf("Profile `%s` is corrupt and needs to be recreated: ", profileId), err)
		}
	}
	return profile, nil
}

func (p *Plugin) setProfile(userId string, profile *Profile) *model.AppError {
	err := p.validateProfile(profile, profile.Identifier)
	if err != nil {
		return err
	}
	err = p.API.KVSet(getProfileKey(userId, profile.Identifier), p.EncodeToByte(profile))
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) deleteProfile(userId, profileId string) *model.AppError {
	return p.API.KVDelete(getProfileKey(userId, profileId))
}

func (p *Plugin) listProfiles(userId string) ([]Profile, *model.AppError) {
	keys, err := p.getKeysAfterPrefix(getProfileKey(userId, ""))
	if err != nil {
		return nil, err
	}
	ret := make([]Profile, 0)
	for _, key := range keys {
		profile, err := p.getProfile(userId, key, true)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *profile)
	}
	return ret, nil
}

func (p *Plugin) getKeysAfterPrefix(prefix string) ([]string, *model.AppError) {
	var ret []string
	i := 0
	for {
		keys, appErr := p.API.KVList(i, perPage)
		if appErr != nil {
			return nil, appErr
		}
		for _, key := range keys {
			if strings.HasPrefix(key, prefix) {
				ret = append(ret, strings.TrimPrefix(key, prefix))
			}
		}
		if len(keys) < perPage {
			break
		}
		i++
	}
	return ret, nil
}

func getDefaultProfileKey(userId, channelId string) string {
	return fmt.Sprintf("defaultprofile_%s_%s", userId, channelId)
}

func (p *Plugin) removeDefaultProfile(userId, channelId string) *model.AppError {
	return p.API.KVDelete(getDefaultProfileKey(userId, channelId))
}

func (p *Plugin) getDefaultProfileIdentifier(userId, channelId string) (string, *model.AppError) {
	b, err := p.API.KVGet(getDefaultProfileKey(userId, channelId))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *Plugin) setDefaultProfileIdentifier(userId, channelId, profileId string) (string, *model.AppError) {
	profile, err := p.getProfile(userId, profileId, true)
	if err != nil {
		return "", err
	}
	err = p.API.KVSet(getDefaultProfileKey(userId, channelId), []byte(profileId))
	if err != nil {
		return "", appError("", err)
	}
	return profile.Name, nil
}

func appErrorPre(prefix string, err *model.AppError) *model.AppError {
	if err == nil {
		return nil
	}
	errCopy := *err
	errCopy.Message = fmt.Sprintf("%s: %s", prefix, err.Message)
	return &errCopy
}

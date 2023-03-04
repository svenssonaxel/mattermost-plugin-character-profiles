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
	Identifier string
	Name       string
	IconUrl    string
}

func validateProfile(profile *Profile, profileId string) *model.AppError {
	if profile == nil {
		return appError("Profile is nil.", nil)
	}
	if profile.Identifier != profileId {
		return appError("Identifier mismatch.", nil)
	}
	matches := regexp.MustCompile(`^[a-z]{1,60}$`).FindStringSubmatch(profile.Identifier)
	if len(matches) != 1 {
		return appError("Identifier must be 1-60 lowercase letters a-z.", nil)
	}
	matches = regexp.MustCompile("^[^-*|`>#*_.~[\\]]{1,200}$").FindStringSubmatch(profile.Name)
	if len(matches) != 1 {
		return appError("Display name must be 1-200 characters and must not contain format control characters.", nil)
	}
	return nil
}

// EncodeToByte returns a profile as a byte array
func (p *Plugin) EncodeToByte(profile Profile) []byte {
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

func (p *Plugin) getProfile(userId, profileId string) (*Profile, *model.AppError) {
	b, err := p.API.KVGet(getProfileKey(userId, profileId))
	if err != nil {
		return nil, err
	}
	profile := p.DecodeProfileFromByte(b)
	if profile == nil {
		return nil, appError("Failed to decode profile", nil)
	}
	err = validateProfile(profile, profileId)
	if err != nil {
		return nil, appError(fmt.Sprintf("Profile `%s` is corrupt and needs to be recreated.", profileId), nil)
	}
	return profile, nil
}

func (p *Plugin) setProfile(userId string, profile Profile) *model.AppError {
	err := validateProfile(&profile, profile.Identifier)
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
		profile, err := p.getProfile(userId, key)
		if err != nil {
			return nil, err
		}
		err = validateProfile(profile, key)
		if err != nil {
			return nil, appError(fmt.Sprintf("Profile `%s` is corrupt and needs to be recreated.", key), nil)
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
	profile, err := p.getProfile(userId, profileId)
	if err != nil {
		return "", err
	}
	err = p.API.KVSet(getDefaultProfileKey(userId, channelId), []byte(profileId))
	if err != nil {
		return "", appError("", err)
	}
	return profile.Name, nil
}

package main

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

// BackendMock is a mock of the Backend interface for testing purposes.

type BackendMock struct {
	ChannelMembers []struct {
		UserId    string
		ChannelId string
	}
	Channels  map[string]*model.Channel
	FileInfos map[string]*model.FileInfo
	KVStore   map[string][]byte
	Posts     map[string]*model.Post
	SiteURL   string
	Teams     map[string]*model.Team
	Users     map[string]*model.User
}

func (b BackendMock) GetChannelMembers(channelId string, page int, perPage int) (*model.ChannelMembers, *model.AppError) {
	ret := make(model.ChannelMembers, 0)
	for _, member := range b.ChannelMembers {
		if member.ChannelId == channelId {
			ret = append(ret, model.ChannelMember{
				ChannelId: channelId,
				UserId:    member.UserId})
		}
	}
	min := page * perPage
	if min > len(ret) {
		min = len(ret)
	}
	max := (page + 1) * perPage
	if max > len(ret) {
		max = len(ret)
	}
	ret = ret[min:max]
	return &ret, nil
}
func (b BackendMock) GetChannelsForTeamForUser(teamId string, userId string, includeDeleted bool) ([]*model.Channel, *model.AppError) {
	ret := []*model.Channel{}
	for _, channel := range b.Channels {
		if channel.TeamId == teamId && (channel.DeleteAt == 0 || includeDeleted) {
			perPage := 100
			page := 0
			for {
				members, err := b.GetChannelMembers(channel.Id, page, perPage)
				if err != nil {
					return nil, err
				}
				if members == nil {
					return nil, model.NewAppError("BackendMock", "channel_members_not_found", nil, "", 0)
				}
				if len(*members) == 0 {
					break
				}
				for _, member := range *members {
					if member.UserId == userId {
						ret = append(ret, channel)
						break
					}
				}
				page++
			}
		}
	}
	return ret, nil
}
func (b BackendMock) GetFileInfo(id string) (*model.FileInfo, *model.AppError) {
	fileInfo, ok := b.FileInfos[id]
	if !ok {
		return nil, model.NewAppError("BackendMock", "file_info_not_found", nil, "", 0)
	}
	return fileInfo, nil
}
func (b BackendMock) GetPost(id string) (*model.Post, *model.AppError) {
	post, ok := b.Posts[id]
	if !ok {
		return nil, model.NewAppError("BackendMock", "post_not_found", nil, "", 0)
	}
	return post, nil
}
func (b BackendMock) GetSiteURL() string {
	return b.SiteURL
}
func (b BackendMock) GetTeam(id string) (*model.Team, *model.AppError) {
	team, ok := b.Teams[id]
	if !ok {
		return nil, model.NewAppError("BackendMock", "team_not_found", nil, "", 0)
	}
	return team, nil
}
func (b BackendMock) GetUser(id string) (*model.User, *model.AppError) {
	user, ok := b.Users[id]
	if !ok {
		return nil, model.NewAppError("BackendMock", "user_not_found", nil, "", 0)
	}
	return user, nil
}
func (b BackendMock) KVCompareAndSet(key string, oldValue, newValue []byte) (bool, *model.AppError) {
	actualOldValue, ok := b.KVStore[key]
	if ok {
		if oldValue == nil {
			return false, nil
		}
		// Check if actualOldValue is equal to oldValue
		if len(actualOldValue) != len(oldValue) {
			return false, nil
		}
		for i := range actualOldValue {
			if actualOldValue[i] != oldValue[i] {
				return false, nil
			}
		}
		// If so, set the new value
		b.KVStore[key] = newValue
		return true, nil
	} else {
		// If not, set the new value if oldValue is nil
		if oldValue == nil {
			b.KVStore[key] = newValue
			return true, nil
		}
		return false, nil
	}
}
func (b BackendMock) KVDelete(key string) *model.AppError {
	delete(b.KVStore, key)
	return nil
}
func (b BackendMock) KVGet(key string) ([]byte, *model.AppError) {
	return b.KVStore[key], nil
}
func (b BackendMock) KVSet(key string, value []byte) *model.AppError {
	b.KVStore[key] = value
	return nil
}

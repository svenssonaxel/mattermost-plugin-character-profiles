package main

import (
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

// BackendImpl is a thin wrapper around a subset of plugin.API, allowing the
// Backend interface to be implemented by a mock.

type Backend interface {
	GetBundlePath() string
	GetChannelMembers(channelId string, page int, perPage int) (*model.ChannelMembers, *model.AppError)
	GetChannelsForTeamForUser(teamId string, userId string, includeDeleted bool) ([]*model.Channel, *model.AppError)
	GetFileInfo(id string) (*model.FileInfo, *model.AppError)
	GetPost(id string) (*model.Post, *model.AppError)
	GetSiteURL() string
	GetTeam(id string) (*model.Team, *model.AppError)
	GetUser(id string) (*model.User, *model.AppError)
	KVCompareAndSet(key string, oldValue, newValue []byte) (bool, *model.AppError)
	KVDelete(key string) *model.AppError
	KVGet(key string) ([]byte, *model.AppError)
	KVSet(key string, value []byte) *model.AppError
	NewId() string
	ReadFile(path string) ([]byte, *model.AppError)
	UpdatePost(post *model.Post) (*model.Post, *model.AppError)
}

type BackendImpl struct {
	API        plugin.API
	BundlePath string
	SiteURL    string
}

func (b BackendImpl) GetBundlePath() string {
	return b.BundlePath
}
func (b BackendImpl) GetChannelMembers(channelId string, page int, perPage int) (*model.ChannelMembers, *model.AppError) {
	return b.API.GetChannelMembers(channelId, page, perPage)
}
func (b BackendImpl) GetChannelsForTeamForUser(teamId string, userId string, includeDeleted bool) ([]*model.Channel, *model.AppError) {
	return b.API.GetChannelsForTeamForUser(teamId, userId, includeDeleted)
}
func (b BackendImpl) GetFileInfo(id string) (*model.FileInfo, *model.AppError) {
	return b.API.GetFileInfo(id)
}
func (b BackendImpl) GetPost(id string) (*model.Post, *model.AppError) {
	return b.API.GetPost(id)
}
func (b BackendImpl) GetSiteURL() string {
	return b.SiteURL
}
func (b BackendImpl) GetTeam(id string) (*model.Team, *model.AppError) {
	return b.API.GetTeam(id)
}
func (b BackendImpl) GetUser(id string) (*model.User, *model.AppError) {
	return b.API.GetUser(id)
}
func (b BackendImpl) KVCompareAndSet(key string, oldValue, newValue []byte) (bool, *model.AppError) {
	return b.API.KVCompareAndSet(key, oldValue, newValue)
}
func (b BackendImpl) KVDelete(key string) *model.AppError {
	return b.API.KVDelete(key)
}
func (b BackendImpl) KVGet(key string) ([]byte, *model.AppError) {
	return b.API.KVGet(key)
}
func (b BackendImpl) KVSet(key string, value []byte) *model.AppError {
	return b.API.KVSet(key, value)
}
func (b BackendImpl) NewId() string {
	return model.NewId()
}
func (b BackendImpl) ReadFile(path string) ([]byte, *model.AppError) {
	return b.API.ReadFile(path)
}
func (b BackendImpl) UpdatePost(post *model.Post) (*model.Post, *model.AppError) {
	return b.API.UpdatePost(post)
}

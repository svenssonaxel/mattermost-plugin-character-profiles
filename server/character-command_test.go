package main_test

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/svenssonaxel/mattermost-plugin-character-profiles/server"
)

func TestScenario1(t *testing.T) {
	var (
		siteURL      = "http://mocksite.tld"
		channel1     = "channel1_________________"
		channel2     = "channel2_________________"
		characterPng = siteURL + "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/character.png"
		file1        = "file1_____________________"
		file1Png     = siteURL + "/api/v4/files/" + file1
		file2        = "file2_____________________"
		file2Png     = siteURL + "/api/v4/files/" + file2
		nosign       = siteURL + "/plugins/com.axelsvensson.mattermost-plugin-character-profiles/no-sign.jpg"
		post1        = "post1_____________________"
		post2        = "post2_____________________"
		team1        = "team1_____________________"
		user1        = "user1_____________________"
		user1image   = siteURL + "/api/v4/users/" + user1 + "/image"
		user2        = "user2_____________________"
	)
	// Initialize the backend mock
	be := main.BackendMock{
		ChannelMembers: []struct {
			UserId    string
			ChannelId string
		}{
			{user1, channel1},
			{user1, channel2},
			{user2, channel1},
		},
		Channels: map[string]*model.Channel{
			channel1: {Id: channel1, Name: "channel-one", DisplayName: "Channel One", TeamId: team1, Type: model.CHANNEL_OPEN},
			channel2: {Id: channel2, Name: "channel-two", DisplayName: "Channel Two", TeamId: team1, Type: model.CHANNEL_OPEN},
		},
		FileInfos: map[string]*model.FileInfo{
			file1: {Id: file1, CreatorId: user1, CreateAt: 1, UpdateAt: 1, Path: "some-path-to/file1.png", Name: "file1.png", Extension: "png", MimeType: "image/png", PostId: post1},
			file2: {Id: file2, CreatorId: user1, CreateAt: 2, UpdateAt: 2, Path: "some-path-to/file2.png", Name: "file2.png", Extension: "png", MimeType: "image/png", PostId: post2},
		},
		KVStore: map[string][]byte{},
		Posts: map[string]*model.Post{
			post1: {Id: post1, UserId: user1, ChannelId: channel1, FileIds: []string{file1}},
			post2: {Id: post2, UserId: user1, ChannelId: channel1, FileIds: []string{file2}},
		},
		SiteURL: siteURL,
		Teams: map[string]*model.Team{
			team1: {Id: team1, Name: team1},
		},
		Users: map[string]*model.User{
			user1: {Id: user1, Username: "user-number-one"},
			user2: {Id: user2, Username: "user-number-two"},
		},
	}
	red := "#ff0000"
	green := "#009900"
	blue := "#5c66ff"
	// In the beginning, the profile list contains only the default profile
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{{"**user-number-one** *(your real profile)*\n`me`, `myself`",
			green, user1image},
		})
	// Create a new profile, then delete it
	cmd(be, "/character someone=Someone", user1, channel1, team1, "", t,
		"Character profile `someone` created with display name \"Someone\"",
		[]tAtt{{"**Someone**\n`someone`",
			blue, characterPng},
		})
	cmd(be, "/character delete someone", user1, channel1, team1, "", t,
		"Deleted character profile `someone`.",
		[]tAtt{})
	// Create a new profile, then set its profile picture in a separate command
	cmd(be, "/character haddock=Captain Haddock", user1, channel1, team1, "", t,
		"Character profile `haddock` created with display name \"Captain Haddock\"",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, characterPng},
		})
	cmd(be, "/character picture haddock", user1, channel1, team1, post1, t,
		"Character profile `haddock` modified by updating the profile picture",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, file1Png},
		})
	// Create a new profile and set its profile picture in the same command
	cmd(be, "/character picture milou=Milou", user1, channel1, team1, post2, t,
		"Character profile `milou` created with display name \"Milou\" and a profile picture",
		[]tAtt{{"**Milou**\n`milou`",
			blue, file2Png},
		})
	// List the profiles
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`",
				blue, file1Png},
			{"**Milou**\n`milou`",
				blue, file2Png},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`",
				green, user1image},
		})
	// Set default profiles for two channels
	cmd(be, "/character I am haddock", user1, channel1, team1, "", t,
		"You are now known as \"Captain Haddock\".",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, file1Png},
		})
	cmd(be, "/character I am milou", user1, channel2, team1, "", t,
		"You are now known as \"Milou\".",
		[]tAtt{{"**Milou**\n`milou`",
			blue, file2Png},
		})
	// Delete the post holding the profile picture of the second profile
	be.Posts[post2].DeleteAt = 1
	// List profiles for user1
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`",
				blue, file1Png},
			{"**Milou** *(corrupt profile)*\n`milou`\nError: Character Profile Plugin: Profile `milou` is corrupt and needs to be recreated: Failed validating profile `milou`: The post supposedly holding the profile picture is deleted., ",
				red, nosign},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`",
				green, user1image},
		})
	// Add a post by user1 to channel1 and check that the first profile is used
	post3 := "post3_____________________"
	post(be, t, &model.Post{Id: post3, UserId: user1, ChannelId: channel1, Message: "Hello from Haddock"},
		"haddock", "Captain Haddock", file1Png)
	// Add a post by user1 to channel2 and check that the default profile is used (because the second profile is corrupt)
	post4 := "post4_____________________"
	post(be, t, &model.Post{Id: post4, UserId: user1, ChannelId: channel2, Message: "No hello from Milou"},
		"", "", "")
	// Add a one-off post by user1 to channel2 and check that the first profile is used
	post5 := "post5_____________________"
	post(be, t, &model.Post{Id: post5, UserId: user1, ChannelId: channel2, Message: "haddock: Hello from Haddock"},
		"haddock", "Captain Haddock", file1Png)
	// Add a one-off post by user1 to channel1 and check that the default profile is used
	post6 := "post6_____________________"
	post(be, t, &model.Post{Id: post6, UserId: user1, ChannelId: channel1, Message: "me: Hello from user-number-one"},
		"", "", "")
	// List default profiles for user1
	cmd(be, "/character who am I", user1, channel1, team1, "", t,
		"## Default character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`\nDefault profile in: ~channel-one",
				blue, file1Png},
			{"**Milou** *(corrupt profile)*\n`milou`\nError: Character Profile Plugin: Profile `milou` is corrupt and needs to be recreated: Failed validating profile `milou`: The post supposedly holding the profile picture is deleted., \nDefault profile in: ~channel-two",
				red, nosign},
		})
	// Delete the second profile
	cmd(be, "/character delete milou", user1, channel1, team1, "", t,
		"Deleted character profile `milou`.",
		[]tAtt{})
	// List default profiles for user1
	cmd(be, "/character who am I", user1, channel1, team1, "", t,
		"## Default character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`\nDefault profile in: ~channel-one",
				blue, file1Png},
			{"*(profile does not exist)*\n`milou`\nDefault profile in: ~channel-two",
				red, nosign},
		})
	// List profiles for user1
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`",
				blue, file1Png},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`",
				green, user1image},
		})
	// Remove the default profile for channel2
	cmd(be, "/character I am myself", user1, channel2, team1, "", t,
		"You are now yourself again. Hope that feels ok.",
		[]tAtt{{"**user-number-one** *(your real profile)*\n`me`, `myself`",
			green, user1image},
		})
	// List default profiles for user1
	cmd(be, "/character who am I", user1, channel1, team1, "", t,
		"## Default character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`\nDefault profile in: ~channel-one",
				blue, file1Png},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`\nDefault profile in: ~channel-two",
				green, user1image},
		})
}

type tAtt struct {
	Text     string
	Color    string
	ThumbURL string
}

func cmd(be main.Backend, command, userId, channelId, teamId, rootId string,
	t *testing.T, expectedResponse string, expectedAttachments []tAtt) {
	msg := fmt.Sprintf("Command: %s", command)
	response, attachments, err := main.DoExecuteCommand(be, command, userId, channelId, teamId, rootId)
	assert.Nil(t, err, msg)
	assert.Equal(t, expectedResponse, response, msg)
	assert.Equal(t, len(expectedAttachments), len(attachments), msg)
	for i, expectedAttachment := range expectedAttachments {
		assert.Equal(t, expectedAttachment.Text, attachments[i].Text, msg)
		assert.Equal(t, expectedAttachment.Color, attachments[i].Color, msg)
		assert.Equal(t, expectedAttachment.ThumbURL, attachments[i].ThumbURL, msg)
	}
}

func post(be main.BackendMock, t *testing.T, inputPost *model.Post, expectedProfile, expectedDisplayName, expectedThumbURL string) {
	msg := fmt.Sprintf("CreatePost: %s", inputPost.Id)
	post, errStr := main.ProfiledPost(be, inputPost, false)
	assert.Equal(t, "", errStr, msg)
	if post == nil {
		post = inputPost
	}
	_, postAlreadyExists := be.Posts[post.Id]
	assert.False(t, postAlreadyExists, msg)
	be.Posts[post.Id] = post
	if expectedProfile == "" {
		assert.Nil(t, post.Props["profile_identifier"], msg)
		assert.Nil(t, post.Props["override_username"], msg)
		assert.Nil(t, post.Props["override_icon_url"], msg)
	} else {
		assert.Equal(t, expectedProfile, post.Props["profile_identifier"], msg)
		assert.Equal(t, expectedDisplayName, post.Props["override_username"], msg)
		assert.Equal(t, expectedThumbURL, post.Props["override_icon_url"], msg)
		assert.Equal(t, "true", post.Props["from_webhook"], msg)
	}
	be.Posts[post.Id] = post
}

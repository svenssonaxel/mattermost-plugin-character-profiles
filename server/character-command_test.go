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
		siteURL  = "http://mocksite.tld"
		channel1 = "channel1_________________"
		channel2 = "channel2_________________"
		file1    = "file1_____________________"
		file2    = "file2_____________________"
		post1    = "post1_____________________"
		post2    = "post2_____________________"
		team1    = "team1_____________________"
		user1    = "user1_____________________"
		user2    = "user2_____________________"
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
		IdCounter: new(int),
		KVStore:   map[string][]byte{},
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
	pluginURL := main.GetPluginURL(be)
	var (
		characterImg = func(thumb bool) string {
			if thumb {
				return pluginURL + "/static/defaultprofilepicture/thumbnail"
			} else {
				return pluginURL + "/static/defaultprofilepicture"
			}
		}
		nosign = func(thumb bool) string {
			if thumb {
				return pluginURL + "/static/corruptedprofilepicture/thumbnail"
			} else {
				return pluginURL + "/static/corruptedprofilepicture"
			}
		}
		user1image     = func(_ bool) string { return siteURL + "/api/v4/users/" + user1 + "/image" }
		characterImage = func(be main.Backend, userId string, profileIdentifier string) func(thumb bool) string {
			return func(thumb bool) string {
				profile, err := main.GetProfile(be, userId, profileIdentifier, main.PROFILE_CHARACTER)
				if err != nil {
					return "ERROR: " + err.Error()
				}
				if profile == nil {
					return "ERROR: profile not found"
				}
				rk := profile.RequestKey
				t := ""
				if thumb {
					t = "/thumbnail"
				}
				return fmt.Sprintf("%s/plugins/com.axelsvensson.mattermost-plugin-character-profiles/profile/%s/%s%s?rk=%s", be.GetSiteURL(), userId, profileIdentifier, t, rk)
			}
		}
		user1haddockImg = characterImage(be, user1, "haddock")
		user1milouImg   = characterImage(be, user1, "milou")
	)
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
			blue, characterImg},
		})
	cmd(be, "/character delete someone", user1, channel1, team1, "", t,
		"Deleted character profile `someone`.",
		[]tAtt{})
	// Create a new profile, then set its profile picture in a separate command
	cmd(be, "/character haddock=Captain Haddock", user1, channel1, team1, "", t,
		"Character profile `haddock` created with display name \"Captain Haddock\"",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, characterImg},
		})
	cmd(be, "/character picture haddock", user1, channel1, team1, post1, t,
		"Character profile `haddock` modified by updating the profile picture",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, user1haddockImg},
		})
	// Create a new profile and set its profile picture in the same command
	cmd(be, "/character picture milou=Milou", user1, channel1, team1, post2, t,
		"Character profile `milou` created with display name \"Milou\" and a profile picture",
		[]tAtt{{"**Milou**\n`milou`",
			blue, user1milouImg},
		})
	// List the profiles
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`",
				blue, user1haddockImg},
			{"**Milou**\n`milou`",
				blue, user1milouImg},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`",
				green, user1image},
		})
	// Set default profiles for two channels
	cmd(be, "/character I am haddock", user1, channel1, team1, "", t,
		"You are now known as \"Captain Haddock\".",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, user1haddockImg},
		})
	cmd(be, "/character I am milou", user1, channel2, team1, "", t,
		"You are now known as \"Milou\".",
		[]tAtt{{"**Milou**\n`milou`",
			blue, user1milouImg},
		})
	// Change the profile picture of the first profile
	cmd(be, "/character picture haddock", user1, channel1, team1, post2, t,
		"Character profile `haddock` modified by updating the profile picture",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, user1haddockImg},
		})
	// Delete the post holding the profile picture of the second profile
	be.Posts[post2].DeleteAt = 1
	// Although both profiles are now corrupt, the first one can be corrected by setting a new profile picture
	cmd(be, "/character picture haddock", user1, channel1, team1, post1, t,
		"Character profile `haddock` modified by updating the profile picture",
		[]tAtt{{"**Captain Haddock**\n`haddock`",
			blue, user1haddockImg},
		})
	// List profiles for user1
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`",
				blue, user1haddockImg},
			{"**Milou** *(corrupt profile)*\n`milou`\nError: Character Profile Plugin: Profile `milou` is corrupt and needs to be recreated: Failed to populate profile `milou`: The post supposedly holding the profile picture could not be found, perhaps it's deleted.",
				red, nosign},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`",
				green, user1image},
		})
	// Add a post by user1 to channel1 and check that the first profile is used
	post3 := post(be, t, &model.Post{UserId: user1, ChannelId: channel1, Message: "Hello from Haddock"},
		"haddock", "Captain Haddock", user1haddockImg)
	// Add a post by user1 to channel2 and check that the default profile is used (because the second profile is corrupt)
	post(be, t, &model.Post{UserId: user1, ChannelId: channel2, Message: "No hello from Milou"},
		"", "", nil)
	// Add a one-off post by user1 to channel2 and check that the first profile is used
	post5 := post(be, t, &model.Post{UserId: user1, ChannelId: channel2, Message: "haddock: Hello from Haddock"},
		"haddock", "Captain Haddock", user1haddockImg)
	// Add a one-off post by user1 to channel1 and check that the default profile is used
	post6 := post(be, t, &model.Post{UserId: user1, ChannelId: channel1, Message: "me: Hello from user-number-one"},
		"", "", nil)
	// List default profiles for user1
	cmd(be, "/character who am I", user1, channel1, team1, "", t,
		"## Default character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`\nDefault profile in: ~channel-one",
				blue, user1haddockImg},
			{"**Milou** *(corrupt profile)*\n`milou`\nError: Character Profile Plugin: Profile `milou` is corrupt and needs to be recreated: Failed to populate profile `milou`: The post supposedly holding the profile picture could not be found, perhaps it's deleted.\nDefault profile in: ~channel-two",
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
				blue, user1haddockImg},
			{"*(profile does not exist)*\n`milou`\nDefault profile in: ~channel-two",
				red, nosign},
		})
	// List profiles for user1
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{
			{"**Captain Haddock**\n`haddock`",
				blue, user1haddockImg},
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
				blue, user1haddockImg},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`\nDefault profile in: ~channel-two",
				green, user1image},
		})
	// Edit a post made by user1 using the first profile, to instead use the default profile
	editPost(be, t, post5, "me: Actually, hello from me instead", "", "", nil)
	// Edit a post made by user1 using the default profile, to instead use the first profile
	editPost(be, t, post6, "haddock: Hello from Haddock, again", "haddock", "Captain Haddock", user1haddockImg)
	// Change the display name of the first profile
	cmd(be, "/character haddock=Mr Haddock Sr", user1, channel1, team1, "", t,
		"Character profile `haddock` modified by changing the display name from \"Captain Haddock\" to \"Mr Haddock Sr\"",
		[]tAtt{{"**Mr Haddock Sr**\n`haddock`",
			blue, user1haddockImg},
		})
	// List profiles for user1
	cmd(be, "/character list", user1, channel1, team1, "", t,
		"## Character profiles",
		[]tAtt{
			{"**Mr Haddock Sr**\n`haddock`",
				blue, user1haddockImg},
			{"**user-number-one** *(your real profile)*\n`me`, `myself`",
				green, user1image},
		})
	// Check that posts previously made using the first profile has the new display name
	for _, postId := range []string{post3, post6} {
		msg := fmt.Sprintf("Checking username of post %s", postId)
		post, err := be.GetPost(postId)
		assert.Nil(t, err, msg)
		overrideUsername, ok := post.Props["override_username"]
		assert.True(t, ok, msg)
		assert.Equal(t, "Mr Haddock Sr", overrideUsername, msg)
	}
	// Check that a post first made using the first profile, then edited to use the default profile, is unaffected by the profile change
	msg := fmt.Sprintf("Checking username of post %s", post5)
	post, err := be.GetPost(post5)
	assert.Nil(t, err, msg)
	overrideUsername, ok := post.Props["override_username"]
	assert.True(t, ok, msg)
	assert.Equal(t, nil, overrideUsername, msg)
}

type tAtt struct {
	Text      string
	Color     string
	GetImgURL func(thumb bool) string
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
		assert.Equal(t, expectedAttachment.GetImgURL(true), attachments[i].ThumbURL, msg)
	}
}

func post(be main.BackendMock, t *testing.T, inputPost *model.Post, expectedProfile, expectedDisplayName string, getExpectedImgURL func(thumb bool) string) string {
	msg := fmt.Sprintf("CreatePost: %s", inputPost.Id)
	post, errStr := main.ProfiledPost(be, inputPost, false)
	assert.Equal(t, "", errStr, msg)
	if post == nil {
		post = inputPost
	}
	_, postAlreadyExists := be.Posts[post.Id]
	assert.False(t, postAlreadyExists, msg)
	if expectedProfile == "" {
		assert.Nil(t, post.Props["profile_identifier"], msg)
		assert.Nil(t, post.Props["override_username"], msg)
		assert.Nil(t, post.Props["override_icon_url"], msg)
		assert.Nil(t, post.Props["from_webhook"], msg)
	} else {
		assert.Equal(t, expectedProfile, post.Props["profile_identifier"], msg)
		assert.Equal(t, expectedDisplayName, post.Props["override_username"], msg)
		assert.Equal(t, getExpectedImgURL(false), post.Props["override_icon_url"], msg)
		assert.Equal(t, "true", post.Props["from_webhook"], msg)
	}
	post.Id = be.NewId()
	be.Posts[post.Id] = post
	main.RegisterPost(be, post)
	return post.Id
}

func editPost(be main.BackendMock, t *testing.T, postId string, newMessage string, expectedProfile, expectedDisplayName string, getExpectedImgURL func(thumb bool) string) {
	msg := fmt.Sprintf("EditPost: %s", postId)
	post, pErr := be.GetPost(postId)
	assert.Nil(t, pErr, msg)
	assert.NotNil(t, post, msg)
	post = main.DeepClonePost(post)
	post.Message = newMessage
	post, ppErrStr := main.ProfiledPost(be, post, true)
	assert.Equal(t, "", ppErrStr, msg)
	if expectedProfile == "" {
		assert.Nil(t, post.Props["profile_identifier"], msg)
		assert.Nil(t, post.Props["override_username"], msg)
		assert.Nil(t, post.Props["override_icon_url"], msg)
		assert.Nil(t, post.Props["from_webhook"], msg)
	} else {
		assert.Equal(t, expectedProfile, post.Props["profile_identifier"], msg)
		assert.Equal(t, expectedDisplayName, post.Props["override_username"], msg)
		assert.Equal(t, getExpectedImgURL(false), post.Props["override_icon_url"], msg)
		assert.Equal(t, "true", post.Props["from_webhook"], msg)
	}
	be.Posts[post.Id] = post
	main.RegisterPost(be, post)
}

package main_test

import (
	"bytes"
	_ "embed"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/svenssonaxel/mattermost-plugin-character-profiles/server"
)

func TestScenario1(t *testing.T) {
	var (
		siteURL  = "http://mocksite.tld"
		channel1 = "channel1aaaaaaaaaaaaaaaaaa"
		channel2 = "channel2aaaaaaaaaaaaaaaaaa"
		file1    = "file1aaaaaaaaaaaaaaaaaaaaa"
		file2    = "file2aaaaaaaaaaaaaaaaaaaaa"
		post1    = "post1aaaaaaaaaaaaaaaaaaaaa"
		post2    = "post2aaaaaaaaaaaaaaaaaaaaa"
		team1    = "team1aaaaaaaaaaaaaaaaaaaaa"
		user1    = "user1aaaaaaaaaaaaaaaaaaaaa"
		user2    = "user2aaaaaaaaaaaaaaaaaaaaa"
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
				return fmt.Sprintf("%s/plugins/%s/profile/%s/%s%s?rk=%s", be.GetSiteURL(), main.PLUGIN_ID, userId, profileIdentifier, t, rk)
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
		postObj, err := be.GetPost(postId)
		assert.Nil(t, err, msg)
		overrideUsername, ok := postObj.Props["override_username"]
		assert.True(t, ok, msg)
		assert.Equal(t, "Mr Haddock Sr", overrideUsername, msg)
	}
	// Check that a post first made using the first profile, then edited to use the default profile, is unaffected by the profile change
	msg := fmt.Sprintf("Checking username of post %s", post5)
	postObj, err := be.GetPost(post5)
	assert.Nil(t, err, msg)
	overrideUsername, ok := postObj.Props["override_username"]
	assert.True(t, ok, msg)
	assert.Equal(t, nil, overrideUsername, msg)
	// Test /character make
	type TestCase struct {
		expectSuccess bool
		msg           string
		resProfile    int
		p1PostCount   int
		p1Status      int
		p2PostCount   int
		p2Status      int
	}
	const (
		OLD  = 0
		NEW  = 1
		NONE = 2
	)
	for testIdx, testcase := range []TestCase{
		{true, "All messages that used character profile `{{.f}}` now use character profile `{{.t}}` instead. Character profile `{{.f}}` has been deleted.", NEW,
			2, main.PROFILE_CHARACTER,
			2, main.PROFILE_CHARACTER},
		{false, "Character Profile Plugin: Character profile `{{.f}}` isn't used by any messages. You can delete it with `/character delete {{.f}}`.", NONE,
			0, main.PROFILE_CHARACTER,
			2, main.PROFILE_CHARACTER},
		{true, "All messages that used character profile `{{.f}}` now use character profile `{{.t}}` instead. Character profile `{{.f}}` has been deleted.", NEW,
			2, main.PROFILE_CHARACTER,
			0, main.PROFILE_CHARACTER},
		{false, "Character Profile Plugin: Character profile `{{.f}}` isn't used by any messages. You can delete it with `/character delete {{.f}}`.", NONE,
			0, main.PROFILE_CHARACTER,
			0, main.PROFILE_CHARACTER},
		{true, "All messages that used character profile `{{.f}}` now use your real profile instead. Character profile `{{.f}}` has been deleted.", NEW,
			2, main.PROFILE_CHARACTER,
			0, main.PROFILE_ME},
		{false, "Character Profile Plugin: Character profile `{{.f}}` isn't used by any messages. You can delete it with `/character delete {{.f}}`.", NONE,
			0, main.PROFILE_CHARACTER,
			2, main.PROFILE_ME},
		{false, "Character Profile Plugin: Target character profile `{{.t}}` is corrupt.", NONE,
			2, main.PROFILE_CHARACTER,
			2, main.PROFILE_CORRUPT},
		{false, "Character Profile Plugin: Target character profile `{{.t}}` doesn't exist, but since it is still used by {{.c}} messages you must recreate it before you can make another character profile into it.", NONE,
			2, main.PROFILE_CHARACTER,
			2, main.PROFILE_NONEXISTENT},
		{true, "Changed identifier for character profile `{{.f}}` to `{{.t}}`.", OLD,
			2, main.PROFILE_CHARACTER,
			0, main.PROFILE_NONEXISTENT},
		{false, "Character Profile Plugin: Cannot make your real profile into something else. Use the Mattermost built-in functionality to change the display name or profile picture for your real Mattermost profile.", NONE,
			2, main.PROFILE_ME,
			2, main.PROFILE_CHARACTER},
		{false, "Character Profile Plugin: Character profile `{{.f}}` is corrupt, but is still used by {{.c}} messages. Before you try to make this character profile into something else, you need to delete and recreate it. The messages will not be affected by deleting the profile.", NONE,
			2, main.PROFILE_CORRUPT,
			2, main.PROFILE_CHARACTER},
		{false, "Character Profile Plugin: Character profile `{{.f}}` is corrupt, and isn't used by any messages. You can delete it with `/character delete {{.f}}`.", NONE,
			0, main.PROFILE_CORRUPT,
			2, main.PROFILE_CHARACTER},
		{false, "Character Profile Plugin: Character profile `{{.f}}` doesn't exist, but is still used by {{.c}} messages. Create a character profile with this identifier in order to manage those messages.", NONE,
			2, main.PROFILE_NONEXISTENT,
			2, main.PROFILE_CHARACTER},
		{false, "Character Profile Plugin: Character profile `{{.f}}` doesn't exist, and isn't used by any messages.", NONE,
			0, main.PROFILE_NONEXISTENT,
			2, main.PROFILE_CHARACTER},
	} {
		msg := fmt.Sprintf("Test %d: %s", testIdx, testcase.msg)
		newPId := func(b main.BackendMock) string {
			*b.IdCounter++
			r := rand.NewSource(int64(*b.IdCounter))
			chars := "abcdefghijklmnopqrstuvwxyz"
			ret := ""
			for i := 0; i < 10; i++ {
				ret += string(chars[r.Int63()%26])
			}
			return ret
		}
		channel := channel2 // Use channel2 for all tests, since it is using the real profile
		setup := func(postCount, status int) (string, string, []string) {
			var pId, pName, message string
			var pPostIds []string
			if status == main.PROFILE_ME {
				pId = "me"
				message = "Test message from user-number-one"
			} else {
				pId = newPId(be)
				pName = be.NewId()
				cmd(be, fmt.Sprintf("/character %s=%s", pId, pName), user1, channel, team1, "", t,
					fmt.Sprintf("Character profile `%s` created with display name \"%s\"", pId, pName),
					[]tAtt{{"**" + pName + "**\n`" + pId + "`",
						blue, characterImg},
					})
				message = fmt.Sprintf("%s: Test message from %s", pId, pName)
			}
			for i := 0; i < postCount; i++ {
				postId := post(be, t, &model.Post{UserId: user1, ChannelId: channel, Message: message}, pId, pName, characterImg)
				pPostIds = append(pPostIds, postId)
			}
			switch status {
			case main.PROFILE_CHARACTER, main.PROFILE_ME:
				break
			case main.PROFILE_CORRUPT:
				// Corrupt the profile
				cmd(be, fmt.Sprintf("/character corrupt1 %s", pId), user1, channel, team1, "", t,
					fmt.Sprintf("Successfully corrupted profile `%s` using method 1.", pId),
					[]tAtt{},
				)
			case main.PROFILE_NONEXISTENT:
				// Delete the profile
				cmd(be, fmt.Sprintf("/character delete %s", pId), user1, channel, team1, "", t,
					fmt.Sprintf("Deleted character profile `%s`.", pId),
					[]tAtt{},
				)
			}
			return pId, pName, pPostIds
		}
		p1Id, p1Name, p1PostIds := setup(testcase.p1PostCount, testcase.p1Status)
		p2Id, p2Name, p2PostIds := setup(testcase.p2PostCount, testcase.p2Status)
		checkPostProfile := func(postIds []string, pId, pName string) {
			for _, postId := range postIds {
				postObj, err := be.GetPost(postId)
				assert.Nil(t, err, msg)
				profileId, ok := postObj.Props["profile_identifier"]
				assert.True(t, ok, msg)
				overrideUsername, ok := postObj.Props["override_username"]
				assert.True(t, ok, msg)
				if main.IsMe(pId) {
					assert.Equal(t, nil, profileId, msg)
					assert.Equal(t, nil, overrideUsername, msg)
				} else {
					assert.Equal(t, pId, profileId, msg)
					assert.Equal(t, pName, overrideUsername, msg)
				}
			}
		}
		tmpl, err := template.New("test").Parse(testcase.msg)
		assert.Nil(t, err, msg)
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, map[string]string{"f": p1Id, "t": p2Id, "c": strconv.Itoa(testcase.p1PostCount)})
		assert.Nil(t, err, msg)
		response := buf.String()
		resProfile := testcase.resProfile
		if testcase.expectSuccess {
			var att []tAtt
			var resName string
			switch resProfile {
			case OLD:
				att = []tAtt{{"**" + p1Name + "**\n`" + p2Id + "`",
					blue, characterImg},
				}
				resName = p1Name
				break
			case NEW:
				att = []tAtt{{"**" + p2Name + "**\n`" + p2Id + "`",
					blue, characterImg},
				}
				resName = p2Name
				if main.IsMe(p2Id) {
					att = []tAtt{{"**user-number-one** *(your real profile)*\n`me`, `myself`",
						green, user1image}}
					resName = ""
				}
				break
			default:
				assert.Fail(t, "Invalid resProfile", msg)
			}
			cmd(be, fmt.Sprintf("/character make %s into %s", p1Id, p2Id), user1, channel, team1, "", t,
				response,
				att)
			checkPostProfile(p1PostIds, p2Id, resName)
			checkPostProfile(p2PostIds, p2Id, resName)
		} else {
			cmdFail(be, fmt.Sprintf("/character make %s into %s", p1Id, p2Id), user1, channel, team1, "", t,
				response,
			)
			checkPostProfile(p1PostIds, p1Id, p1Name)
			checkPostProfile(p2PostIds, p2Id, p2Name)
		}
	}
}

type tAtt struct {
	Text      string
	Color     string
	GetImgURL func(thumb bool) string
}

func cmd(be main.Backend, command, userId, channelId, teamId, rootId string,
	t *testing.T, expectedResponse string, expectedAttachments []tAtt) {
	msg := fmt.Sprintf("Command: %s", command)
	response, attachments, err := main.DoExecuteCommand(be, command, userId, channelId, teamId, rootId, true)
	assert.Nil(t, err, msg)
	assert.Equal(t, expectedResponse, response, msg)
	assert.Equal(t, len(expectedAttachments), len(attachments), msg)
	for i, expectedAttachment := range expectedAttachments {
		assert.Equal(t, expectedAttachment.Text, attachments[i].Text, msg)
		assert.Equal(t, expectedAttachment.Color, attachments[i].Color, msg)
		assert.Equal(t, expectedAttachment.GetImgURL(true), attachments[i].ThumbURL, msg)
	}
}

func cmdFail(be main.Backend, command, userId, channelId, teamId, rootId string,
	t *testing.T, expectedError string) {
	msg := fmt.Sprintf("Command: %s", command)
	response, attachments, err := main.DoExecuteCommand(be, command, userId, channelId, teamId, rootId, true)
	assert.NotNil(t, err, msg)
	assert.Equal(t, expectedError, main.ErrStr(err), msg)
	assert.Equal(t, "", response, msg)
	assert.Equal(t, 0, len(attachments), msg)
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
	if main.IsMe(expectedProfile) {
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

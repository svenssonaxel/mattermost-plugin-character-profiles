package main

import (
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

//go:embed helptext.md
var helpText string

func isMe(id string) bool {
	return id == "" || id == "myself" || id == "me"
}

func DoExecuteCommand(be Backend, command, userId, channelId, teamId, rootId string) (string, []*model.SlackAttachment, *model.AppError) {

	// Make sure command begins correctly with `/character `
	matches := regexp.MustCompile(`^/character (.*)$`).FindStringSubmatch(command)
	if len(matches) != 2 {
		return "", nil, appError("Expected trigger /character but got "+command, nil)
	}
	query := matches[1]

	// `/character help`
	if query == "help" || query == "--help" || query == "h" || query == "-h" {
		return helpText, nil, nil
	}

	// `/character haddock=Captain Haddock`: Create a character profile with identifier `haddock` unless it already exists, and set its display name to `Captain Haddock`.
	// `/character picture haddock=Captain Haddock`: Create a character profile with identifier `haddock` unless it already exists, set its display name to `Captain Haddock`, and set its profile picture to the picture uploaded in the parent message. (Note that you can **not** attach a picture to the slash command itself, for technical reasons.)
	// `/character picture haddock`: Modify an existing character profile by updating the profile picture to the picture uploaded in the parent message, leaving the display name as it is. (Note that you can **not** attach a picture to the slash command itself, for technical reasons.)
	matches = regexp.MustCompile(`^(picture )?([a-z]+)(=.*)?$`).FindStringSubmatch(query)
	if len(matches) == 4 && (matches[1] != "" || matches[3] != "") {
		profileId := matches[2]
		if isMe(profileId) {
			return "", nil, appError("You cannot use `myself` or `me` as a character profile identifyer. Use the Mattermost built-in functionality to change the display name or profile picture for your real Mattermost profile.", nil)
		}
		existed, err := profileExists(be, userId, profileId)
		if err != nil {
			return "", nil, err
		}
		profileDisplayName := strings.TrimPrefix(matches[3], "=")
		setName := matches[3] != ""
		setPicture := matches[1] != ""
		var newProfile Profile
		var newPictureFileId string
		if setPicture {
			if rootId == "" {
				return "", nil, appError("Setting character profile picture can only be done in a thread, with the parent post containing the picture.", nil)
			}
			rootPost, err := be.GetPost(rootId)
			if err != nil {
				return "", nil, err
			}
			if rootPost == nil {
				return "", nil, appError(fmt.Sprintf("Could not fetch root post with id `%s`", rootId), nil)
			}
			if len(rootPost.FileIds) > 1 {
				return "", nil, appError("Parent post cannot have more than one file when creating or modifying a character profile.", nil)
			}
			if len(rootPost.FileIds) == 0 {
				return "", nil, appError("No more than one file when creating or modifying a character profile.", nil)
			}
			newPictureFileId = rootPost.FileIds[0]
			if newPictureFileId == "" {
				return "", nil, appError("Could not find file Id in parent post.", nil)
			}
		}
		var successMessage string
		if existed {
			// Modify character profile
			oldProfile, err := getProfile(be, userId, profileId, PROFILE_CHARACTER|PROFILE_CORRUPT)
			if err != nil {
				return "", nil, err
			}
			newProfile = Profile{
				UserId:        userId,
				Identifier:    profileId,
				Name:          oldProfile.Name,
				PictureFileId: oldProfile.PictureFileId,
				Status:        PROFILE_CHARACTER,
			}
			successMessage = fmt.Sprintf("Character profile `%s` modified by", profileId)
			if setName {
				newProfile.Name = profileDisplayName
				sameName := oldProfile.Name == newProfile.Name
				if sameName {
					successMessage += fmt.Sprintf(" setting the display name to \"%s\" (same as before)", newProfile.Name)
				} else {
					successMessage += fmt.Sprintf(" changing the display name from \"%s\" to \"%s\"", oldProfile.Name, newProfile.Name)
				}
			}
			if setPicture {
				newProfile.PictureFileId = newPictureFileId
				samePicture := oldProfile.PictureFileId == newProfile.PictureFileId
				if setName {
					successMessage += " and"
				}
				if samePicture {
					successMessage += " updating the profile picture (to the same as before)"
				} else {
					successMessage += fmt.Sprintf(" updating the profile picture")
				}
			}
		} else {
			// Create character profile
			if !setName {
				return "", nil, appError(fmt.Sprintf("No character profile with identifyer `%s` exists. In order to create it, you must at least provide a display name. Try `/character help` for details.", profileId), nil)
			}
			newProfile = Profile{
				UserId:     userId,
				Identifier: profileId,
				Name:       profileDisplayName,
				Status:     PROFILE_CHARACTER,
			}
			successMessage = fmt.Sprintf("Character profile `%s` created with display name \"%s\"", newProfile.Identifier, newProfile.Name)
			if setPicture {
				newProfile.PictureFileId = newPictureFileId
				successMessage += " and a profile picture"
			}
		}
		err = setProfile(be, userId, &newProfile)
		if err != nil {
			return "", nil, err
		}
		// Update all existing messages that uses this profile. This is done no
		// matter if the profile existed or not, because it is possible to delete a
		// profile without deleting all messages that use it.
		err = updatePostsUsingProfile(be, userId, profileId)
		if err != nil {
			return "", nil, err
		}
		return successMessage, attachmentsFromProfile(be, newProfile), nil
	}

	// `/character delete haddock`: Delete character profile with identifier `haddock`.
	matches = regexp.MustCompile(`^delete ([a-z]+)$`).FindStringSubmatch(query)
	if len(matches) == 2 {
		profileId := matches[1]
		if isMe(profileId) {
			return "", nil, appError("Please do not try to delete yourself. If you have suicidal thoughts, call 90101 (Sweden) or +1-800-273-8255 (International).", nil)
		}
		exists, err := profileExists(be, userId, profileId)
		if err != nil {
			return "", nil, err
		}
		if !exists {
			return "", nil, appError(fmt.Sprintf("Character profile `%s` does not exist.", profileId), nil)
		}
		err = deleteProfile(be, userId, profileId)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("Deleted character profile `%s`.", profileId), nil, nil
	}

	// `/character list`: List your character profiles.
	if query == "list" {
		profiles, err := listProfiles(be, userId)
		if err != nil {
			return "", nil, err
		}
		return "## Character profiles", attachmentsFromProfiles(be, profiles), nil
	}

	// `/character I am haddock`: Set default character profile identifier for the current channel to `haddock`.
	// `/character I am myself`: Remove the default character profile for the current channel.
	matches = regexp.MustCompile(`^I am ([a-z]+)$`).FindStringSubmatch(query)
	if len(matches) == 2 {
		newProfileId := matches[1]
		oldProfileId, err := getDefaultProfileIdentifier(be, userId, channelId)
		if err != nil {
			return "", nil, err
		}
		if isMe(newProfileId) {
			if isMe(oldProfileId) {
				return "You are already yourself. Multiplicity was a fun movie, but let's leave it at that.", nil, nil
			}
			err := removeDefaultProfile(be, userId, channelId)
			if err != nil {
				return "", nil, err
			}
			realProfile, err := getProfile(be, userId, "", PROFILE_ME)
			if err != nil {
				return "", nil, err
			}
			return "You are now yourself again. Hope that feels ok.", attachmentsFromProfile(be, *realProfile), nil
		} else {
			newProfile, err := setDefaultProfileIdentifier(be, userId, channelId, newProfileId)
			if err != nil {
				return "", nil, err
			}
			if newProfile == nil {
				return "", nil, appError(fmt.Sprintf("Could not fetch profile `%s`.", newProfileId), nil)
			}
			if oldProfileId == newProfileId {
				return fmt.Sprintf("You are already \"%s\", and if that's not enough you should've rolled better stats.", newProfile.Name), nil, nil
			}
			return fmt.Sprintf("You are now known as \"%s\".", newProfile.Name), attachmentsFromProfile(be, *newProfile), nil
		}
	}

	// `/character who am I`: List default character profiles for the channels in this team.
	if query == "who am I" {
		channels, err := be.GetChannelsForTeamForUser(teamId, userId, false)
		if err != nil {
			return "", nil, err
		}
		profileIdToChannelMentions := map[string][]string{}
		// Get default profile identifiers for all channels in this team.
		for _, channel := range channels {
			defaultProfileIdentifier, err := getDefaultProfileIdentifier(be, userId, channel.Id)
			if err != nil {
				return "", nil, err
			}
			channelMention, err := channelMention(be, channel, userId, teamId)
			if err != nil {
				return "", nil, err
			}
			profileIdToChannelMentions[defaultProfileIdentifier] = append(profileIdToChannelMentions[defaultProfileIdentifier], channelMention)
		}
		// Get profiles for all default profile identifiers and sort them.
		profiles := []Profile{}
		for profileId := range profileIdToChannelMentions {
			profile, err := getProfile(be, userId, profileId, PROFILE_CHARACTER|PROFILE_ME|PROFILE_CORRUPT|PROFILE_NONEXISTENT)
			if err != nil {
				return "", nil, err
			}
			if profile != nil {
				profiles = append(profiles, *profile)
			}
		}
		sortProfiles(profiles)
		// Build attachments.
		attachments := make([]*model.SlackAttachment, len(profiles))
		for i, profile := range profiles {
			profileId := profile.Identifier
			channelMentions := profileIdToChannelMentions[profileId]
			sortChannelMentions(channelMentions)
			// Join channel mentions with commas.
			channelNamesString := "\nDefault profile in: " + strings.Join(channelMentions, ", ")
			attachment := attachmentFromProfile(be, profile)
			attachment.Text += channelNamesString
			attachments[i] = attachment
		}
		return "## Default character profiles", attachments, nil
	}

	return "", nil, appError("Unrecognized command. Try `/character help`.", nil)
}

func attachmentFromProfile(be Backend, profile Profile) *model.SlackAttachment {
	switch profile.Status {
	case PROFILE_CHARACTER:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("**%s**\n`%s`", profile.Name, profile.Identifier),
			ThumbURL: profileIconUrl(be, profile, false),
			Color:    "#5c66ff",
		}
	case PROFILE_ME:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("**%s** *(your real profile)*\n`me`, `myself`", profile.Name),
			ThumbURL: profileIconUrl(be, profile, false),
			Color:    "#009900",
		}
	case PROFILE_CORRUPT:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("**%s** *(corrupt profile)*\n`%s`\nError: %s", profile.Name, profile.Identifier, errStr(profile.Error)),
			ThumbURL: profileIconUrl(be, profile, false),
			Color:    "#ff0000",
		}
	case PROFILE_NONEXISTENT:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("*(profile does not exist)*\n`%s`", profile.Identifier),
			ThumbURL: profileIconUrl(be, profile, false),
			Color:    "#ff0000",
		}
	default:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("*(BUG in profile)*\n`%s`", profile.Identifier),
			ThumbURL: "",
			Color:    "#ff0000",
		}
	}
}

func attachmentsFromProfile(be Backend, profile Profile) []*model.SlackAttachment {
	return []*model.SlackAttachment{attachmentFromProfile(be, profile)}
}

func attachmentsFromProfiles(be Backend, profiles []Profile) []*model.SlackAttachment {
	ret := make([]*model.SlackAttachment, len(profiles))
	for i, profile := range profiles {
		ret[i] = attachmentFromProfile(be, profile)
	}
	return ret
}

func channelMention(be Backend, channel *model.Channel, userId string, teamId string) (string, *model.AppError) {
	switch channel.Type {
	case model.CHANNEL_OPEN, model.CHANNEL_PRIVATE:
		return fmt.Sprintf("~%s", channel.Name), nil
	case model.CHANNEL_DIRECT:
		members, err := be.GetChannelMembers(channel.Id, 0, 100)
		if err != nil {
			return "", err
		}
		if members == nil {
			return "", appError(fmt.Sprintf("Channel %s has no members.", channel.Id), nil)
		}
		if len(*members) != 2 {
			return "", appError(fmt.Sprintf("Channel %s has %d members, expected 2.", channel.Id, len(*members)), nil)
		}
		for _, member := range *members {
			if member.UserId != userId {
				user, err := be.GetUser(member.UserId)
				if err != nil {
					return "", err
				}
				if user == nil {
					return "", appError(fmt.Sprintf("User %s does not exist.", member.UserId), nil)
				}
				return fmt.Sprintf("@%s", user.Username), nil
			}
		}
		return "", appError(fmt.Sprintf("Channel %s has no members other than %s.", channel.Id, userId), nil)
	case model.CHANNEL_GROUP:
		team, err := be.GetTeam(teamId)
		if err != nil {
			return "", err
		}
		if team == nil {
			return "", appError(fmt.Sprintf("Team %s does not exist.", teamId), nil)
		}
		teamName := team.Name
		memberNames := []string{}
		members, err := be.GetChannelMembers(channel.Id, 0, 100)
		if err != nil {
			return "", err
		}
		if members == nil {
			return "", appError(fmt.Sprintf("Channel %s has no members.", channel.Id), nil)
		}
		for _, member := range *members {
			if member.UserId == userId {
				continue
			}
			user, err := be.GetUser(member.UserId)
			if err != nil {
				return "", err
			}
			if user == nil {
				return "", appError(fmt.Sprintf("User %s does not exist.", member.UserId), nil)
			}
			memberNames = append(memberNames, user.Username)
		}
		sort.Strings(memberNames)
		if len(memberNames) == 0 {
			return "", appError(fmt.Sprintf("Channel %s has no members other than user %s.", channel.Id, userId), nil)
		}
		if len(memberNames) > 5 {
			memberNames[4] = fmt.Sprintf("%d others", len(memberNames)-4)
			memberNames = memberNames[:5]
		}
		if len(memberNames) > 1 {
			ml := len(memberNames)
			memberNames[ml-2] = memberNames[ml-2] + " and " + memberNames[ml-1]
			memberNames = memberNames[:ml-1]
		}
		return fmt.Sprintf("[Group Chat](%s/%s/messages/%s) with %s", be.GetSiteURL(), teamName, channel.Name, strings.Join(memberNames, ", ")), nil
	default:
		return "", appError(fmt.Sprintf("Unknown channel type %s.", channel.Type), nil)
	}
}

func sortChannelMentions(channelMentions []string) {
	sort.Slice(channelMentions, func(i, j int) bool {
		// Sort channels first, then group chats, then direct messages. Within each category, sort alphabetically.
		getChannelCategory := func(channelMention string) string {
			if strings.HasPrefix(channelMention, "~") {
				return "1: Channel"
			}
			if strings.HasPrefix(channelMention, "[Group") {
				return "2: Group"
			}
			if strings.HasPrefix(channelMention, "@") {
				return "3: Direct message"
			}
			return "4: Unknown"
		}
		iCategory := getChannelCategory(channelMentions[i])
		jCategory := getChannelCategory(channelMentions[j])
		if iCategory != jCategory {
			return iCategory < jCategory
		}
		return channelMentions[i] < channelMentions[j]
	})
}

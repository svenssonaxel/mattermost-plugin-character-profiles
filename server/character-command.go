package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

//go:embed helptext.md
var helpText string

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if p.API == nil {
		return nil, appError("Cannot access the plugin API.", nil)
	}

	userId := args.UserId
	channelId := args.ChannelId
	teamId := args.TeamId

	responseMessage, attachments, err := doExecuteCommand(p, args.Command, userId, channelId, teamId, args.RootId)

	if err != nil {
		return nil, err
	}

	if responseMessage != "" {
		return &model.CommandResponse{
			Username: "Character Profiles",
			// todo IconURL:
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         responseMessage,
			Props: map[string]interface{}{
				"from_webhook": "true",
			},
			Attachments: attachments,
		}, nil
	}

	return nil, appError("Unexpectedly got no return value from doExecuteCommand", nil)
}

func isMe(id string) bool {
	return id == "" || id == "myself" || id == "me"
}

func doExecuteCommand(p *Plugin, command, userId, channelId, teamId, rootId string) (string, []*model.SlackAttachment, *model.AppError) {

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
		existed, err := p.profileExists(userId, profileId)
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
			rootPost, err := p.API.GetPost(rootId)
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
			oldProfile, err := p.getProfile(userId, profileId, PROFILE_CHARACTER|PROFILE_CORRUPT)
			if err != nil {
				return "", nil, err
			}
			newProfile = *oldProfile
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
		err = p.setProfile(userId, &newProfile)
		if err != nil {
			return "", nil, err
		}
		return successMessage, p.attachmentsFromProfile(newProfile), nil
	}

	// `/character delete haddock`: Delete character profile with identifier `haddock`.
	matches = regexp.MustCompile(`^delete ([a-z]+)$`).FindStringSubmatch(query)
	if len(matches) == 2 {
		profileId := matches[1]
		if isMe(profileId) {
			return "", nil, appError("Please do not try to delete yourself. If you have suicidal thoughts, call 90101 (Sweden) or +1-800-273-8255 (International).", nil)
		}
		exists, err := p.profileExists(userId, profileId)
		if err != nil {
			return "", nil, err
		}
		if !exists {
			return "", nil, appError(fmt.Sprintf("Character profile `%s` does not exist.", profileId), nil)
		}
		err = p.deleteProfile(userId, profileId)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("Deleted character profile `%s`.", profileId), nil, nil
	}

	// `/character list`: List your character profiles.
	if query == "list" {
		profiles, err := p.listProfiles(userId)
		if err != nil {
			return "", nil, err
		}
		return "## Character profiles", p.attachmentsFromProfiles(profiles), nil
	}

	// `/character I am haddock`: Set default character profile identifier for the current channel to `haddock`.
	// `/character I am myself`: Remove the default character profile for the current channel.
	matches = regexp.MustCompile(`^I am ([a-z]+)$`).FindStringSubmatch(query)
	if len(matches) == 2 {
		newProfileId := matches[1]
		oldProfileId, err := p.getDefaultProfileIdentifier(userId, channelId)
		if err != nil {
			return "", nil, err
		}
		if isMe(newProfileId) {
			if isMe(oldProfileId) {
				return "You are already yourself. Multiplicity was a fun movie, but let's leave it at that.", nil, nil
			}
			err := p.removeDefaultProfile(userId, channelId)
			if err != nil {
				return "", nil, err
			}
			realProfile, err := p.getProfile(userId, "", PROFILE_ME)
			if err != nil {
				return "", nil, err
			}
			return "You are now yourself again. Hope that feels ok.", p.attachmentsFromProfile(*realProfile), nil
		} else {
			newProfile, err := p.setDefaultProfileIdentifier(userId, channelId, newProfileId)
			if err != nil {
				return "", nil, err
			}
			if newProfile == nil {
				return "", nil, appError(fmt.Sprintf("Could not fetch profile `%s`.", newProfileId), nil)
			}
			if oldProfileId == newProfileId {
				return fmt.Sprintf("You are already \"%s\", and if that's not enough you should've rolled better stats.", newProfile.Name), nil, nil
			}
			return fmt.Sprintf("You are now known as \"%s\".", newProfile.Name), p.attachmentsFromProfile(*newProfile), nil
		}
	}

	// `/character who am I`: List default character profiles for the channels in this team.
	if query == "who am I" {
		channels, err := p.API.GetChannelsForTeamForUser(teamId, userId, false)
		if err != nil {
			return "", nil, err
		}
		profileIdToChannelMentions := map[string][]string{}
		// Get default profile identifiers for all channels in this team.
		for _, channel := range channels {
			defaultProfileIdentifier, err := p.getDefaultProfileIdentifier(userId, channel.Id)
			if err != nil {
				return "", nil, err
			}
			channelMention, err := p.channelMention(channel, userId, teamId)
			if err != nil {
				return "", nil, err
			}
			profileIdToChannelMentions[defaultProfileIdentifier] = append(profileIdToChannelMentions[defaultProfileIdentifier], channelMention)
		}
		// Get profiles for all default profile identifiers and sort them.
		profiles := []Profile{}
		for profileId := range profileIdToChannelMentions {
			profile, err := p.getProfile(userId, profileId, PROFILE_CHARACTER|PROFILE_ME|PROFILE_CORRUPT|PROFILE_NONEXISTENT)
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
			attachment := p.attachmentFromProfile(profile)
			attachment.Text += channelNamesString
			attachments[i] = attachment
		}
		return "## Default character profiles", attachments, nil
	}

	return "", nil, appError("Unrecognized command. Try `/character help`.", nil)
}

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Character Profile Plugin", message, nil, errorMessage, http.StatusBadRequest)
}

func (p *Plugin) attachmentFromProfile(profile Profile) *model.SlackAttachment {
	switch profile.Status {
	case PROFILE_CHARACTER:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("**%s**\n`%s`", profile.Name, profile.Identifier),
			ThumbURL: p.profileIconUrl(profile, false),
			Color:    "#5c66ff",
		}
	case PROFILE_ME:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("**%s** *(your real profile)*\n`me`, `myself`", profile.Name),
			ThumbURL: p.profileIconUrl(profile, false),
			Color:    "#009900",
		}
	case PROFILE_CORRUPT:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("**%s** *(corrupt profile)*\n`%s`\nError: %s", profile.Name, profile.Identifier, profile.Error.Error()),
			ThumbURL: p.profileIconUrl(profile, false),
			Color:    "#ff0000",
		}
	case PROFILE_NONEXISTENT:
		return &model.SlackAttachment{
			Text:     fmt.Sprintf("*(profile does not exist)*\n`%s`", profile.Identifier),
			ThumbURL: p.profileIconUrl(profile, false),
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

func (p *Plugin) attachmentsFromProfile(profile Profile) []*model.SlackAttachment {
	return []*model.SlackAttachment{p.attachmentFromProfile(profile)}
}

func (p *Plugin) attachmentsFromProfiles(profiles []Profile) []*model.SlackAttachment {
	ret := make([]*model.SlackAttachment, len(profiles))
	for i, profile := range profiles {
		ret[i] = p.attachmentFromProfile(profile)
	}
	return ret
}

func (p *Plugin) channelMention(channel *model.Channel, userId string, teamId string) (string, *model.AppError) {
	switch channel.Type {
	case model.CHANNEL_OPEN, model.CHANNEL_PRIVATE:
		return fmt.Sprintf("~%s", channel.Name), nil
	case model.CHANNEL_DIRECT:
		members, err := p.API.GetChannelMembers(channel.Id, 0, 100)
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
				user, err := p.API.GetUser(member.UserId)
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
		team, err := p.API.GetTeam(teamId)
		if err != nil {
			return "", err
		}
		if team == nil {
			return "", appError(fmt.Sprintf("Team %s does not exist.", teamId), nil)
		}
		teamName := team.Name
		memberNames := []string{}
		members, err := p.API.GetChannelMembers(channel.Id, 0, 100)
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
			user, err := p.API.GetUser(member.UserId)
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
		return fmt.Sprintf("[Group Chat](%s/%s/messages/%s) with %s", p.siteURL, teamName, channel.Name, strings.Join(memberNames, ", ")), nil
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

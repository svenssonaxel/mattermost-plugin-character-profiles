package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

func (p *Plugin) OnActivate() error {
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "character",
		Description:      "Become a nomad of names, a litany of labels, to master monikers and fabricate fables.",
		DisplayName:      "Character profiles",
		AutoComplete:     true,
		AutoCompleteDesc: "Try `/character help` to become a nomad of names, a litany of labels, to master monikers and fabricate fables.",
		AutoCompleteHint: "haddock=Captain Haddock",
	})
}

//go:embed helptext.md
var helpText string

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if p.API == nil {
		return nil, appError("Cannot access the plugin API.", nil)
	}

	userId := args.UserId
	channelId := args.ChannelId
	teamId := args.TeamId

	responseMessage, err := doExecuteCommand(p, args.Command, userId, channelId, teamId, args.RootId)

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
		}, nil
	}

	return nil, appError("Unexpectedly got no return value from doExecuteCommand", nil)
}

func isMe(id string) bool {
	return id == "" || id == "myself" || id == "me"
}

func doExecuteCommand(p *Plugin, command, userId, channelId, teamId, rootId string) (string, *model.AppError) {

	// Make sure command begins correctly with `/character `
	matches := regexp.MustCompile(`^/character (.*)$`).FindStringSubmatch(command)
	if len(matches) != 2 {
		return "", appError("Expected trigger /character but got "+command, nil)
	}
	query := matches[1]

	// `/character help`
	if query == "help" || query == "--help" || query == "h" || query == "-h" {
		return helpText, nil
	}

	// `/character haddock=Captain Haddock`: Create a character profile with identifier `haddock` unless it already exists, and set its display name to `Captain Haddock`.
	// `/character picture haddock=Captain Haddock`: Create a character profile with identifier `haddock` unless it already exists, set its display name to `Captain Haddock`, and set its profile picture to the picture uploaded in the parent message. (Note that you can **not** attach a picture to the slash command itself, for technical reasons.)
	// `/character picture haddock`: Modify an existing character profile by updating the profile picture to the picture uploaded in the parent message, leaving the display name as it is. (Note that you can **not** attach a picture to the slash command itself, for technical reasons.)
	matches = regexp.MustCompile(`^(picture )?([a-z]+)(=.*)?$`).FindStringSubmatch(query)
	if len(matches) == 4 && (matches[1] != "" || matches[3] != "") {
		profileId := matches[2]
		if isMe(profileId) {
			return "", appError("You cannot use `myself` or `me` as a character profile identifyer. Use the Mattermost built-in functionality to change the display name or profile picture for your real Mattermost profile.", nil)
		}
		existed, err := p.profileExists(userId, profileId)
		if err != nil {
			return "", err
		}
		profileDisplayName := strings.TrimPrefix(matches[3], "=")
		setName := matches[3] != ""
		setPicture := matches[1] != ""
		var newProfile Profile
		var newPictureFileId string
		if setPicture {
			if rootId == "" {
				return "", appError("Setting character profile picture can only be done in a thread, with the parent post containing the picture.", nil)
			}
			rootPost, err := p.API.GetPost(rootId)
			if err != nil {
				return "", err
			}
			if rootPost == nil {
				return "", appError(fmt.Sprintf("Could not fetch root post with id `%s`", rootId), nil)
			}
			if len(rootPost.FileIds) > 1 {
				return "", appError("Parent post cannot have more than one file when creating or modifying a character profile.", nil)
			}
			if len(rootPost.FileIds) == 0 {
				return "", appError("No more than one file when creating or modifying a character profile.", nil)
			}
			newPictureFileId = rootPost.FileIds[0]
			if newPictureFileId == "" {
				return "", appError("Could not find file Id in parent post.", nil)
			}
		}
		var successMessage string
		if existed {
			// Modify character profile
			oldProfile, err := p.getProfile(userId, profileId, false)
			if err != nil {
				return "", err
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
				return "", appError(fmt.Sprintf("No character profile with identifyer `%s` exists. In order to create it, you must at least provide a display name. Try `/character help` for details.", profileId), nil)
			}
			newProfile = Profile{
				Identifier: profileId,
				Name:       profileDisplayName,
			}
			successMessage = fmt.Sprintf("Character profile `%s` created with display name \"%s\"", newProfile.Identifier, newProfile.Name)
			if setPicture {
				newProfile.PictureFileId = newPictureFileId
				successMessage += " and a profile picture"
			}
		}
		err = p.setProfile(userId, &newProfile)
		if err != nil {
			return "", err
		}
		return successMessage, nil
	}

	// `/character delete haddock`: Delete character profile with identifier `haddock`.
	matches = regexp.MustCompile(`^delete ([a-z]+)$`).FindStringSubmatch(query)
	if len(matches) == 2 {
		profileId := matches[1]
		if isMe(profileId) {
			return "", appError("Please do not try to delete yourself. If you have suicidal thoughts, call 90101 (Sweden) or +1-800-273-8255 (International).", nil)
		}
		exists, err := p.profileExists(userId, profileId)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", appError(fmt.Sprintf("Character profile `%s` does not exist.", profileId), nil)
		}
		err = p.deleteProfile(userId, profileId)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted character profile `%s`.", profileId), nil
	}

	// `/character list`: List your character profiles.
	if query == "list" {
		profiles, err := p.listProfiles(userId)
		if err != nil {
			return "", err
		}
		if len(profiles) == 0 {
			return "You have no character profiles yet.", nil
		}
		ret := "|Identifier|Profile Picture|Display name|\n|---|---|---|"
		for _, profile := range profiles {
			ret += fmt.Sprintf("\n|%s|![](%s)|%s|", profile.Identifier, profileIconUrl(profile.PictureFileId, true), profile.Name)
		}
		return ret, nil
	}

	// `/character I am haddock`
	// `/character I am myself`
	matches = regexp.MustCompile(`^I am ([a-z]+)$`).FindStringSubmatch(query)
	if len(matches) == 2 {
		newProfileId := matches[1]
		oldProfileId, err := p.getDefaultProfileIdentifier(userId, channelId)
		if err != nil {
			return "", err
		}
		if isMe(newProfileId) {
			if isMe(oldProfileId) {
				return "You are already yourself. Multiplicity was a fun movie, but let's leave it at that.", nil
			}
			err := p.removeDefaultProfile(userId, channelId)
			if err != nil {
				return "", err
			}
			return "You are now yourself again. Hope that feels ok.", nil
		} else {
			profileDisplayName, err := p.setDefaultProfileIdentifier(userId, channelId, newProfileId)
			if err != nil {
				return "", err
			}
			if oldProfileId == newProfileId {
				return fmt.Sprintf("You are already \"%s\", and if that's not enough you should've rolled better stats.", profileDisplayName), nil
			}
			return fmt.Sprintf("You are now known as \"%s\".", profileDisplayName), nil
		}
	}

	return "", appError("Unrecognized command. Try `/character help`.", nil)
}

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Character Profile Plugin", message, nil, errorMessage, http.StatusBadRequest)
}

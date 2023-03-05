package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"regexp"

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

	responseMessage, err := doExecuteCommand(p, args.Command, userId, channelId, teamId)

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

func doExecuteCommand(p *Plugin, command, userId, channelId, teamId string) (string, *model.AppError) {

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

	// `/character haddock=Captain Haddock`: Create or overwrite a character profile with identifier `haddock` and set its display name to `Captain Haddock`.
	matches = regexp.MustCompile(`^([a-z]+)=(.*)$`).FindStringSubmatch(query)
	if len(matches) == 3 {
		profileId := matches[1]
		if isMe(profileId) {
			return "", appError("You cannot use `myself` or `me` as a character profile identifyer. Use the Mattermost built-in functionality to change your display name.", nil)
		}
		existed, err := p.profileExists(userId, profileId)
		if err != nil {
			return "", err
		}
		profileDisplayName := matches[2]
		err = p.setProfile(userId, Profile{Identifier: profileId, Name: profileDisplayName})
		if err != nil {
			return "", err
		}
		if existed {
			return fmt.Sprintf("Altered character profile `%s` with display name `%s`", profileId, profileDisplayName), nil
		} else {
			return fmt.Sprintf("Saved character profile `%s` with display name `%s`", profileId, profileDisplayName), nil
		}
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
		ret := "|Identifier|Display name|\n|---|---|"
		for _, profile := range profiles {
			ret += fmt.Sprintf("\n|%s|%s|", profile.Identifier, profile.Name)
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
				return "", appError("You are already yourself. Multiplicity was a fun movie, but let's leave it at that.", nil)
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
				return "", appError(fmt.Sprintf("You are already \"%s\", and if that's not enough you should've rolled better stats.", profileDisplayName), nil)
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

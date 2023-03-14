package main

import (
	"net/http"
	"path/filepath"
	"sync"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// Mockable backend, the only thing passed to non-glue code.
	backend *Backend
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if p.backend == nil {
		return nil, appError("Cannot access the plugin backend.", nil)
	}

	userId := args.UserId
	channelId := args.ChannelId
	teamId := args.TeamId

	responseMessage, attachments, err := doExecuteCommand(*p.backend, args.Command, userId, channelId, teamId, args.RootId)

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

func getRealBackendFromPlugin(p *Plugin) (Backend, *model.AppError) {
	backend := BackendImpl{}
	maybeSiteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if maybeSiteURL == nil {
		return backend, model.NewAppError("backendFromPlugin", "Cannot get Site URL", nil, "", http.StatusInternalServerError)
	}
	backend.SiteURL = *maybeSiteURL
	if p.API == nil {
		return backend, model.NewAppError("backendFromPlugin", "Cannot get API", nil, "", http.StatusInternalServerError)
	}
	backend.API = p.API
	return backend, nil
}

func (p *Plugin) OnActivate() error {
	backend, beErr := getRealBackendFromPlugin(p)
	if beErr != nil {
		return beErr
	}
	p.backend = &backend
	err := p.API.RegisterCommand(&model.Command{
		Trigger:          "character",
		Description:      "Become a nomad of names, a litany of labels, to master monikers and fabricate fables.",
		DisplayName:      "Character profiles",
		AutoComplete:     true,
		AutoCompleteDesc: "Try `/character help` to become a nomad of names, a litany of labels, to master monikers and fabricate fables.",
		AutoCompleteHint: "haddock=Captain Haddock",
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) MessageWillBePosted(_ *plugin.Context, post *model.Post) (*model.Post, string) {
	if p.backend == nil {
		return nil, "Backend not initialized"
	}
	return ProfiledPost(*p.backend, post, false)
}

func (p *Plugin) MessageWillBeUpdated(_ *plugin.Context, newPost *model.Post, _ *model.Post) (*model.Post, string) {
	if p.backend == nil {
		return nil, "Backend not initialized"
	}
	return ProfiledPost(*p.backend, newPost, true)
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		p.API.LogWarn("Failed to get bundle path", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filepath.Join(bundlePath, "assets", r.URL.Path))
}

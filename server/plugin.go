package main

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	PLUGIN_ID       = "com.axelsvensson.mattermost-plugin-character-profiles"
	BOT_DISPLAYNAME = "Character Profiles"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	router *mux.Router

	// Mockable backend, the only thing passed to non-glue code.
	backend Backend
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if p.backend == nil {
		return nil, appError("Cannot access the plugin backend.", nil)
	}

	userId := args.UserId
	channelId := args.ChannelId
	teamId := args.TeamId

	responseMessage, attachments, err := DoExecuteCommand(p.backend, args.Command, userId, channelId, teamId, args.RootId, false)

	if err != nil {
		return nil, err
	}

	if responseMessage != "" || len(attachments) > 0 {
		iconURL := GetPluginURL(p.backend) + "/static/botprofilepicture"
		return &model.CommandResponse{
			Username:     BOT_DISPLAYNAME,
			IconURL:      iconURL,
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         responseMessage,
			Attachments:  attachments,
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
	bundlePath, bpErr := p.API.GetBundlePath()
	if bpErr != nil {
		return backend, model.NewAppError("backendFromPlugin", "Cannot get bundle path", nil, "", http.StatusInternalServerError)
	}
	backend.BundlePath = bundlePath
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
	p.backend = backend
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
	p.router = routerFromBackend(p.backend)
	return nil
}

func (p *Plugin) MessageWillBePosted(_ *plugin.Context, post *model.Post) (*model.Post, string) {
	if p.backend == nil {
		return nil, "Backend not initialized"
	}
	return ProfiledPost(p.backend, post, false)
}

func (p *Plugin) MessageWillBeUpdated(_ *plugin.Context, newPost *model.Post, _ *model.Post) (*model.Post, string) {
	if p.backend == nil {
		return nil, "Backend not initialized"
	}
	return ProfiledPost(p.backend, newPost, true)
}

func (p *Plugin) MessageHasBeenPosted(_ *plugin.Context, post *model.Post) {
	err := RegisterPost(p.backend, post)
	if err != nil {
		p.API.LogError("Failed to register message", "error", err.Error())
	}
}

func (p *Plugin) MessageHasBeenUpdated(_ *plugin.Context, newPost *model.Post, _ *model.Post) {
	err := RegisterPost(p.backend, newPost)
	if err != nil {
		p.API.LogError("Failed to register message", "error", err.Error())
	}
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	be := p.backend
	if be == nil {
		http.Error(w, "Backend not initialized", http.StatusInternalServerError)
		return
	}
	p.router.ServeHTTP(w, r)
}

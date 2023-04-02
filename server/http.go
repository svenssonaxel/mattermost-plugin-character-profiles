package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/api4"
	"github.com/mattermost/mattermost-server/v5/model"
)

func routerFromBackend(be Backend) *mux.Router {
	router := mux.NewRouter()
	router.Use(checkAuthenticity)
	// Serve static files from /static
	router.HandleFunc("/static/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		serveStaticFile(be, w, r, mux.Vars(r)["path"])
	})
	// Serve profile images from /profile
	router.HandleFunc("/profile/{userId:[a-z0-9]{26}}/{profileId:[a-z]+}", func(w http.ResponseWriter, r *http.Request) {
		serveProfileImage(be, w, r, mux.Vars(r)["userId"], mux.Vars(r)["profileId"], r.URL.Query().Get("rk"), false)
	})
	router.HandleFunc("/profile/{userId:[a-z0-9]{26}}/{profileId:[a-z]+}/thumbnail", func(w http.ResponseWriter, r *http.Request) {
		serveProfileImage(be, w, r, mux.Vars(r)["userId"], mux.Vars(r)["profileId"], r.URL.Query().Get("rk"), true)
	})
	router.HandleFunc("/api/v1/confirm", func(w http.ResponseWriter, r *http.Request) {
		serveConfirm(be, w, r)
	})
	router.HandleFunc("/api/v1/echo", func(w http.ResponseWriter, r *http.Request) {
		serveEcho(be, w, r)
	})
	return router
}

func checkAuthenticity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Mattermost-User-ID") == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Map of paths to files in the static directory
var staticFiles = map[string]string{
	"botprofilepicture":                 "pluginicon.png",
	"botprofilepicture/thumbnail":       "pluginicon-thumbnail.jpeg",
	"defaultprofilepicture":             "character.png",
	"defaultprofilepicture/thumbnail":   "character-thumbnail.jpeg",
	"corruptedprofilepicture":           "no-sign.jpg",
	"corruptedprofilepicture/thumbnail": "no-sign-thumbnail.jpg",
}

func serveStaticFile(be Backend, w http.ResponseWriter, r *http.Request, path string) {
	filename, ok := staticFiles[path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(be.GetBundlePath(), "assets", filename))
}

func serveProfileImage(be Backend, w http.ResponseWriter, r *http.Request, userId string, profileId string, requestKey string, thumbnail bool) {
	profile, err := GetProfile(be, userId, profileId, PROFILE_CORRUPT|PROFILE_ME|PROFILE_CHARACTER|PROFILE_NONEXISTENT)
	if profile == nil && err != nil {
		http.Error(w, ErrStr(err), http.StatusInternalServerError)
		return
	}
	if profile.Status == PROFILE_ME {
		http.Error(w, "Use Mattermost built-in API to get profile pictures for real profiles", http.StatusNotFound)
		return
	}
	if profile.Status == PROFILE_CORRUPT || profile.Status == PROFILE_NONEXISTENT {
		if thumbnail {
			serveStaticFile(be, w, r, "corruptedprofilepicture/thumbnail")
		} else {
			serveStaticFile(be, w, r, "corruptedprofilepicture")
		}
		return
	}
	if profile.Status != PROFILE_CHARACTER {
		http.Error(w, "Bug in profile status handling", http.StatusInternalServerError)
		return
	}
	if profile.RequestKey == "" {
		http.Error(w, "Profile image request key not set", http.StatusInternalServerError)
		return
	}
	if profile.RequestKey != requestKey {
		http.Error(w, "Invalid request key", http.StatusForbidden)
		return
	}
	info := profile.PictureFileInfo
	if info == nil {
		http.Error(w, "Could not get file info", http.StatusInternalServerError)
		return
	}
	// Serve the file. Some of this code is copied and refactored from
	// mattermost-server/api4/file.go.
	path := ""
	contentType := ""
	if thumbnail {
		path = info.ThumbnailPath
		contentType = api4.ThumbnailImageType
	} else {
		path = info.Path
		contentType = info.MimeType
	}
	if path == "" {
		http.NotFound(w, r)
		return
	}
	content, cErr := be.ReadFile(path)
	if cErr != nil {
		http.Error(w, ErrStr(cErr), http.StatusInternalServerError)
		return
	}
	filename := profile.Identifier + filepath.Ext(info.Name)
	if contentType == "" {
		contentType = "application/octet-stream"
	} else {
		for _, unsafeContentType := range api4.UnsafeContentTypes {
			if strings.HasPrefix(contentType, unsafeContentType) {
				contentType = "text/plain"
				break
			}
		}
	}
	header := w.Header()
	header.Set("Cache-Control", "private, immutable, max-age=604800")
	header.Set("Content-Disposition", "inline;filename=\""+filename+"\"; filename*=UTF-8''"+filename)
	header.Set("Content-Security-Policy", "Frame-ancestors 'none'")
	header.Set("Content-Type", contentType)
	header.Set("Last-Modified", time.Unix(0, info.UpdateAt*int64(1000*1000)).UTC().Format(http.TimeFormat))
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	w.Write(content)
}

// Subset of model.PostActionIntegrationRequest
type PAIR struct {
	UserId    string `json:"user_id"`
	PostId    string `json:"post_id"`
	ChannelId string `json:"channel_id"`
	TeamId    string `json:"team_id"`
	Command   string `json:"context.command"`
	Message   string `json:"context.message"`
	RootId    string `json:"context.root_id"`
}

func servePAIR(be Backend, w http.ResponseWriter, r *http.Request, f func(be Backend, w http.ResponseWriter, ir PAIR) (string, model.StringInterface)) {
	var ir PAIR
	dErr := json.NewDecoder(r.Body).Decode(&ir)
	if dErr != nil {
		http.Error(w, dErr.Error(), http.StatusBadRequest)
		return
	}
	if ir.UserId != r.Header.Get("Mattermost-User-ID") {
		http.Error(w, "User ID mismatch", http.StatusBadRequest)
		return
	}
	msg, props := f(be, w, ir)
	be.UpdateEphemeralPost(ir.UserId, &model.Post{
		Id:        ir.PostId,
		UserId:    ir.UserId,
		ChannelId: ir.ChannelId,
		RootId:    ir.RootId,
		Message:   msg,
		Props:     props,
	})
	w.WriteHeader(http.StatusOK)
}

func serveConfirm(be Backend, w http.ResponseWriter, r *http.Request) {
	servePAIR(be, w, r, func(be Backend, w http.ResponseWriter, ir PAIR) (string, model.StringInterface) {
		msg, attachments, eErr := DoExecuteCommand(be, ir.Command, ir.UserId, ir.ChannelId, ir.TeamId, ir.RootId, true)
		if eErr != nil {
			msg, attachments = uiError(fmt.Sprintf("Command `%s` failed:\n%s", ir.Command, eErr.Error()), ir.Command, ir.RootId)
		}
		return msg, model.StringInterface{
			"attachments":       attachments,
			"override_username": BOT_DISPLAYNAME,
			"override_icon_url": GetPluginURL(be) + "/static/botprofilepicture",
			"from_webhook":      "true",
		}
	})
}

func serveEcho(be Backend, w http.ResponseWriter, r *http.Request) {
	servePAIR(be, w, r, func(be Backend, w http.ResponseWriter, ir PAIR) (string, model.StringInterface) {
		iconURL := GetPluginURL(be) + "/static/botprofilepicture"
		return ir.Message, model.StringInterface{
			"override_username": BOT_DISPLAYNAME,
			"override_icon_url": iconURL,
			"from_webhook":      "true",
		}
	})
}

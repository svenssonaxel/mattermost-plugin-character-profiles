package main

import (
	"net/http"

	"github.com/mattermost/mattermost-server/v5/model"
)

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Character Profile Plugin", message, nil, errorMessage, http.StatusBadRequest)
}

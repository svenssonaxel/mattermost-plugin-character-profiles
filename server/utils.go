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

func ErrStr(err *model.AppError) string {
	if err == nil {
		return "Error is nil"
	}
	return err.Where + ": " + err.Message
}

func appErrorPre(prefix string, err *model.AppError) *model.AppError {
	if err == nil {
		return nil
	}
	errCopy := *err
	errCopy.Message = prefix + err.Message
	return &errCopy
}

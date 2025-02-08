package main

import (
	"errors"
	"fmt"
	"net/http"
)

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.writeError(w, err)
		return
	}

	// Validate

	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.writeError(w, errors.New("invalid credential"), http.StatusBadRequest)
		return
	}

	// Compare password
	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.writeError(w, errors.New("invalid credential"), http.StatusBadRequest)
		return
	}

	res := jsonResponse{
		Error:   false,
		Message: fmt.Sprint("Welcome, ", user.FirstName),
		Data:    user,
	}

	app.writeJson(w, http.StatusOK, res)
}

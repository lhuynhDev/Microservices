package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type requestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth",omitempty`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	_ = app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) HandleSumition(w http.ResponseWriter, r *http.Request) {

	var requestPayload requestPayload
	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.writeError(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	default:
		app.writeError(w, errors.New("invalid action"), http.StatusBadRequest)
	}
}

func (app *Config) authenticate(w http.ResponseWriter, auth AuthPayload) {

	jsonData, err := json.MarshalIndent(auth, "", " \t")
	if err != nil {
		app.writeError(w, err)
		return
	}

	// Call the authentication microservice
	request, err := http.NewRequest("POST",
		"http://authenticate-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.writeError(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.writeError(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		app.writeError(w, errors.New("invalid credential"), http.StatusBadRequest)
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.writeError(w, errors.New("error calling authentication service"), http.StatusInternalServerError)
		return
	}
	var responsePayload jsonResponse
	err = json.NewDecoder(response.Body).Decode(&responsePayload)
	if err != nil {
		app.writeError(w, err)
		return
	}

	if responsePayload.Error {
		app.writeError(w, errors.New(responsePayload.Message), http.StatusBadRequest)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = responsePayload.Message
	payload.Data = responsePayload.Data
	app.writeJson(w, http.StatusOK, payload)
}

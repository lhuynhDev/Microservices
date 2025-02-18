package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/rpc"
	"time"

	"github.com/lhuynhDev/Microservices/broker/event"
	"github.com/lhuynhDev/Microservices/broker/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   mailPayload `json:"mail,omitempty"`
}

type mailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type RPCPayload struct {
	Name string
	Data string
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	_ = app.writeJson(w, http.StatusOK, payload)
}

// HandleSubmission is the main point of entry into the broker. It accepts a JSON
// payload and performs an action based on the value of "action" in that JSON.
func (app *Config) HandleSumition(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.writeError(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	case "log":
		//app.logItem(w, requestPayload.Log)
		//app.logEventViaRabbit(w, requestPayload.Log)
		app.LogItemViaRPC(w, requestPayload.Log)

	case "mail":
		app.sendMail(w, requestPayload.Mail)
	default:
		app.writeError(w, errors.New("unknown action"))
	}
}

// authenticate calls the authentication microservice and sends back the appropriate response
func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.writeError(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.writeError(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.writeError(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.writeError(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.writeError(w, err)
		return
	}

	if jsonFromService.Error {
		app.writeError(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated!"
	payload.Data = jsonFromService.Data

	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) logItem(w http.ResponseWriter, log LogPayload) {
	// create some json we'll send to the log microservice
	jsonData, _ := json.MarshalIndent(log, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
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

	// make sure we get back the correct status code
	if response.StatusCode != http.StatusAccepted {
		app.writeError(w, errors.New("error calling log service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the log service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.writeError(w, err)
		return
	}

	if jsonFromService.Error {
		app.writeError(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged!"
	payload.Data = jsonFromService.Data

	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) sendMail(w http.ResponseWriter, m mailPayload) {
	// create some json we'll send to the mail microservice
	jsonData, _ := json.MarshalIndent(m, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://mailer-service/send", bytes.NewBuffer(jsonData))
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

	// make sure we get back the correct status code
	if response.StatusCode != http.StatusAccepted {
		app.writeError(w, errors.New("error calling mail service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the mail service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.writeError(w, err)
		return
	}

	if jsonFromService.Error {
		app.writeError(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Mail sent to " + m.To
	payload.Data = jsonFromService.Data

	app.writeJson(w, http.StatusAccepted, payload)
}

// logEvent with RabbitMQ
func (app *Config) logEventViaRabbit(w http.ResponseWriter, logPayLoad LogPayload) {
	if err := app.pushToQueue(logPayLoad.Name, logPayLoad.Data); err != nil {
		app.writeError(w, err)
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged by RabbitMQ!"

	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.RabbitConn)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	jsonData, _ := json.MarshalIndent(payload, "", "\t")
	if err := emitter.Push(string(jsonData), "log.INFO"); err != nil {
		return err
	}

	return nil
}

func (app *Config) LogItemViaRPC(w http.ResponseWriter, log LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		app.writeError(w, err)
		return
	}

	rpcPayload := RPCPayload{
		Name: log.Name,
		Data: log.Data,
	}

	var response string
	if err := client.Call("RPCServer.LogInfo", rpcPayload, &response); err != nil {
		app.writeError(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = response

	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) LogItemViaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	if err := app.readJson(w, r, &requestPayload); err != nil {
		app.writeError(w, err)
		return
	}

	conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.writeError(w, errors.New("failed to dial logger-service"))
		return
	}
	defer conn.Close()
	client := logs.NewLoggerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})
	if err != nil {
		app.writeError(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged by gRPC!"

	app.writeJson(w, http.StatusAccepted, payload)
}

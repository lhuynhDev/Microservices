package main

import (
	"context"
	"log"
	"log-service/data"
	"time"
)

type RPCServer struct {
}

type RPCPayload struct {
	Name string
	Data string
}

func (r *RPCServer) LogInfo(payload RPCPayload, response *string) error {
	collection := client.Database("logger").Collection("logs")
	_, err := collection.InsertOne(context.TODO(), data.LogEntry{
		Name:      payload.Name,
		Data:      payload.Data,
		CreatedAt: time.Now(),
	})
	if err != nil {
		log.Println("Error inserting log entry: ", err)
		return err
	}

	*response = " Processed payload by RPC: " + payload.Name
	return nil
}

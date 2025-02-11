package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"log-service/logs"
	"net"

	"google.golang.org/grpc"
)

type LogService struct {
	logs.UnimplementedLoggerServiceServer
	Models data.Models
}

func (app *Config) gRPCListen() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", gRpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	logs.RegisterLoggerServiceServer(s, &LogService{Models: app.Models})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	log.Println("gRPC server started on port", gRpcPort)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (l *LogService) WriteLog(ctx context.Context,
	req *logs.LogRequest) (*logs.LogResponse, error) {
	entry := req.GetLogEntry()

	//Wrtie to the database
	logEntry := data.LogEntry{
		Name: entry.Name,
		Data: entry.Data,
	}

	if err := l.Models.LogEntry.Insert(logEntry); err != nil {
		return nil, err
	}

	return &logs.LogResponse{
		Message: "Log entry created",
	}, nil
}

package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = "80"

type Config struct {
	RabbitConn *amqp.Connection
}

func connect() (*amqp.Connection, error) {
	var count int64
	var backOff = 1 * time.Second
	var conn *amqp.Connection

	for {
		c, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err != nil {
			fmt.Println("Failed to connect to RabbitMQ")
			count++
		} else {
			log.Println("Connected to RabbitMQ")
			conn = c
			break
		}
		if count > 5 {
			fmt.Println("Failed to connect to RabbitMQ after 5 attempts")
			return nil, err
		}
		backOff = time.Duration(math.Pow(2, float64(count))) * time.Second
		log.Printf("Retrying in %d seconds", backOff)
		time.Sleep(backOff)
		continue

	}
	return conn, nil
}

func main() {

	// Connect to rabbitmq
	rabbitConn, err := connect()
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}
	defer rabbitConn.Close()

	app := Config{
		RabbitConn: rabbitConn,
	}

	// define http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}
	log.Printf("Starting broker service on port %s\n", webPort)
	// start the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

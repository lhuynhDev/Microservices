package main

import (
	"fmt"
	"log"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const ()

func connect() (*amqp.Connection, error) {
	var count int64
	var backOff = 1 * time.Second
	var conn *amqp.Connection

	for {
		c, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
		if err != nil {
			fmt.Println("Failed to connect to RabbitMQ")
			count++
		} else {
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
	log.Println("Connected to RabbitMQ")

	// Start to listen to the queue

	// Create the consumer

	// watch the queue and consume the events

}

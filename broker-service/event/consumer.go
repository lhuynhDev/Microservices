package event

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn      *amqp.Connection
	queueName string
}
type Payload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (c *Consumer) setup() error {
	channel, err := c.conn.Channel()
	if err != nil {
		return err
	}

	return declareExchange(channel)
}

func (consumer *Consumer) Listen(topics []string) error {
	ch, err := consumer.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := declareRandomQueue(ch)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		if err := ch.QueueBind(
			q.Name,      // queue name
			topic,       // routing key
			"log_topic", // exchange
			false,       // no-wait
			nil,         // arguments
		); err != nil {
			return err
		}
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			var payload Payload
			err := json.Unmarshal(d.Body, &payload)
			if err != nil {
				log.Printf("Failed to unmarshal message: %s", err)
				continue
			}
			log.Printf("Received a message: %s", payload)
			go handlePayload(payload)
		}
	}()
	log.Printf(" Waiting for messages. [ Exchange: logs_topic, Topics: %s]", q.Name)
	<-forever

	return nil
}

func handlePayload(payload Payload) {
	switch payload.Name {
	case "log", "event":
		err := logEvent(payload)
		if err != nil {
			log.Printf("Failed to log event: %s", err)
		}
	case "auth":
		// err := authenticate(payload)
		// if err != nil {
		// 	log.Printf("Failed to authenticate: %s", err)
		// }

	default:
		err := logEvent(payload)
		if err != nil {
			log.Printf("Failed to log event: %s", err)
		}
	}
}

func logEvent(log Payload) error {
	// create some json we'll send to the log microservice
	jsonData, _ := json.MarshalIndent(log, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode != http.StatusAccepted {
		return err
	}

	return nil
}

func NewConsumer(conn *amqp.Connection) (Consumer, error) {
	consumer := Consumer{
		conn: conn,
	}

	err := consumer.setup()
	if err != nil {
		return Consumer{}, err
	}

	return consumer, nil

}

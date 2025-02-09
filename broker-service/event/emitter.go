package event

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Emitter struct {
	conn *amqp.Connection
}

func (e *Emitter) setup() error {
	channel, err := e.conn.Channel()
	if err != nil {
		return err
	}

	defer channel.Close()
	return declareExchange(channel)
}

func (e *Emitter) Emit(event string, severity string) error {
	channel, err := e.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	log.Println("Emitting event: ", event)
	return channel.Publish(
		"log_topic",
		severity,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(event),
		},
	)
}

func NewEmitter(conn *amqp.Connection) (Emitter, error) {
	emitter := Emitter{conn: conn}

	if err := emitter.setup(); err != nil {
		return Emitter{}, err
	}

	return emitter, nil
}

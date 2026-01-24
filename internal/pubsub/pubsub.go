package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	NoQueue SimpleQueueType = iota
	Durable
	Transient
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {

	v, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        v,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	defer ch.Close()

	var qType bool
	var autoDelete bool
	var exclusive bool

	switch queueType {
	case Durable:
		qType = true
		autoDelete = false
		exclusive = false
	case Transient:
		qType = false
		autoDelete = true
		exclusive = true
	}

	newQueue, err := ch.QueueDeclare(
		queueName,
		qType,
		autoDelete,
		exclusive,
		false,
		nil,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, newQueue, nil
}

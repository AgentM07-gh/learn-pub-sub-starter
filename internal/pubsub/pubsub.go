package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

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

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {
	connChannel, connQueue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	msgs, err := connChannel.Consume(connQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		defer connChannel.Close()
		var data T
		for d := range msgs {
			if err := json.Unmarshal(d.Body, &data); err != nil {
				fmt.Println(err)
				continue
			}
			handler(data)
			if err := d.Ack(false); err != nil {
				fmt.Println(err)
				continue
			}
		}
	}()

	return nil
}

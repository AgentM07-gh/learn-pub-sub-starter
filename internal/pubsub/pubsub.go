package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
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

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
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

	args := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}

	newQueue, err := ch.QueueDeclare(
		queueName,
		qType,
		autoDelete,
		exclusive,
		false,
		args,
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
	handler func(T) Acktype,
) error {
	connChannel, connQueue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	err = connChannel.Qos(
		10,    // prefetchCount
		0,     // prefetchSize
		false, // global
	)
	if err != nil {
		fmt.Println(err)
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
			result := handler(data)
			switch result {
			case Ack:
				d.Ack(false)
			case NackRequeue:
				d.Nack(false, true)
				fmt.Println("Nack:false, true")
			case NackDiscard:
				d.Nack(false, false)
				fmt.Println("Nack:false, false")
			}
		}
	}()

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
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
			ContentType: "application/gob",
			Body:        buffer.Bytes(),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	connChannel, connQueue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	err = connChannel.Qos(
		10,    // prefetchCount
		0,     // prefetchSize
		false, // global
	)
	if err != nil {
		fmt.Println(err)
	}

	msgs, err := connChannel.Consume(connQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		defer connChannel.Close()
		var t T
		for d := range msgs {
			buf := bytes.NewReader(d.Body)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(&t)
			if err != nil {
				fmt.Println(err)
			}
			result := handler(t)
			switch result {
			case Ack:
				d.Ack(false)
			case NackRequeue:
				d.Nack(false, true)
				fmt.Println("Nack:false, true")
			case NackDiscard:
				d.Nack(false, false)
				fmt.Println("Nack:false, false")
			}
		}
	}()

	return nil
}

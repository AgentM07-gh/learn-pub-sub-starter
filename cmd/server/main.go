package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	connectString := "amqp://guest:guest@localhost:5672/"
	amqpConnection, err := amqp.Dial(connectString)
	if err != nil {
		fmt.Println("Error creating connection:", err)
	}
	defer amqpConnection.Close()
	fmt.Println("amqp connection successful")

	gamelogic.PrintServerHelp()

	amqpCh, err := amqpConnection.Channel()
	if err != nil {
		fmt.Println("Error creating channel:", err)
	}
	defer amqpCh.Close()

	/*
		_, _, err = pubsub.DeclareAndBind(amqpConnection, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", pubsub.Durable)
		if err != nil {
			fmt.Println("Error Binding Queue:", err)
		}
	*/

	err = pubsub.SubscribeGob(
		amqpConnection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.Durable,
		handlerGameLogs,
	)

	for {
		input := gamelogic.GetInput()
		if len(input) != 0 {
			switch input[0] {
			case "pause":
				fmt.Println("Sending 'pause' message.")
				err = pubsub.PublishJSON(
					amqpCh,
					routing.ExchangePerilDirect,
					routing.PauseKey,
					routing.PlayingState{
						IsPaused: true,
					},
				)
			case "resume":
				fmt.Println("Sending 'resume' message.")
				err = pubsub.PublishJSON(
					amqpCh,
					routing.ExchangePerilDirect,
					routing.PauseKey,
					routing.PlayingState{
						IsPaused: false,
					},
				)
			case "quit":
				fmt.Println("Exiting...")
				return
			default:
				fmt.Println("Unknown Command")
			}

		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	fmt.Println("Server is running... Press Ctrl+C to stop")
	sig := <-signalChan
	fmt.Printf("\nReceived signal: %v\n", sig)
	fmt.Println("Shutting down")
}

func handlerGameLogs(gl routing.GameLog) pubsub.Acktype {
	defer fmt.Print("> ")
	gamelogic.WriteLog(gl)
	return pubsub.Ack
}

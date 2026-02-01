package main

import (
	"fmt"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {

	fmt.Println("Starting Peril client...")
	connectString := "amqp://guest:guest@localhost:5672/"

	amqpConnection, err := amqp.Dial(connectString)
	if err != nil {
		fmt.Println("Error creating connection:", err)
	}
	defer amqpConnection.Close()
	fmt.Println("amqp connection successful")

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println("Error in Client Welcome:", err)
	}

	amqpCh, err := amqpConnection.Channel()
	if err != nil {
		fmt.Println("Error creating channel:", err)
	}
	defer amqpCh.Close()

	queueName := routing.PauseKey + "." + userName

	gameState := gamelogic.NewGameState(userName)

	err = pubsub.SubscribeJSON(
		amqpConnection,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gameState),
	)
	if err != nil {
		fmt.Println("Error in subscribing:", err)
		return
	}

	armyQueueName := routing.ArmyMovesPrefix + "." + userName
	armyRoutingKey := routing.ArmyMovesPrefix + ".*"

	err = pubsub.SubscribeJSON(
		amqpConnection,
		routing.ExchangePerilTopic,
		armyQueueName,
		armyRoutingKey,
		pubsub.Transient,
		handlerMove(gameState),
	)
	if err != nil {
		fmt.Println("Error in subscribing:", err)
		return
	}

	for {
		input := gamelogic.GetInput()
		if len(input) != 0 {
			switch input[0] {
			case "spawn":
				err = gameState.CommandSpawn(input)
				if err != nil {
					fmt.Println("Error:", err)
				}
			case "move":
				mv, err := gameState.CommandMove(input)
				if err != nil {
					fmt.Println("Error:", err)
				}
				fmt.Println("Sending 'move' message.")
				err = pubsub.PublishJSON(
					amqpCh,
					routing.ExchangePerilTopic,
					armyQueueName,
					mv,
				)
				fmt.Println("Move Published Successfully")
			case "status":
				gameState.CommandStatus()
			case "help":
				gamelogic.PrintClientHelp()
			case "spam":
				fmt.Println("Spamming not allowed yet!")
			case "quit":
				gamelogic.PrintQuit()
				return
			default:
				fmt.Println("Error: Unknown Command")
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

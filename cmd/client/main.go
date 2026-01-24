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

	queueName := routing.PauseKey + "." + userName

	_, _, err = pubsub.DeclareAndBind(amqpConnection, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.Transient)
	if err != nil {
		fmt.Println("Error Binding Queue:", err)
	}

	gameState := gamelogic.NewGameState(userName)

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
				_, err = gameState.CommandMove(input)
				if err != nil {
					fmt.Println("Error:", err)
				}
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

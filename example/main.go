package main

import (
	"example/container"
	"fmt"
)

func main() {
	httpC := container.NewHttpContainer()
	fmt.Printf("HTTP app: %+v\n", httpC.App())

	asyncC := container.NewAsyncContainer()
	fmt.Printf("Async app: %+v\n", asyncC.App())

	wsC := container.NewWebsocketContainer()
	fmt.Printf("WebSocket app: %+v\n", wsC.App())
}

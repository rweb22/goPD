package main

import (
	//"encoding/json"
	"fmt"
	//"log"
	//"io"
	//"net/http"
	//"sort"
	//"strconv"
	"os"
	"os/signal"
	//"sync"
	"syscall"
	"time"
)

func main() {
	//var wg sync.WaitGroup
	stopChan := make(chan os.Signal, 1)

	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	server := StartServer()

	delay := 1 * time.Second

	Loop:
	for {
		select {
		case <-stopChan:
			fmt.Println("Server shutdown initiated...")
			server.Close()
			fmt.Println("Server shutdown completed.")
			break Loop
		case <-time.After(delay):
			nextOpen, err := StartCollector(stopChan)
			if err != nil {
				fmt.Println("Server shutdown initiated...")
				server.Close()
				fmt.Println("Server shutdown completed.")
				break Loop
			} else {
				fmt.Println("Waiting until:", nextOpen)
				delay = time.Until(nextOpen)
			}
		}
	}

	fmt.Println("Shutting Down...")
}


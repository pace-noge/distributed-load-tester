// main.go
// Entry point for the distributed load tester application
// Test comment added by Copilot to verify editing functionality
package main

import (
	"log"
	"os"

	"github.com/pace-noge/distributed-load-tester/cmd"
)

func main() {
	app := cmd.NewRootApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

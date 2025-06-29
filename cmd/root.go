package cmd

import (
	"fmt"
	"log"
	"net"

	"github.com/urfave/cli/v2"
)

// NewRootApp creates the root CLI application
func NewRootApp() *cli.App {
	return &cli.App{
		Name:  "load-tester-app",
		Usage: "A distributed load testing application (master or worker).",
		Commands: []*cli.Command{
			NewMasterCommand(),
			NewWorkerCommand(),
		},
	}
}

// GetWorkerAddress attempts to get the worker's reachable IP address and port.
// This is a helper function that can be used by both master and worker commands.
func GetWorkerAddress(workerGRPCPort int) string {
	conn, err := net.Dial("udp", "8.8.8.8:80") // Connect to a public DNS server to get local IP
	if err != nil {
		log.Printf("Warning: Could not determine local IP: %v. Using localhost.", err)
		return fmt.Sprintf("localhost:%d", workerGRPCPort) // Default worker gRPC port
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return fmt.Sprintf("%s:%d", localAddr.IP.String(), workerGRPCPort) // Use worker's gRPC port
}

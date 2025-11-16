package main

import (
	"amattu2/blink-middleware/pkg/liveview"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	region := flag.String("region", "", "Blink account region (e.g., u011)")
	apiToken := flag.String("token", "", "Blink API token")
	deviceType := flag.String("device-type", "", "Device type (camera, owl, hawk, doorbell, lotus)")
	accountId := flag.Int("account-id", 0, "Blink account ID")
	networkId := flag.Int("network-id", 0, "Network ID")
	cameraId := flag.Int("camera-id", 0, "Camera ID")

	flag.Parse()

	// Validate required flags
	if *region == "" || *apiToken == "" || *accountId == 0 || *networkId == 0 || *cameraId == 0 {
		log.Fatal("Error: --region, --token, --account-id, --network-id, and --camera-id are required")
	}

	// Initialize the client
	client := liveview.NewClient(
		*region,
		*apiToken,
		*deviceType,
		*accountId,
		*networkId,
		*cameraId,
	)

	ffplayCmd := exec.Command("ffplay",
		"-f", "mpegts",
		"-err_detect", "ignore_err",
		"-window_title", "Blink Liveview Middleware",
		"-",
	)
	inputPipe, err := ffplayCmd.StdinPipe()
	if err != nil {
		log.Println("error creating ffplay stdin pipe", err)
	}
	defer inputPipe.Close()

	if err := ffplayCmd.Start(); err != nil {
		log.Println("error starting ffplay", err)
	}
	defer ffplayCmd.Process.Kill()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received...")
		if err := client.Disconnect(); err != nil {
			log.Printf("Error disconnecting: %v", err)
		}
		os.Exit(0)
	}()

	// Connect to the livestream
	if err := client.Connect(inputPipe); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}

	select {}
}

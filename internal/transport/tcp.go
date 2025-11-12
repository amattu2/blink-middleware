package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"
)

type StreamConfig struct {
	// The output writer for the stream
	Writer io.Writer
	// The cancelable context for managing the stream lifecycle
	Ctx context.Context
	// Read timeout duration for the initial TCP connection
	ReadTimeout time.Duration
	// Interval for sending keep-alive pings
	PingInterval time.Duration
	// Callback for handling ping actions, if necessary
	OnPing func(*tls.Conn) error
	// Callback for handling actions upon successful connection
	OnConnect func(*tls.Conn) error
	// Error callback for handling stream-level errors
	OnError func(error)
	// Log callback for handling stream-level logs
	OnLog func(string)
}

// Stream connects to the liveview server using a TCP connection.
// Returns an error if the connection fails or if the stream ends unexpectedly.
//
// streamConfig: configuration for the stream connection
//
// host: the server hostname
//
// port: the server port
//
// Example: Stream(config, "0.0.0.0", "443") = nil
func Stream(config StreamConfig, host string, port string) error {
	config.OnLog(fmt.Sprintf("Connecting to %s:%s", host, port))

	client, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
		Certificates:       []tls.Certificate{},
	})
	if err != nil {
		return fmt.Errorf("unable to initialize stream: %w", err)
	} else {
		config.OnLog(fmt.Sprintf("Connected to %s", client.RemoteAddr()))
	}
	defer client.Close()
	defer config.OnLog(fmt.Sprintf("Disconnected from %s", client.RemoteAddr()))

	start := time.Now()
	if err := config.OnConnect(client); err != nil {
		return fmt.Errorf("error on connect: %w", err)
	}

	buf := make([]byte, 64)
	var streamErr error
	var readTimeout = config.ReadTimeout
stream:
	for {
		select {
		case <-config.Ctx.Done():
			config.OnLog("Closing TCP stream")
			break stream
		default:
			if err := client.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
				streamErr = fmt.Errorf("error setting read deadline: %w", err)
				break stream
			}

			n, err := client.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					streamErr = fmt.Errorf("connection closed gracefully by peer: %w", err)
				} else if errors.Is(err, syscall.ECONNRESET) {
					streamErr = fmt.Errorf("connection reset by peer: %w", err)
				} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					streamErr = fmt.Errorf("read timeout: %w", err)
				} else {
					streamErr = fmt.Errorf("error reading from server: %w", err)
				}
				break stream
			}

			if _, err := config.Writer.Write(buf[:n]); err != nil {
				streamErr = fmt.Errorf("error writing to writer: %w", err)
				break stream
			}

			// Send a keep-alive ping to the server
			if time.Since(start) > config.PingInterval {
				if err := config.OnPing(client); err != nil {
					streamErr = fmt.Errorf("error sending keep-alive: %w", err)
					break stream
				}

				// Reset the timer
				start = time.Now()
			}

			// After the initial connection, reduce the read timeout tolerance
			readTimeout = 2 * time.Second
		}
	}

	return streamErr
}

package liveview

import (
	blinkAdapter "amattu2/blink-middleware/internal/adapters/blink"
	blinkProtocol "amattu2/blink-middleware/internal/protocol/blink"
	"amattu2/blink-middleware/internal/transport"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"time"
)

type Client struct {
	// Credentials for connecting to the client service
	credentials blinkAdapter.ClientCredentials
	// Configuration options for the client
	config ClientConfig
	// Internal state of the client
	state clientState
}

type ClientConfig struct {
	// Initial connection read timeout duration
	ConnectTimeout time.Duration
	// Callback for handling stream-level errors
	OnError func(error)
	// Callback for logging messages
	OnLog func(string)
}

type clientState struct {
	// Whether the client is currently connected
	connected bool
	// The Blink command ID for the live view request
	lvCommandId int
	// Context for managing the stream lifecycle
	streamContext context.Context
	// Cancel function for the stream context
	streamCancel context.CancelFunc
}

// NewClient initializes a new Client instance with the provided details.
func NewClient(region string, apiToken string, deviceType string, accountId int, networkId int, cameraId int) *Client {
	return &Client{
		credentials: blinkAdapter.ClientCredentials{
			Region:     region,
			ApiToken:   apiToken,
			DeviceType: deviceType,
			AccountId:  accountId,
			NetworkId:  networkId,
			CameraId:   cameraId,
		},
		config: ClientConfig{
			ConnectTimeout: 15 * time.Second,
			OnError: func(err error) {
				// TODO: Make configurable
				log.Println(err)
			},
			OnLog: func(msg string) {
				// TODO: Make configurable
				log.Println(msg)
			},
		},
		state: clientState{
			connected:     false,
			lvCommandId:   0,
			streamContext: nil,
			streamCancel:  nil,
		},
	}
}

// Connect establishes a connection to the livestream.
//
// writer: the pipe to write the stream data to. This will not be closed by the function.
//
// Example: Connect(writer) = nil
func (c *Client) Connect(writer io.Writer) error {
	if c.state.connected {
		return fmt.Errorf("error during connect: client is already connected")
	}

	resp, err := blinkAdapter.InitiateLiveView(c.credentials)
	if err != nil {
		return fmt.Errorf("error during connect: %w", err)
	}

	c.state.streamContext, c.state.streamCancel = context.WithCancel(context.Background())
	c.state.lvCommandId = resp.CommandId
	c.state.connected = true
	go blinkAdapter.PollCommand(c.state.streamContext, c.credentials, resp.CommandId, resp.PollingInterval)

	// Get the connection details
	host, port, clientId, connId, err := blinkAdapter.ParseConnectionString(resp.Server)
	if err != nil {
		return fmt.Errorf("error during connect: parsing connection string: %w", err)
	}

	streamConfig := transport.StreamConfig{
		Writer:       writer,
		Ctx:          c.state.streamContext,
		ReadTimeout:  c.config.ConnectTimeout,
		PingInterval: 1 * time.Second,
		OnPing:       blinkProtocol.SendPing,
		OnConnect: func(conn *tls.Conn) error {
			return blinkProtocol.SendAuthFrames(conn, connId, clientId)
		},
		OnError: c.config.OnError,
		OnLog:   c.config.OnLog,
	}

	// Connect to the TCP server
	go func() {
		if err := transport.Stream(streamConfig, host, port); err != nil {
			c.config.OnError(fmt.Errorf("stream error: %w", err))
		}

		// Force disconnect on stream end if not directly cancelled
		c.Disconnect()
	}()

	return nil
}

// Disconnect terminates the connection to the livestream.
func (c *Client) Disconnect() error {
	if !c.state.connected {
		return nil
	}

	c.state.streamCancel()
	c.state.connected = false

	if err := blinkAdapter.StopCommand(c.credentials, c.state.lvCommandId); err != nil {
		log.Printf("Error stopping command: %v", err)
	}

	c.state.streamContext = nil
	c.state.streamCancel = nil
	c.state.lvCommandId = 0

	return nil
}

// IsConnected returns whether the client is currently connected to the livestream.
func (c *Client) IsConnected() bool {
	return c.state.connected
}

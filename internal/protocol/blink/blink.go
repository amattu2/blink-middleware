package blink

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"time"
)

// FRAMES_KEEPALIVE is the keep-alive ping frame sent to the Blink stream server.
var FRAMES_KEEPALIVE = []byte{
	0x12, 0x00, 0x00, 0x03, 0xe8, 0x00, 0x00, 0x00,
	0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x00,
}

// GenerateAuthFrames returns the header payload for the TCP connection
//
// connectionId: the connection ID to use in the header
//
// clientId: the client ID to use in the header
//
// Example: GenerateAuthFrames("connection-id", 123)
func GenerateAuthFrames(connectionId string, clientId int) [][]byte {
	// Frame 1 (unknown)
	frame1 := []byte{
		0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	// Frame 2 (Client ID)
	clientIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(clientIDBytes, uint32(clientId))
	frame2 := []byte{
		clientIDBytes[0], clientIDBytes[1], clientIDBytes[2], clientIDBytes[3],
	}

	// Frame 3 (unknown)
	frame3 := []byte{
		0x01, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x10,
	}

	// Frame 4 (Connection ID)
	frame4 := []byte(connectionId)

	// Frame 5 (unknown)
	frame5 := []byte{
		0x00, 0x00, 0x00, 0x01, 0x0a, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
	}

	return [][]byte{
		frame1,
		frame2,
		frame3,
		frame4,
		frame5,
	}
}

// SendAuthFrames sends the authentication frames to the server.
//
// client: the TCP client connection to send the frames on
//
// connectionId: the Blink connection ID to use in the header
//
// clientId: the Blink client ID to use in the header
//
// Example: SendAuthFrames(client, "connection-id", 123) = nil
func SendAuthFrames(client *tls.Conn, connectionId string, clientId int) error {
	if err := client.SetWriteDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return fmt.Errorf("error setting write deadline: %w", err)
	}

	frames := GenerateAuthFrames(connectionId, clientId)
	for _, frame := range frames {
		if _, err := client.Write(frame); err != nil {
			return fmt.Errorf("error sending connection header: %w", err)
		}
	}

	return nil
}

// SendPing sends a keep-alive ping to the server.
//
// client: the client connection to send the ping on
//
// Example: SendPing(client) = nil
func SendPing(client *tls.Conn) (err error) {
	if err := client.SetWriteDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return fmt.Errorf("error setting write deadline: %w", err)
	}

	if _, err := client.Write(FRAMES_KEEPALIVE); err != nil {
		return fmt.Errorf("error sending keep-alive: %w", err)
	}

	return nil
}

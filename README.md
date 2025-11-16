# Introduction

This project provides a developer interface for initiating a livestream (live view)
for a Blink Smart Security camera.

# Usage

## Installation

```sh
go get github.com/amattu2/blink-middleware
```

## API Usage

### Creating a Client

Initialize a new Blink client using [`liveview.NewClient`](pkg/liveview/liveview.go):

```go
import "amattu2/blink-middleware/pkg/liveview"

client := liveview.NewClient(
    "u011",           // region
    "your-api-token", // apiToken
    "owl",            // deviceType
    12345,            // accountId
    67890,            // networkId
    11111,            // cameraId
)
```

### Connecting to the Livestream

Connect to the livestream by providing an `io.Writer` to receive the raw stream data:

```go
import "os"

if err := client.Connect(os.Stdout); err != nil {
    // The livestream errored out or did not connect
}
```

The [`Connect`](pkg/liveview/liveview.go) method establishes the connection
and begins streaming video data to the provided writer. The stream will continue
until explicitly disconnected or an error occurs.

### Disconnecting

Gracefully terminate the livestream connection:

```go
if err := client.Disconnect(); err != nil {
    // The connection did not gracefully terminate
}
```

The [`Disconnect`](pkg/liveview/liveview.go) method stops the stream,
closes the connection, and cleans up resources.

### Checking Connection Status

Check if the client is currently connected:

```go
if client.IsConnected() {
    log.Println("Client is connected")
}
```

# Dependencies

Aside from Go 1.23+, this project has no external dependencies.

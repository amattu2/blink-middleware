package blink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var BASE_URL = "https://rest-%s.immedia-semi.com"

type ClientCredentials struct {
	// Region to use for the API URL (e.g. "u011")
	Region string
	// Blink Authentication token to use for the API requests
	ApiToken string
	// Type of device to connect to (e.g. "owl")
	DeviceType string
	// The ID of the account that the camera belongs to
	AccountId int
	// The ID of the network that the camera is associated with
	NetworkId int
	// The ID of the camera to connect to
	CameraId int
}

// CreateLiveViewURI returns the live view path based on the device type
//
// cc: the client credentials to use for building the URL
//
// Example: CreateLiveViewURI(ClientCredentials{...}) = ".../api/v5/accounts/X/networks/X/cameras/X/liveview"
func CreateLiveViewURI(cc ClientCredentials) (string, error) {
	var path string
	switch cc.DeviceType {
	case "camera":
		path = "/api/v5/accounts/%d/networks/%d/cameras/%d/liveview"
	case "owl", "hawk":
		path = "/api/v2/accounts/%d/networks/%d/owls/%d/liveview"
	case "doorbell", "lotus":
		path = "/api/v2/accounts/%d/networks/%d/doorbells/%d/liveview"
	}

	if path != "" {
		return fmt.Sprintf(BASE_URL+path, cc.Region, cc.AccountId, cc.NetworkId, cc.CameraId), nil
	}

	return "", fmt.Errorf("cannot build path for unknown device type: %s", cc.DeviceType)
}

// CreatePollingURI returns the polling URL for the given command ID
//
// cc: the client credentials to use for building the URL
//
// commandId: the command ID to poll
//
// Example: CreatePollingURI(ClientCredentials{...}, 123) = ".../api/v5/networks/%d/command/%d"
func CreatePollingURI(cc ClientCredentials, commandId int) (string, error) {
	return fmt.Sprintf(BASE_URL+"/network/%d/command/%d", cc.Region, cc.NetworkId, commandId), nil
}

// ParseConnectionString parses the connection string to extract the connection details
//
// url: the connection string to parse
//
// Example: ParseConnectionString("TODO")
func ParseConnectionString(server string) (string, string, int, string, error) {
	parsedUrl, err := url.Parse(server)
	if err != nil {
		return "", "", 0, "", err
	}

	if parsedUrl.Hostname() == "" {
		return "", "", 0, "", fmt.Errorf("invalid host")
	}

	if parsedUrl.Port() != "443" {
		return "", "", 0, "", fmt.Errorf("unexpected port %s. Expecting 443", parsedUrl.Port())
	}

	pathSegments := strings.Split(parsedUrl.Path, "/")
	if len(pathSegments) == 0 {
		return "", "", 0, "", fmt.Errorf("invalid path")
	}

	connID := strings.Split(pathSegments[len(pathSegments)-1], "_")
	if len(connID) < 2 || connID[0] == "" {
		return "", "", 0, "", fmt.Errorf("invalid connection ID")
	}

	clientID, err := strconv.Atoi(parsedUrl.Query().Get("client_id"))
	if clientID == 0 || err != nil {
		return "", "", 0, "", fmt.Errorf("invalid client ID")
	}

	return parsedUrl.Hostname(), parsedUrl.Port(), clientID, connID[0], nil
}

// SetRequestHeaders appends the required headers to the request
//
// req: the request to append headers to
//
// token: the token to use for the request
//
// Example: SetRequestHeaders(req, "bearer-token-here")
func SetRequestHeaders(req *http.Request, token string) {
	req.Header.Set("locale", "en_US")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("content-type", "application/json; charset=UTF-8")
}

type CommandResponse struct {
	Code       int    `json:"code"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Complete   bool   `json:"complete"`
}

// PollCommand will repeatedly poll the command URL with the provided token
//
// ctx: the context to use for the command
//
// cc: the client credentials to use for building the URL
//
// commandId: the command ID to poll
//
// pollInterval: the interval (in seconds) to poll the command at
//
// Example: PollCommand(ctx, ClientCredentials{...}, 123, 5) = nil
func PollCommand(ctx context.Context, cc ClientCredentials, commandId int, pollInterval int) error {
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()

	url, err := CreatePollingURI(cc, commandId)
	if err != nil {
		return fmt.Errorf("error creating polling URL: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}

			SetRequestHeaders(req, cc.ApiToken)

			client := &http.Client{Timeout: time.Second * 10}
			resp, err := client.Do(req)
			if resp.StatusCode != http.StatusOK || err != nil {
				return fmt.Errorf("error polling command. HTTP Status Code %d", resp.StatusCode)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			result := CommandResponse{}
			if err != nil {
				return err
			}

			err = json.Unmarshal(body, &result)
			if err != nil {
				return err
			}

			if result.Complete {
				return fmt.Errorf("command marked as complete. Cannot poll further")
			}
		}
	}
}

type LiveviewInput struct {
	Intent string `json:"intent"`
}

type LiveviewResponse struct {
	CommandId       int    `json:"command_id"`
	PollingInterval int    `json:"polling_interval"`
	Server          string `json:"server"`
}

// InitiateLiveView starts the liveview intention for the camera
//
// Example: InitiateLiveView(ClientCredentials{...}) = TODO
func InitiateLiveView(cc ClientCredentials) (*LiveviewResponse, error) {
	url, err := CreateLiveViewURI(cc)
	if err != nil {
		return nil, fmt.Errorf("error getting liveview path: %w", err)
	}

	jsonBody, _ := json.Marshal(&LiveviewInput{
		Intent: "liveview",
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	SetRequestHeaders(req, cc.ApiToken)

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || err != nil {
		return nil, fmt.Errorf("error from API. HTTP Status Code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result LiveviewResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	} else if resp == nil || result.CommandId == 0 {
		return nil, fmt.Errorf("error sending liveview command: %v", resp)
	}

	return &result, nil
}

// StopCommand marks the command (liveview) as completed
//
// cc: the client credentials to use for building the URL
//
// commandId: the command ID to stop
//
// Example: StopCommand(ClientCredentials{...}, 123)
func StopCommand(cc ClientCredentials, commandId int) error {
	url, err := CreatePollingURI(cc, commandId)
	if err != nil {
		return fmt.Errorf("error creating polling URL: %w", err)
	}

	req, err := http.NewRequest("POST", url+"/done", nil)
	if err != nil {
		return err
	}

	SetRequestHeaders(req, cc.ApiToken)

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK || err != nil {
		return fmt.Errorf("cannot stop command. HTTP Status Code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result CommandResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	if result.Code != 902 {
		return fmt.Errorf("cannot stop command. API Code %d with message %s", result.Code, result.Message)
	}

	return nil
}

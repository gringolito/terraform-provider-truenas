package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/truenas/api_client_golang/truenas_api"
)

type Client struct {
	api            *truenas_api.Client
	timeoutSeconds int64
	mutex          sync.Mutex
}

func NewClient(serverAddress string, protocol string, verifySSL bool, apiKey string) (*Client, error) {
	apiURL := protocol + serverAddress + "/api/current"
	client, err := truenas_api.NewClient(apiURL, verifySSL)
	if err != nil {
		return nil, ClientError{err}
	}

	if err := client.Login("", "", apiKey); err != nil {
		return nil, AuthenticationError{err}
	}

	if _, err := client.Ping(); err != nil {
		return nil, APIError{"API Ping failed", err}
	}

	return &Client{api: client, timeoutSeconds: 10}, nil
}

func (c *Client) Call(method string, params any) ([]byte, error) {
	c.mutex.Lock()
	res, err := c.api.Call(method, c.timeoutSeconds, params)
	c.mutex.Unlock()
	if err != nil {
		return nil, APIError{"API call failed", err}
	}

	var response map[string]json.RawMessage
	if err := json.Unmarshal(res, &response); err != nil {
		return nil, APIError{"Failed to parse API response", err}
	}

	// Check if there's an error in the response
	if errorData, exists := response["error"]; exists {
		var errorMsg errorMessage
		json.Unmarshal(errorData, &errorMsg)
		var errorFull map[string]any
		json.Unmarshal(errorData, &errorFull)
		log.Printf("[terraform-provider-truenas] client.Client.Call() called with: %q, %v", method, params)
		log.Printf("[terraform-provider-truenas] client.Client.Call() returned: %v", errorFull)
		return nil, fmt.Errorf(
			"API method %s responded with an error: %s (code: %d)", method, errorMsg.Message, errorMsg.Code,
		)
	}

	// Return the result operation if exists
	if result, exists := response["result"]; exists {
		return result, nil
	}

	return nil, errors.New("unexpected API response format")
}

func (c *Client) toAPIParams(v any) (map[string]any, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var params map[string]any
	err = json.Unmarshal(jsonData, &params)
	return params, err
}

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	truenas_api "github.com/truenas/api_client_golang/truenas_api"
)

type rawCaller interface {
	Call(method string, timeoutSeconds int64, params interface{}) (json.RawMessage, error)
	Login(username, password, apiKey string) error
	Close() error
}

type WebSocketClient struct {
	inner rawCaller
	mu    sync.Mutex
}

func NewWebSocketClient(host, apiKey, username, password string, insecureSkipVerify bool) (*WebSocketClient, error) {
	url := fmt.Sprintf("wss://%s/api/current", host)
	c, err := truenas_api.NewClient(url, !insecureSkipVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to truenas: %w", err)
	}
	if err := c.Login(username, password, apiKey); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}
	return &WebSocketClient{inner: c}, nil
}

func (c *WebSocketClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	timeoutSeconds := int64(30)
	if deadline, ok := ctx.Deadline(); ok {
		secs := int64(time.Until(deadline).Seconds())
		if secs > 0 {
			timeoutSeconds = secs
		}
	}

	c.mu.Lock()
	raw, err := c.inner.Call(method, timeoutSeconds, params)
	c.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return unwrapResult(raw)
}

func (c *WebSocketClient) CallWithJob(_ context.Context, _ string, _ any) (json.RawMessage, error) {
	return nil, errors.New("CallWithJob not implemented")
}

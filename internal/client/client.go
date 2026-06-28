package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	truenas_api "github.com/truenas/api_client_golang/truenas_api"
)

type rawCaller interface {
	Call(method string, timeoutSeconds int64, params any) (json.RawMessage, error)
	Login(username, password, apiKey string) error
	Close() error
}

type WebSocketClient struct {
	inner rawCaller
	mu    sync.Mutex
}

const (
	loginMaxRetries = 5
	loginBaseDelay  = 2 * time.Second
)

func NewWebSocketClient(host, apiKey, username, password string, insecureSkipVerify bool) (*WebSocketClient, error) {
	url := fmt.Sprintf("wss://%s/api/current", host)
	c, err := truenas_api.NewClient(url, !insecureSkipVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to truenas: %w", err)
	}

	var loginErr error
	for attempt := range loginMaxRetries {
		loginErr = c.Login(username, password, apiKey)
		if loginErr == nil {
			return &WebSocketClient{inner: c}, nil
		}
		if !isTransientLoginError(loginErr) || attempt == loginMaxRetries-1 {
			break
		}
		// Exponential backoff: 2s, 4s, 8s, 16s
		time.Sleep(loginBaseDelay * (1 << attempt))
	}

	c.Close()
	return nil, fmt.Errorf("failed to authenticate: %w", loginErr)
}

// isTransientLoginError reports whether err is a transient failure worth
// retrying — a TrueNAS EBUSY rate-limit or a call timeout. The upstream SDK
// returns plain string errors, so detection is done by substring.
func isTransientLoginError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "EBUSY") || strings.Contains(msg, "timed out")
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

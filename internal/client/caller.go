package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type Caller interface {
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)
	CallWithJob(ctx context.Context, method string, params any) (json.RawMessage, error)
}

type APIError struct {
	ErrName string
	Type    string
	Reason  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("truenas api error: %s (%s): %s", e.ErrName, e.Type, e.Reason)
}

type envelope struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

type apiErrorPayload struct {
	ErrName string `json:"errname"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
}

func unwrapResult(raw json.RawMessage) (json.RawMessage, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("failed to parse response envelope: %w", err)
	}

	if len(env.Error) > 0 && string(env.Error) != "null" {
		var payload apiErrorPayload
		if err := json.Unmarshal(env.Error, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse api error: %w", err)
		}
		return nil, &APIError{
			ErrName: payload.ErrName,
			Type:    payload.Type,
			Reason:  payload.Reason,
		}
	}

	return env.Result, nil
}

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
	// Code and Message hold the alternative RPC/validation error shape
	// ({"code":…,"message":…}) returned by some TrueNAS endpoints.
	Code    int
	Message string
}

func (e *APIError) Error() string {
	if e.ErrName != "" {
		return fmt.Sprintf("truenas api error: %s (%s): %s", e.ErrName, e.Type, e.Reason)
	}
	return fmt.Sprintf("truenas api error (code %d): %s", e.Code, e.Message)
}

// IsNotFound reports whether the error indicates the resource does not exist.
// TrueNAS encodes "not found" as errname="MatchNotFound" in older middleware
// and as code=-32602 in newer RPC format.
func (e *APIError) IsNotFound() bool {
	return e.ErrName == "MatchNotFound" || e.Code == -32602
}

type envelope struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// apiErrorPayload covers the two error shapes TrueNAS may return:
//
//	{"errname":"…","type":"…","reason":"…"}   — middleware/CallError shape
//	{"code":…,"message":"…"}                  — RPC/validation shape
type apiErrorPayload struct {
	ErrName string `json:"errname"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Code    int    `json:"code"`
	Message string `json:"message"`
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
			Code:    payload.Code,
			Message: payload.Message,
		}
	}

	return env.Result, nil
}

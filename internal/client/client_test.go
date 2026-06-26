package client

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeRawCaller is a test double for rawCaller.
type fakeRawCaller struct {
	mu       sync.Mutex
	response json.RawMessage
	err      error
	delay    time.Duration
	calls    []string
}

func (f *fakeRawCaller) Call(method string, _ int64, _ interface{}) (json.RawMessage, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	f.mu.Lock()
	f.calls = append(f.calls, method)
	f.mu.Unlock()
	return f.response, f.err
}

func (f *fakeRawCaller) Login(_, _, _ string) error { return nil }
func (f *fakeRawCaller) Close() error               { return nil }

func makeEnvelope(result, apiErr string) json.RawMessage {
	r := "null"
	if result != "" {
		r = result
	}
	e := "null"
	if apiErr != "" {
		e = apiErr
	}
	return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":` + r + `,"error":` + e + `}`)
}

func TestUnwrapResult_Success(t *testing.T) {
	raw := makeEnvelope(`{"id":42}`, "")
	result, err := unwrapResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if got["id"] != float64(42) {
		t.Errorf("expected id=42, got %v", got["id"])
	}
}

func TestUnwrapResult_APIError(t *testing.T) {
	apiErr := `{"errname":"ValidationError","type":"VALIDATION","reason":"invalid value"}`
	raw := makeEnvelope("", apiErr)
	_, err := unwrapResult(raw)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiError.ErrName != "ValidationError" {
		t.Errorf("ErrName: got %q, want %q", apiError.ErrName, "ValidationError")
	}
	if apiError.Type != "VALIDATION" {
		t.Errorf("Type: got %q, want %q", apiError.Type, "VALIDATION")
	}
	if apiError.Reason != "invalid value" {
		t.Errorf("Reason: got %q, want %q", apiError.Reason, "invalid value")
	}
}

func TestUnwrapResult_NullError(t *testing.T) {
	raw := makeEnvelope(`"pong"`, "")
	result, err := unwrapResult(raw)
	if err != nil {
		t.Fatalf("unexpected error for null error field: %v", err)
	}
	if string(result) != `"pong"` {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestWebSocketClient_Call_Success(t *testing.T) {
	fake := &fakeRawCaller{response: makeEnvelope(`{"name":"test"}`, "")}
	c := &WebSocketClient{inner: fake}

	result, err := c.Call(context.Background(), "user.query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if got["name"] != "test" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestWebSocketClient_Call_APIError(t *testing.T) {
	apiErr := `{"errname":"NotFound","type":"NOT_FOUND","reason":"object not found"}`
	fake := &fakeRawCaller{response: makeEnvelope("", apiErr)}
	c := &WebSocketClient{inner: fake}

	_, err := c.Call(context.Background(), "user.get_instance", []int{999})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiError.ErrName != "NotFound" {
		t.Errorf("ErrName: got %q, want %q", apiError.ErrName, "NotFound")
	}
}

func TestWebSocketClient_Call_TransportError(t *testing.T) {
	transportErr := errors.New("connection reset")
	fake := &fakeRawCaller{err: transportErr}
	c := &WebSocketClient{inner: fake}

	_, err := c.Call(context.Background(), "core.ping", nil)
	if !errors.Is(err, transportErr) {
		t.Errorf("expected transport error, got %v", err)
	}
}

func TestWebSocketClient_Call_ContextDeadline(t *testing.T) {
	fake := &fakeRawCaller{response: makeEnvelope(`true`, "")}
	c := &WebSocketClient{inner: fake}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Call(ctx, "core.ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebSocketClient_CallWithJob_NotImplemented(t *testing.T) {
	c := &WebSocketClient{inner: &fakeRawCaller{}}
	_, err := c.CallWithJob(context.Background(), "pool.dataset.create", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "CallWithJob not implemented" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestWebSocketClient_Call_Concurrent(t *testing.T) {
	fake := &fakeRawCaller{
		response: makeEnvelope(`"pong"`, ""),
		delay:    5 * time.Millisecond,
	}
	c := &WebSocketClient{inner: fake}

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			_, err := c.Call(context.Background(), "core.ping", nil)
			if err != nil {
				t.Errorf("unexpected error in concurrent call: %v", err)
			}
		}()
	}
	wg.Wait()

	fake.mu.Lock()
	callCount := len(fake.calls)
	fake.mu.Unlock()
	if callCount != n {
		t.Errorf("expected %d calls, got %d", n, callCount)
	}
}

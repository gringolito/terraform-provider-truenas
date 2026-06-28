package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
)

var wsUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type wsTestServer struct {
	*httptest.Server
	mu      sync.Mutex
	results map[string]json.RawMessage
	errors  map[string]json.RawMessage
	delay   time.Duration
}

func newWSTestServer(t *testing.T) *wsTestServer {
	t.Helper()
	ts := &wsTestServer{
		results: make(map[string]json.RawMessage),
		errors:  make(map[string]json.RawMessage),
	}
	ts.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/current" {
			http.NotFound(w, r)
			return
		}
		ts.serveWS(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func (ts *wsTestServer) setResult(method string, result json.RawMessage) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.results[method] = result
}

func (ts *wsTestServer) setError(method string, errPayload json.RawMessage) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.errors[method] = errPayload
}

func (ts *wsTestServer) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}

		ts.mu.Lock()
		delay := ts.delay
		errPayload := ts.errors[req.Method]
		result := ts.results[req.Method]
		ts.mu.Unlock()

		if delay > 0 {
			time.Sleep(delay)
		}

		var resp map[string]any
		if errPayload != nil {
			resp = map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": nil, "error": errPayload}
		} else {
			resp = map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}
		}

		writeMu.Lock()
		conn.WriteJSON(resp) //nolint:errcheck
		writeMu.Unlock()
	}
}

func (ts *wsTestServer) newClient(t *testing.T) *client.WebSocketClient {
	t.Helper()
	ts.setResult("auth.login_with_api_key", json.RawMessage("true"))
	c, err := client.NewWebSocketClient(ts.Listener.Addr().String(), "test-key", "", "", true)
	if err != nil {
		t.Fatalf("NewWebSocketClient: %v", err)
	}
	return c
}

func TestNewWebSocketClient_Success(t *testing.T) {
	srv := newWSTestServer(t)
	srv.setResult("auth.login_with_api_key", json.RawMessage("true"))

	_, err := client.NewWebSocketClient(srv.Listener.Addr().String(), "test-key", "", "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCall_SuccessResult(t *testing.T) {
	srv := newWSTestServer(t)
	c := srv.newClient(t)

	srv.setResult("user.query", json.RawMessage(`{"id":42}`))

	result, err := c.Call(context.Background(), "user.query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["id"] != float64(42) {
		t.Errorf("id: got %v, want 42", got["id"])
	}
}

func TestCall_APIError(t *testing.T) {
	srv := newWSTestServer(t)
	c := srv.newClient(t)

	srv.setError("user.create", json.RawMessage(`{"errname":"ValidationError","type":"VALIDATION","reason":"invalid value"}`))

	_, err := c.Call(context.Background(), "user.create", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError, got %T: %v", err, err)
	}
	if apiErr.ErrName != "ValidationError" {
		t.Errorf("ErrName: got %q, want %q", apiErr.ErrName, "ValidationError")
	}
	if apiErr.Type != "VALIDATION" {
		t.Errorf("Type: got %q, want %q", apiErr.Type, "VALIDATION")
	}
	if apiErr.Reason != "invalid value" {
		t.Errorf("Reason: got %q, want %q", apiErr.Reason, "invalid value")
	}
}

func TestCall_RPCCodeError(t *testing.T) {
	srv := newWSTestServer(t)
	c := srv.newClient(t)

	srv.setError("group.get_instance", json.RawMessage(`{"code":-32602,"message":"Invalid params"}`))

	_, err := c.Call(context.Background(), "group.get_instance", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError, got %T: %v", err, err)
	}
	if apiErr.Code != -32602 {
		t.Errorf("Code: got %d, want -32602", apiErr.Code)
	}
	if apiErr.Message != "Invalid params" {
		t.Errorf("Message: got %q, want %q", apiErr.Message, "Invalid params")
	}
	if !apiErr.IsNotFound() {
		t.Error("IsNotFound() should be true for code -32602")
	}
}

func TestCallWithJob_NotImplemented(t *testing.T) {
	srv := newWSTestServer(t)
	c := srv.newClient(t)

	_, err := c.CallWithJob(context.Background(), "pool.dataset.create", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "CallWithJob not implemented" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestCall_Concurrent(t *testing.T) {
	srv := newWSTestServer(t)
	srv.delay = 5 * time.Millisecond
	c := srv.newClient(t)

	srv.setResult("core.ping", json.RawMessage(`"pong"`))

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			if _, err := c.Call(context.Background(), "core.ping", nil); err != nil {
				t.Errorf("unexpected error in concurrent call: %v", err)
			}
		}()
	}
	wg.Wait()
}

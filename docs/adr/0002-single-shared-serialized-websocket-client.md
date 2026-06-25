# Single shared WebSocket client with serialized calls

The provider uses one persistent, authenticated `api_client_golang` client shared across
all resources, and **serializes every call** through a mutex (or single writer goroutine),
with reconnect-on-failure.

The client holds one gorilla/websocket connection. gorilla/websocket forbids concurrent
writers, but Terraform runs resources in parallel (`-parallelism=10` by default), and the
client's `Call` performs `WriteJSON` *outside* its mutex — so unsynchronized concurrent
calls would interleave frame writes and corrupt the connection. A TrueNAS provider is
low-volume, so serializing all middleware traffic costs little and buys correctness and
simplicity. A connection pool was rejected as unjustified complexity (per-connection auth,
job subscription, and reconnection) for the throughput involved.

## Consequences

- The typed client owns connection lifecycle: connect, login, the `listen()` demux, and
  reconnect. Resources never manage connections.
- If real concurrency is ever needed, revisit with a pool — but only with evidence that
  serialization is the bottleneck.

package activity

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

type mockTracker struct {
	connected    int
	disconnected int
	mu           sync.Mutex
}

func (m *mockTracker) PeerConnected()    { m.mu.Lock(); m.connected++; m.mu.Unlock() }
func (m *mockTracker) PeerDisconnected() { m.mu.Lock(); m.disconnected++; m.mu.Unlock() }

func TestAwaitActivityFile_Create_Socket(t *testing.T) {
	raw := json.RawMessage(`{"type":"socket","connection":"tcp","from":"127.0.0.1:9000","to":"127.0.0.1:9001"}`)
	file := &AwaitActivityFile{Type: "socket"}
	result := file.Create(raw)
	if result == nil {
		t.Fatal("expected non-nil AwaitActivity for socket type")
	}
	s, ok := result.(*SocketAwaitActivity)
	if !ok {
		t.Fatalf("expected *SocketAwaitActivity, got %T", result)
	}
	if s.Connection != "tcp" {
		t.Errorf("expected Connection=tcp, got %s", s.Connection)
	}
	if s.From != "127.0.0.1:9000" {
		t.Errorf("expected From=127.0.0.1:9000, got %s", s.From)
	}
	if s.To != "127.0.0.1:9001" {
		t.Errorf("expected To=127.0.0.1:9001, got %s", s.To)
	}
}

func TestAwaitActivityFile_Create_HTTP(t *testing.T) {
	raw := json.RawMessage(`{"type":"http","from":":8080","to":"http://localhost:3000","interval":10}`)
	file := &AwaitActivityFile{Type: "http"}
	result := file.Create(raw)
	if result == nil {
		t.Fatal("expected non-nil AwaitActivity for http type")
	}
	h, ok := result.(*HTTPAwaitActivity)
	if !ok {
		t.Fatalf("expected *HTTPAwaitActivity, got %T", result)
	}
	if h.From != ":8080" {
		t.Errorf("expected From=:8080, got %s", h.From)
	}
	if h.To != "http://localhost:3000" {
		t.Errorf("expected To=http://localhost:3000, got %s", h.To)
	}
	if h.Interval != 10 {
		t.Errorf("expected Interval=10, got %d", h.Interval)
	}
}

func TestAwaitActivityFile_Create_Unknown(t *testing.T) {
	raw := json.RawMessage(`{"type":"unknown"}`)
	file := &AwaitActivityFile{Type: "unknown"}
	result := file.Create(raw)
	if result != nil {
		t.Fatalf("expected nil for unknown type, got %T", result)
	}
}

func TestAwaitActivityFile_Create_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{invalid json!!!}`)
	file := &AwaitActivityFile{Type: "socket"}
	result := file.Create(raw)
	if result != nil {
		t.Fatalf("expected nil for invalid JSON, got %T", result)
	}

	file2 := &AwaitActivityFile{Type: "http"}
	result2 := file2.Create(raw)
	if result2 != nil {
		t.Fatalf("expected nil for invalid JSON (http), got %T", result2)
	}
}

func TestSocketAwaitActivity_GetActivityType(t *testing.T) {
	s := &SocketAwaitActivity{}
	if s.GetActivityType() != SocketActivity {
		t.Errorf("expected SocketActivity (%d), got %d", SocketActivity, s.GetActivityType())
	}
}

func TestHTTPAwaitActivity_GetActivityType(t *testing.T) {
	h := &HTTPAwaitActivity{}
	if h.GetActivityType() != HTTPActivity {
		t.Errorf("expected HTTPActivity (%d), got %d", HTTPActivity, h.GetActivityType())
	}
}

func TestDialWithTimeout_Failure(t *testing.T) {
	start := time.Now()
	_, err := dialWithTimeout("tcp", "198.51.100.1:1", 100*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error for unreachable address")
	}
	// Allow some slack for OS-level delays but it should finish reasonably fast.
	if elapsed > 10*time.Second {
		t.Errorf("dialWithTimeout took too long: %v", elapsed)
	}
}

func TestProxyConnections(t *testing.T) {
	tracker := &mockTracker{}

	clientLocal, clientRemote := net.Pipe()
	targetLocal, targetRemote := net.Pipe()

	// proxyConnections copies:
	//   client (clientRemote) -> target (targetRemote)
	//   target (targetRemote) -> client (clientRemote)
	// We write/read on clientLocal and targetLocal (the other ends).
	go proxyConnections(tracker, clientRemote, targetRemote)

	// Test client -> target direction.
	msg1 := []byte("hello from client")
	go func() {
		clientLocal.Write(msg1)
	}()
	buf := make([]byte, 128)
	n, err := targetLocal.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from target side: %v", err)
	}
	if string(buf[:n]) != string(msg1) {
		t.Errorf("expected %q, got %q", msg1, buf[:n])
	}

	// Test target -> client direction.
	msg2 := []byte("hello from target")
	go func() {
		targetLocal.Write(msg2)
	}()
	n, err = clientLocal.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from client side: %v", err)
	}
	if string(buf[:n]) != string(msg2) {
		t.Errorf("expected %q, got %q", msg2, buf[:n])
	}

	// Close one side to trigger cleanup.
	clientLocal.Close()
	targetLocal.Close()

	// Give goroutines time to finish cleanup.
	time.Sleep(100 * time.Millisecond)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if tracker.disconnected != 1 {
		t.Errorf("expected PeerDisconnected called exactly once, got %d", tracker.disconnected)
	}
}

func TestStartSocketListener_InvalidAddress(t *testing.T) {
	tracker := &mockTracker{}
	act := &SocketAwaitActivity{
		Connection: "tcp",
		From:       "invalid-address-no-port",
		To:         "127.0.0.1:9999",
	}

	// Should not panic; just logs and returns.
	done := make(chan struct{})
	go func() {
		defer close(done)
		StartSocketListener(context.Background(), tracker, act)
	}()

	select {
	case <-done:
		// Returned without panic - success.
	case <-time.After(2 * time.Second):
		t.Fatal("StartSocketListener did not return for invalid address")
	}
}

func TestStartSocketListener_ContextCancel(t *testing.T) {
	tracker := &mockTracker{}

	// Pick a random available port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	act := &SocketAwaitActivity{
		Connection: "tcp",
		From:       addr,
		To:         "127.0.0.1:1", // doesn't matter, we cancel before connecting
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		StartSocketListener(ctx, tracker, act)
	}()

	// Give the listener time to start and block on Accept().
	time.Sleep(100 * time.Millisecond)

	// Cancel context. The goroutine in StartSocketListener closes the
	// listener, which unblocks Accept() and the ctx.Done() check exits.
	cancel()

	select {
	case <-done:
		// Stopped - success.
	case <-time.After(5 * time.Second):
		t.Fatal("StartSocketListener did not stop after context cancel")
	}

	// Suppress unused import warning.
	_ = io.Discard
}

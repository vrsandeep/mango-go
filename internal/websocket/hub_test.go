package websocket

import (
	"testing"
	"time"
)

func TestHub(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Mock client
	client := &Client{
		hub:  hub,
		send: make(chan []byte, 1),
	}

	// Test registration
	hub.register <- client
	if len(hub.clients) != 1 {
		t.Fatalf("Expected 1 client after registration, got %d", len(hub.clients))
	}

	// Test broadcast
	message := []byte("hello")
	hub.broadcast <- message

	select {
	case received := <-client.send:
		if string(received) != "hello" {
			t.Errorf("Client received wrong message: got %s, want %s", received, message)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Client did not receive broadcast message in time")
	}

	// Test unregistration
	hub.unregister <- client
	// Allow the hub to process the unregister message
	time.Sleep(10 * time.Millisecond)
	if len(hub.clients) != 0 {
		t.Fatalf("Expected 0 clients after unregistration, got %d", len(hub.clients))
	}
}

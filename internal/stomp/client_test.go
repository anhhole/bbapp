package stomp_test

import (
	"testing"
	"bbapp/internal/stomp"
)

func TestClient_Connect(t *testing.T) {
	// Skip if no STOMP server available
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := stomp.NewClient("localhost:61613", "", "")
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

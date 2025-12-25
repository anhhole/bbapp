package browser_test

import (
	"testing"
	"bbapp/internal/browser"
)

func TestManager_CreateBrowser(t *testing.T) {
	manager := browser.NewManager()

	ctx, cancel, err := manager.CreateBrowser("test-id")
	if err != nil {
		t.Fatalf("CreateBrowser failed: %v", err)
	}
	defer cancel()

	if ctx == nil {
		t.Fatal("Expected non-nil context")
	}
}

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

func TestManager_Navigate(t *testing.T) {
	manager := browser.NewManager()
	ctx, cancel, _ := manager.CreateBrowser("nav-test")
	defer cancel()

	err := manager.Navigate(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}
}

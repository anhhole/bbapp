package listener_test

import (
	"testing"
	"bbapp/internal/listener"
	"bbapp/internal/browser"
)

func TestBigoListener_Start(t *testing.T) {
	manager := browser.NewManager()
	ctx, cancel, _ := manager.CreateBrowser("bigo-test")
	defer cancel()

	bigoListener := listener.NewBigoListener("12345", ctx)

	frameCount := 0
	bigoListener.OnGift(func(gift listener.Gift) {
		frameCount++
	})

	err := bigoListener.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

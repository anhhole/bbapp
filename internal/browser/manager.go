package browser

import (
	"context"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Manager manages browser instances
type Manager struct {
	browsers map[string]context.Context
	mutex    sync.RWMutex
}

// NewManager creates a new browser manager
func NewManager() *Manager {
	return &Manager{
		browsers: make(map[string]context.Context),
	}
}

// CreateBrowser creates a headless Chrome instance
func (m *Manager) CreateBrowser(id string) (context.Context, context.CancelFunc, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)

	// Start browser
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		cancelAlloc()
		return nil, nil, err
	}

	m.mutex.Lock()
	m.browsers[id] = ctx
	m.mutex.Unlock()

	return ctx, func() {
		cancel()
		cancelAlloc()
	}, nil
}

// Navigate navigates browser to URL
func (m *Manager) Navigate(ctx context.Context, url string) error {
	return chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second), // Wait for page load
	)
}

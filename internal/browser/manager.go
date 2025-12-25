package browser

import (
	"context"
	"sync"

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

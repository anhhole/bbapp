package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Setup log redirection
	setupLogging()

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "bbapp-temp",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func setupLogging() {
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		println("Error creating log dir:", err.Error())
		return
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("debug_%s.log", time.Now().Format("2006-01-02_15-04-05")))
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		println("Error opening log file:", err.Error())
		return
	}

	// Redirect standard output and error to file
	// Note: In a real GUI app without a console attached, this effectively captures
	// everything that *would* have gone to the console.
	// For Wails, we can't easily "tee" to the Wails console and file without
	// losing the Wails bindings' own output handling, but standard os.Stdout
	// replacement works for fmt.Printf calls.

	// Use MultiWriter if we still want to try sending to console (if it exists)
	// mw := io.MultiWriter(os.Stdout, f) // Can't easily replace os.Stdout with interface

	// Simple redirection
	os.Stdout = f
	os.Stderr = f

	fmt.Printf("=== Log Session Started: %s ===\n", time.Now().Format(time.RFC3339))
}

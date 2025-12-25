package fingerprint

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
)

var cachedHash string

func GenerateDeviceHash() (string, error) {
	if cachedHash != "" {
		return cachedHash, nil
	}

	// Collect machine identifiers
	hostname, _ := os.Hostname()
	osInfo := runtime.GOOS + runtime.GOARCH

	// TODO: Add MAC address if needed
	// For Windows, could use: wmic csproduct get UUID
	// For now, use hostname + OS info

	data := fmt.Sprintf("%s|%s", hostname, osInfo)
	hash := sha256.Sum256([]byte(data))

	cachedHash = fmt.Sprintf("%x", hash)
	return cachedHash, nil
}

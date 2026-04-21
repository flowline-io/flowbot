// Package utils provides utility functions for the CLI
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateID generates a unique ID using crypto/rand
func GenerateID() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fallbackID()
	}
	return hex.EncodeToString(bytes)
}

func fallbackID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

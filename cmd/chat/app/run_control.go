package app

import "fmt"

// runControlHint builds the footer status line after a cancel/confirm API call.
func runControlHint(err error, okHint, failLabel string) string {
	if err != nil {
		return fmt.Sprintf("%s: %v", failLabel, err)
	}
	return okHint
}

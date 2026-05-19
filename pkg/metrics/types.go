package metrics

import (
	"log"
	"regexp"
)

var safeLabelRe = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

func sanitizeLabel(v string) string {
	if len(v) > 128 {
		v = v[:128]
	}
	return safeLabelRe.ReplaceAllString(v, "_")
}

func recoverLog(metricName string) {
	if r := recover(); r != nil {
		log.Printf("[metrics] %s panic: %v", metricName, r)
	}
}

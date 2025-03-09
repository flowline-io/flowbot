package uikit

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	NotificationPosTopLeft      = "top-left"
	NotificationPosTopCenter    = "top-center"
	NotificationPosTopRight     = "top-right"
	NotificationPosBottomLeft   = "bottom-left"
	NotificationPosBottomCenter = "bottom-center"
	NotificationPosBottomRight  = "bottom-right"

	NotificationStatusPrimary = "primary"
	NotificationStatusSuccess = "success"
	NotificationStatusWarning = "warning"
	NotificationStatusDanger  = "danger"
)

// NotificationJS generates JavaScript code to display a notification
func NotificationJS(message string) string {
	return fmt.Sprintf("UIkit.notification('%s')", message)
}

// NotificationWithPosJS generates JavaScript code for a notification with position
func NotificationWithPosJS(message string, pos string) string {
	return fmt.Sprintf("UIkit.notification('%s', {pos: '%s'})", message, pos)
}

// NotificationWithStatusJS generates JavaScript code for a notification with status
func NotificationWithStatusJS(message string, status string) string {
	return fmt.Sprintf("UIkit.notification('%s', {status: '%s'})", message, status)
}

// NotificationWithOptionsJS generates JavaScript code for a notification with complete options
func NotificationWithOptionsJS(message string, pos string, status string, timeout int, group string) string {
	options := "{"

	if pos != "" {
		options += fmt.Sprintf("pos: '%s', ", pos)
	}

	if status != "" {
		options += fmt.Sprintf("status: '%s', ", status)
	}

	if timeout != 0 {
		options += fmt.Sprintf("timeout: %d, ", timeout)
	}

	if group != "" {
		options += fmt.Sprintf("group: '%s', ", group)
	}

	// Remove the last comma and space (if any)
	if len(options) > 1 {
		options = options[:len(options)-2]
	}

	options += "}"

	return fmt.Sprintf("UIkit.notification('%s', %s)", message, options)
}

// ShowNotification displays a notification
func ShowNotification(message string) app.HTMLScript {
	return app.Script().Text(NotificationJS(message))
}

// ShowNotificationWithPos displays a notification with position
func ShowNotificationWithPos(message string, pos string) app.HTMLScript {
	return app.Script().Text(NotificationWithPosJS(message, pos))
}

// ShowNotificationWithStatus displays a notification with status
func ShowNotificationWithStatus(message string, status string) app.HTMLScript {
	return app.Script().Text(NotificationWithStatusJS(message, status))
}

// ShowNotificationWithOptions displays a notification with complete options
func ShowNotificationWithOptions(message string, pos string, status string, timeout int, group string) app.HTMLScript {
	return app.Script().Text(NotificationWithOptionsJS(message, pos, status, timeout, group))
}

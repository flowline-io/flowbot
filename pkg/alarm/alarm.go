package alarm

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/rs/zerolog"
)

func Alarm(err error, skip int) {
	if err == nil {
		return
	}
	errorText := err.Error()

	// ignore filtered errors
	ok := filter(err.Error())
	if ok {
		return
	}

	caller := ""
	pc, file, line, ok := runtime.Caller(skip + zerolog.CallerSkipFrameCount)
	if !ok {
		caller = "unknown"
	} else {
		caller = zerolog.CallerMarshalFunc(pc, file, line)
	}

	ok, err = nx(err.Error())
	if err != nil {
		_, _ = fmt.Printf("[alarm] failed to set alarm key: %v\n", err)
	}
	if !ok {
		return
	}

	err = notify("flowbot alarm", errorText, caller)
	if err != nil {
		_, _ = fmt.Printf("[alarm] failed to send notification: %v\n", err)
	}
}

// filter checks if the given string contains any of the keywords in the alarm filter.
func filter(str string) bool {
	if config.App.Alarm.Filter == "" {
		return false
	}
	keywords := strings.SplitSeq(config.App.Alarm.Filter, "|")
	for keyword := range keywords {
		if strings.Contains(str, strings.TrimSpace(keyword)) {
			return true
		}
	}

	return false
}

// nx checks if an alarm error has been seen before in the last 24 hours.
func nx(text string) (bool, error) {
	h := sha1.New()
	_, _ = h.Write([]byte(text))
	hash := hex.EncodeToString(h.Sum(nil))
	key := fmt.Sprintf("alarm:%s", hash)

	_, ok := cache.Instance.Get(key)
	if ok {
		return false, nil
	}

	ok = cache.Instance.SetWithTTL(key, "1", 0, 24*time.Hour)
	if !ok {
		return false, nil
	}

	return true, nil
}

// notify sends a Slack notification with the given title and content.
func notify(title, content, caller string) error {
	// message template
	templateString := `{
    "text": "*🚨 {{.Title}}*",
    "attachments": [
        {
            "color": "#FF0000",
            "blocks": [
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": "*Error Details:*\n>>>{{.Content}}"
                    }
                },
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": "*Location:*\n>>>{{.Caller}}"
                    }
                },
                {
                    "type": "context",
                    "elements": [
                        {
                            "type": "mrkdwn",
                            "text": "🕐 {{.Timestamp}}"
                        }
                    ]
                }
            ]
        }
    ]
}`
	temp := template.Must(template.New("notify").Parse(templateString))
	data := struct {
		Title     string
		Content   string
		Caller    string
		Timestamp string
	}{
		Title:     title,
		Content:   content,
		Caller:    caller,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	var payload bytes.Buffer
	err := temp.Execute(&payload, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	resp, err := http.Post(
		config.App.Alarm.SlackWebhook,
		"application/json",
		bytes.NewBuffer(payload.Bytes()),
	)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read notification body: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send notification: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

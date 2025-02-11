package alarm

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/redis/go-redis/v9"
	"net/http"
	"strings"
	"text/template"
	"time"
)

var db *redis.Client

func InitAlarm() error {
	addr := fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port)
	password := config.App.Redis.Password
	if addr == ":" || password == "" {
		return errors.New("redis config error")
	}
	db = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           config.App.Redis.DB,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	})
	return nil
}

func filter(str string) bool {
	if config.App.Alarm.Filter == "" {
		return false
	}
	keywords := strings.Split(config.App.Alarm.Filter, "|")
	for _, keyword := range keywords {
		if strings.Contains(str, strings.TrimSpace(keyword)) {
			return true
		}
	}

	return false
}

func Alarm(err error) {
	if err == nil {
		return
	}
	errorText := err.Error()

	// ignore filtered errors
	ok := filter(err.Error())
	if ok {
		return
	}

	ok, err = nx(err.Error())
	if err != nil {
		_, _ = fmt.Printf("[alarm] failed to set alarm key: %v\n", err)
	}
	if !ok {
		return
	}

	err = notify("flowbot alarm", errorText)
	if err != nil {
		_, _ = fmt.Printf("[alarm] failed to send notification: %v\n", err)
	}
}

func nx(text string) (bool, error) {
	h := sha1.New()
	_, _ = h.Write([]byte(text))
	hash := hex.EncodeToString(h.Sum(nil))
	key := fmt.Sprintf("alarm:%s", hash)

	ok, err := db.SetNX(context.Background(), key, "1", 24*time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set alarm key: %w", err)
	}
	if !ok {
		return false, nil
	}

	return true, nil
}

// notify usage slack webhook
func notify(title, content string) error {
	// message template
	templateString := `{
    "text": "{{.Title}}",
	"attachments": [
		{
			"text": "{{.Content}}"
		}
	]
}`
	temp := template.Must(template.New("notify").Parse(templateString))
	data := struct {
		Title   string
		Content string
	}{title, content}
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
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send notification: %s", resp.Status)
	}

	return nil
}

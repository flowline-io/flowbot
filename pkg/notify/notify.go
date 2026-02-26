package notify

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

var handlers map[string]Notifyer

func Register(id string, notifyer Notifyer) {
	if handlers == nil {
		handlers = make(map[string]Notifyer)
	}

	if notifyer == nil {
		flog.Fatal("Register: notifyer is nil")
	}
	if _, dup := handlers[id]; dup {
		flog.Fatal("Register: called twice for notifyer %s", id)
	}
	handlers[id] = notifyer
}

func List() map[string]Notifyer {
	return handlers
}

func ParseTemplate(testString string, templates []string) (types.KV, error) {
	var patterns []string

	regex, err := regexp.Compile(`{(\w+)}`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}

	for _, v := range templates {
		s := regex.ReplaceAllString(v, `(?P<$1>[a-zA-Z0-9\.\-_]+)`)
		patterns = append(patterns, s)
	}

	result := make(types.KV)
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(testString)
		if len(match) > 0 {
			tmp := make(types.KV)
			for i, name := range re.SubexpNames() {
				if i != 0 && name != "" {
					tmp[name] = match[i]
				}
			}
			result = tmp
			break
		}
	}

	return result, nil
}

func ParseSchema(testString string) (string, error) {
	regex, err := regexp.Compile(`^([a-zA-Z0-9\-_]+)://`)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}
	s := regex.FindString(testString)
	s = strings.TrimSuffix(s, "://")
	return s, nil
}

func Send(text string, message Message) error {
	lines := strings.SplitSeq(text, "\n")
	for v := range lines {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		scheme, err := ParseSchema(v)
		if err != nil {
			flog.Info("[notify] %s parse schema error: %s", scheme, err)
			continue
		}
		if _, ok := handlers[scheme]; !ok {
			continue
		}

		tokens, err := ParseTemplate(v, handlers[scheme].Templates())
		if err != nil {
			flog.Info("[notify] %s parse template error: %s", scheme, err)
			continue
		}
		if err := handlers[scheme].Send(tokens, message); err != nil {
			flog.Info("[notify] %s send message error: %s", scheme, err)
		}
		flog.Info("[notify] %s send message", scheme)
	}

	return nil
}

func ChannelSend(uid types.Uid, name string, message Message) error {
	kv, err := store.Database.ConfigGet(uid, "", fmt.Sprintf("notify:%s", name))
	if err != nil {
		return err
	}
	template, ok := kv.String("value")
	if !ok {
		return errors.New("[notify] template not found")
	}

	return Send(template, message)
}

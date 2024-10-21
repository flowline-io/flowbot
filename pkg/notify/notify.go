package notify

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"regexp"
	"strings"
)

var handlers map[string]Notifyer

func Register(id string, notifyer Notifyer) {
	if handlers == nil {
		handlers = make(map[string]Notifyer)
	}

	if notifyer == nil {
		panic("Register: notifyer is nil")
	}
	if _, dup := handlers[id]; dup {
		panic("Register: called twice for notifyer " + id)
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
		return nil, err
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
	regex, err := regexp.Compile(`^(\w+)://`)
	if err != nil {
		return "", err
	}
	s := regex.FindString(testString)
	s = strings.TrimSuffix(s, "://")
	return s, nil
}

func Send(text string, message Message) error {
	lines := strings.Split(text, "\n")
	for _, v := range lines {
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

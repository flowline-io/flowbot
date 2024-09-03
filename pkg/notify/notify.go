package notify

import (
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/internal/types"
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
	patterns := []string{}

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

package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/flowline-io/flowbot/internal/types"
)

func Parse(_ types.Context, text string, variable ...interface{}) (string, error) {
	data := make(map[string]interface{})
	if len(variable) > 0 {
		for index, item := range variable {
			data[fmt.Sprintf("var%d", index+1)] = item
			// check variable placeholder, eg. $1, $2
			placeholder := fmt.Sprintf("$%d", index+1)
			if !strings.Contains(text, placeholder) {
				return "", fmt.Errorf("not contain placeholder %s", placeholder)
			}
			// todo Only hold less than 9 parameters
			text = strings.Replace(text, placeholder, fmt.Sprintf("{{ .var%d }}", index+1), 1)
		}
	}

	// Built-in variables
	if strings.Contains(text, "$username") {
		data["username"] = "username" // todo find username
		text = strings.Replace(text, "$username", "{{ .username }}", 1)
	}

	buf := bytes.NewBufferString("")
	t, err := template.New("tmpl").Parse(text)
	if err != nil {
		return "", err
	}
	err = t.Execute(buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

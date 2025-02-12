package reader

import (
	"fmt"

	rssClient "miniflux.app/v2/client"
)

func entriyFilter(entry *rssClient.Entry) bool {
	// todo allow_list
	// todo deny_list
	return false
}

func getAIResult(prompt, request string) (string, error) {
	messages := []string{
		// {"role": "system", "content": "You are a helpful assistant."},
		// {"role": "user", "content": request + "\n---\n" + prompt},
	}
	_, _ = fmt.Println(messages)
	return "", nil
}

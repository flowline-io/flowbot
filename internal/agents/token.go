package agents

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/pkoukk/tiktoken-go"
)

func CountToken(text string) int {
	encoding := tiktoken.MODEL_CL100K_BASE

	tke, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		flog.Warn("get encoding failed: %v", err)
		return 0
	}

	token := tke.Encode(text, nil, nil)

	return len(token)
}

func CountMessageTokens(messages []*Message) (int, error) {
	start := time.Now()

	var tokensPerMessage, tokensPerName int
	tokensPerMessage = 3
	tokensPerName = 1

	numTokens := 0
	for _, msg := range messages {
		numTokens += tokensPerMessage
		numTokens += CountToken(msg.Content)
		numTokens += CountToken(string(msg.Role))
		numTokens += CountToken(msg.Name)
		if msg.Name != "" {
			numTokens += tokensPerName
		}
	}
	numTokens += 3

	elapsed := time.Since(start)
	flog.Info("token count: %d, time: %s", numTokens, elapsed)
	return numTokens, nil
}

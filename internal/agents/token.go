package agents

import (
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/pkoukk/tiktoken-go"
)

// CountToken counts the number of tokens in text.
func CountToken(text string) int {
	encoding := tiktoken.MODEL_CL100K_BASE

	// if you don't want download dictionary at runtime, you can use offline loader
	// tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	tke, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		flog.Warn("get encoding failed: %v", err)
		return 0
	}

	// encode
	token := tke.Encode(text, nil, nil)

	return len(token)
}

// CountMessageTokens counts the total number of tokens in a message list.
func CountMessageTokens(messages []*schema.Message) (int, error) {
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
	numTokens += 3 // every reply is primed with <|start|>assistant<|message|>

	elapsed := time.Since(start)
	flog.Info("token count: %d, time: %s", numTokens, elapsed)
	return numTokens, nil
}

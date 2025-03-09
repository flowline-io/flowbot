package agents

import (
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/pkoukk/tiktoken-go"
)

func CountToken(text string) (int, error) {
	encoding := tiktoken.MODEL_CL100K_BASE

	// if you don't want download dictionary at runtime, you can use offline loader
	// tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
	tke, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return 0, fmt.Errorf("failed to get encoding: %w", err)
	}

	// encode
	token := tke.Encode(text, nil, nil)

	return len(token), nil
}

func CountMessageTokens(messages []*schema.Message) (int, error) {
	start := time.Now()
	totalToken := 0
	for _, msg := range messages {
		token, err := CountToken(msg.Content)
		if err != nil {
			return 0, fmt.Errorf("count token failed: %w", err)
		}
		totalToken += token
	}
	elapsed := time.Since(start)
	flog.Info("token count: %d, time: %s", totalToken, elapsed)
	return totalToken, nil
}

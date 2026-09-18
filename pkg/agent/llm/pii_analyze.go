package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/config"
)

const (
	defaultAnalyzeChunkBytes = 12 * 1024
	piiAnalyzePath           = "/analyze"
)

type analyzerClient struct {
	baseURL string
	timeout time.Duration
	score   float64
	client  *http.Client
}

func newAnalyzerClientFromConfig() *analyzerClient {
	cfg := config.App.PII
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = config.DefaultPIITimeout
	}
	score := cfg.ScoreThreshold
	if score <= 0 {
		score = config.DefaultPIIScoreThreshold
	}
	return &analyzerClient{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.AnalyzerURL), "/"),
		timeout: timeout,
		score:   score,
		client:  &http.Client{Timeout: timeout},
	}
}

type analyzeRequest struct {
	Text           string  `json:"text"`
	Language       string  `json:"language"`
	ScoreThreshold float64 `json:"score_threshold,omitempty"`
}

type analyzeResponseItem struct {
	EntityType string  `json:"entity_type"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Score      float64 `json:"score"`
}

func (c *analyzerClient) analyzeDual(ctx context.Context, text string) ([]recognizedSpan, error) {
	if c == nil || c.baseURL == "" {
		return nil, errors.New("agent llm pii: analyzer url is empty")
	}
	if text == "" {
		return nil, nil
	}
	var all []recognizedSpan
	for _, chunk := range chunkText(text, defaultAnalyzeChunkBytes) {
		en, err := c.analyzeLanguage(ctx, chunk.text, "en")
		if err != nil {
			return nil, err
		}
		zh, err := c.analyzeLanguage(ctx, chunk.text, "zh")
		if err != nil {
			return nil, err
		}
		for _, s := range append(en, zh...) {
			s.Start += chunk.offset
			s.End += chunk.offset
			all = append(all, s.withSnippet(text))
		}
	}
	return mergeRecognizedSpans(all), nil
}

func (c *analyzerClient) analyzeLanguage(ctx context.Context, text, language string) ([]recognizedSpan, error) {
	body, err := sonic.Marshal(analyzeRequest{
		Text:           text,
		Language:       language,
		ScoreThreshold: c.score,
	})
	if err != nil {
		return nil, fmt.Errorf("agent llm pii: marshal analyze request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+piiAnalyzePath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("agent llm pii: create analyze request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent llm pii: analyze %s: %w", language, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("agent llm pii: read analyze response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("agent llm pii: analyze %s status %d: %s", language, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var items []analyzeResponseItem
	if err := sonic.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("agent llm pii: decode analyze response: %w", err)
	}
	return spansFromAnalyzerItems(text, language, items), nil
}

// spansFromAnalyzerItems maps Presidio character (rune) indices onto Go UTF-8 byte offsets.
func spansFromAnalyzerItems(text, language string, items []analyzeResponseItem) []recognizedSpan {
	out := make([]recognizedSpan, 0, len(items))
	for _, item := range items {
		start, end, ok := runeRangeToBytes(text, item.Start, item.End)
		if !ok {
			continue
		}
		out = append(out, recognizedSpan{
			EntityType: item.EntityType,
			Start:      start,
			End:        end,
			Score:      item.Score,
			Language:   language,
		})
	}
	return out
}

// runeRangeToBytes converts a Presidio half-open rune range into UTF-8 byte offsets.
func runeRangeToBytes(text string, runeStart, runeEnd int) (byteStart, byteEnd int, ok bool) {
	if runeStart < 0 || runeEnd <= runeStart {
		return 0, 0, false
	}
	byteStart, byteEnd = -1, -1
	runeIdx := 0
	for i := 0; i < len(text); {
		if runeIdx == runeStart {
			byteStart = i
		}
		_, size := utf8.DecodeRuneInString(text[i:])
		i += size
		runeIdx++
		if runeIdx == runeEnd {
			byteEnd = i
			break
		}
	}
	if byteStart < 0 || byteEnd < 0 || byteStart >= byteEnd {
		return 0, 0, false
	}
	return byteStart, byteEnd, true
}

type textChunk struct {
	text   string
	offset int // byte offset into the original text
}

func chunkText(text string, maxBytes int) []textChunk {
	if maxBytes <= 0 || len(text) <= maxBytes {
		return []textChunk{{text: text, offset: 0}}
	}
	var chunks []textChunk
	for offset := 0; offset < len(text); {
		end := offset + maxBytes
		if end > len(text) {
			end = len(text)
		} else {
			split := end
			for i := end; i > offset+maxBytes/2; i-- {
				if text[i-1] == '\n' || text[i-1] == ' ' || text[i-1] == '\t' {
					split = i
					break
				}
			}
			end = split
		}
		end = clampToRuneStart(text, end)
		if end <= offset {
			_, size := utf8.DecodeRuneInString(text[offset:])
			end = offset + size
		}
		chunks = append(chunks, textChunk{text: text[offset:end], offset: offset})
		offset = end
	}
	return chunks
}

func clampToRuneStart(text string, i int) int {
	if i <= 0 {
		return 0
	}
	if i >= len(text) {
		return len(text)
	}
	for i > 0 && !utf8.RuneStart(text[i]) {
		i--
	}
	return i
}

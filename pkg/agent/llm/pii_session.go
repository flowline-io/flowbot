package llm

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/flowline-io/flowbot/pkg/config"
)

type piiSessionContextKey struct{}

// WithPIISession attaches a chat session ID so GenerateContent reuses one reversible
// placeholder table across turns (including compaction).
func WithPIISession(ctx context.Context, sessionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ctx
	}
	return context.WithValue(ctx, piiSessionContextKey{}, sessionID)
}

// PIISessionFromContext returns the session ID attached by WithPIISession.
func PIISessionFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw, ok := ctx.Value(piiSessionContextKey{}).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(raw)
}

type recognizedSpan struct {
	EntityType  string
	Start       int
	End         int
	Score       float64
	Language    string
	snippetHint string
}

type piiTable struct {
	mu         sync.Mutex
	forward    map[string]string // original -> placeholder
	reverse    map[string]string // placeholder -> original
	counters   map[string]int
	hashCache  map[string]string // sha256(original) -> anonymized
	lastAccess time.Time
}

type textReplacement struct {
	start, end int
	token      string
}

func newPIITable() *piiTable {
	return &piiTable{
		forward:    make(map[string]string),
		reverse:    make(map[string]string),
		counters:   make(map[string]int),
		hashCache:  make(map[string]string),
		lastAccess: time.Now(),
	}
}

func (t *piiTable) touch() {
	t.lastAccess = time.Now()
}

func (t *piiTable) anonymizeLocked(text string, spans []recognizedSpan) string {
	t.touch()
	if text == "" {
		return text
	}
	sum := hashText(text)
	if cached, ok := t.hashCache[sum]; ok {
		return cached
	}
	built := applySpanReplacements(text, t.collectReplacements(text, spans))
	built = t.applyKnownOriginals(built)
	t.hashCache[sum] = built
	return built
}

func (t *piiTable) collectReplacements(text string, spans []recognizedSpan) []textReplacement {
	reps := make([]textReplacement, 0, len(spans))
	occupied := make([]bool, len(text))
	for _, span := range spans {
		if !validSpan(text, span) || rangeOccupied(occupied, span.Start, span.End) {
			continue
		}
		original := text[span.Start:span.End]
		if original == "" || looksLikePlaceholder(original) {
			continue
		}
		token, ok := t.forward[original]
		if !ok {
			token = t.allocateLocked(span.EntityType, original)
		}
		markOccupied(occupied, span.Start, span.End)
		reps = append(reps, textReplacement{start: span.Start, end: span.End, token: token})
	}
	slices.SortFunc(reps, func(a, b textReplacement) int { return cmp.Compare(b.start, a.start) })
	return reps
}

func (t *piiTable) applyKnownOriginals(built string) string {
	known := make([]string, 0, len(t.forward))
	for original := range t.forward {
		known = append(known, original)
	}
	slices.SortFunc(known, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	for _, original := range known {
		built = strings.ReplaceAll(built, original, t.forward[original])
	}
	return built
}

func applySpanReplacements(text string, reps []textReplacement) string {
	built := text
	for _, r := range reps {
		built = built[:r.start] + r.token + built[r.end:]
	}
	return built
}

func validSpan(text string, span recognizedSpan) bool {
	return span.Start >= 0 && span.End <= len(text) && span.Start < span.End
}

func rangeOccupied(occupied []bool, start, end int) bool {
	for i := start; i < end; i++ {
		if occupied[i] {
			return true
		}
	}
	return false
}

func markOccupied(occupied []bool, start, end int) {
	for i := start; i < end; i++ {
		occupied[i] = true
	}
}

func (t *piiTable) allocateLocked(entityType, original string) string {
	if token, ok := t.forward[original]; ok {
		return token
	}
	key := sanitizeEntityType(entityType)
	t.counters[key]++
	token := fmt.Sprintf("{{PII_%s_%d}}", key, t.counters[key])
	for t.reverse[token] != "" || strings.Contains(original, token) {
		t.counters[key]++
		token = fmt.Sprintf("{{PII_%s_%d}}", key, t.counters[key])
	}
	t.forward[original] = token
	t.reverse[token] = original
	return token
}

func (t *piiTable) restoreLocked(text string) string {
	t.touch()
	if text == "" || len(t.reverse) == 0 {
		return text
	}
	out := text
	tokens := make([]string, 0, len(t.reverse))
	for token := range t.reverse {
		tokens = append(tokens, token)
	}
	slices.SortFunc(tokens, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	for _, token := range tokens {
		out = strings.ReplaceAll(out, token, t.reverse[token])
	}
	return out
}

var placeholderRE = regexp.MustCompile(`\{\{PII_[A-Z0-9_]+_\d+\}\}`)

func looksLikePlaceholder(s string) bool {
	return placeholderRE.MatchString(s) && placeholderRE.FindString(s) == s
}

func sanitizeEntityType(entityType string) string {
	entityType = strings.TrimSpace(strings.ToUpper(entityType))
	if entityType == "" {
		return "ENTITY"
	}
	var b strings.Builder
	for _, r := range entityType {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			_, _ = b.WriteRune(r)
		case r == '_' || r == '-' || r == ' ':
			_ = b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "ENTITY"
	}
	return out
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

type piiSessionStore struct {
	mu      sync.Mutex
	tables  map[string]*piiTable
	ttl     time.Duration
	lastGC  time.Time
	gcEvery time.Duration
}

var globalPIISessions = &piiSessionStore{
	tables:  make(map[string]*piiTable),
	ttl:     config.DefaultPIISessionTTL,
	gcEvery: time.Hour,
}

func (s *piiSessionStore) tableFor(sessionID string, ttl time.Duration) *piiTable {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ttl > 0 {
		s.ttl = ttl
	}
	s.gcLocked(time.Now())
	if sessionID == "" {
		return newPIITable()
	}
	tab, ok := s.tables[sessionID]
	if !ok {
		tab = newPIITable()
		s.tables[sessionID] = tab
	}
	tab.mu.Lock()
	tab.touch()
	tab.mu.Unlock()
	return tab
}

func (s *piiSessionStore) gcLocked(now time.Time) {
	if s.ttl <= 0 {
		return
	}
	if !s.lastGC.IsZero() && now.Sub(s.lastGC) < s.gcEvery {
		return
	}
	s.lastGC = now
	for id, tab := range s.tables {
		tab.mu.Lock()
		idle := now.Sub(tab.lastAccess)
		tab.mu.Unlock()
		if idle > s.ttl {
			delete(s.tables, id)
		}
	}
}

func tableFromContext(ctx context.Context) *piiTable {
	ttl := config.App.PII.SessionTTL
	if ttl <= 0 {
		ttl = config.DefaultPIISessionTTL
	}
	return globalPIISessions.tableFor(PIISessionFromContext(ctx), ttl)
}

// mergeRecognizedSpans unions en/zh Analyzer hits and drops language-mismatched NER.
func mergeRecognizedSpans(spans []recognizedSpan) []recognizedSpan {
	if len(spans) == 0 {
		return nil
	}
	filtered := filterSpans(spans)
	slices.SortFunc(filtered, compareSpansByScore)
	kept := pickNonOverlapping(filtered)
	slices.SortFunc(kept, func(a, b recognizedSpan) int { return cmp.Compare(a.Start, b.Start) })
	return kept
}

func filterSpans(spans []recognizedSpan) []recognizedSpan {
	filtered := make([]recognizedSpan, 0, len(spans))
	for _, s := range spans {
		if s.Start < 0 || s.End <= s.Start || shouldDropLanguageMismatch(s) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

func compareSpansByScore(a, b recognizedSpan) int {
	if c := cmp.Compare(b.Score, a.Score); c != 0 {
		return c
	}
	if c := cmp.Compare(b.End-b.Start, a.End-a.Start); c != 0 {
		return c
	}
	return cmp.Compare(a.Start, b.Start)
}

func pickNonOverlapping(filtered []recognizedSpan) []recognizedSpan {
	type occupied struct{ start, end int }
	kept := make([]recognizedSpan, 0, len(filtered))
	var used []occupied
	for _, s := range filtered {
		overlap := false
		for _, u := range used {
			if s.Start < u.end && s.End > u.start {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		kept = append(kept, s)
		used = append(used, occupied{start: s.Start, end: s.End})
	}
	return kept
}

func shouldDropLanguageMismatch(s recognizedSpan) bool {
	entity := strings.ToUpper(s.EntityType)
	ner := entity == "PERSON" || entity == "LOCATION" || entity == "NRP" || entity == "ORGANIZATION" || entity == "ORG"
	if !ner {
		return false
	}
	switch s.Language {
	case "en":
		return hasCJK(s.snippetHint)
	case "zh":
		return s.snippetHint != "" && !hasCJK(s.snippetHint)
	default:
		return false
	}
}

func (s recognizedSpan) withSnippet(text string) recognizedSpan {
	if s.Start >= 0 && s.End <= len(text) && s.Start < s.End {
		s.snippetHint = text[s.Start:s.End]
	}
	return s
}

func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) {
			return true
		}
	}
	return false
}

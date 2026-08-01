package slack

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/slack-go/slack"
)

// ──────────────────────────────────────────
// Block Kit builder helpers
// ──────────────────────────────────────────

// header creates a header block (Slack plain_text max 150 runes).
func header(text string) *slack.HeaderBlock {
	return slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, truncateRunes(text, slackHeaderMaxLen), false, false),
	)
}

func section(text string) *slack.SectionBlock {
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
		nil, nil,
	)
}

// sectionWithButton creates a section block with a button accessory.
// When url is non-empty, Slack opens it in the browser (link button).
func sectionWithButton(text, btnText, actionID, value, url string, style slack.Style) *slack.SectionBlock {
	btn := slack.NewButtonBlockElement(actionID, value,
		slack.NewTextBlockObject(slack.PlainTextType, btnText, true, false),
	)
	if style != "" {
		btn.Style = style
	}
	if url != "" {
		btn.URL = url
	}
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
		nil, slack.NewAccessory(btn),
	)
}

// slackMaxSectionFields is Slack's limit for fields in a single section block.
const (
	slackMaxSectionFields = 10
	slackHeaderMaxLen     = 150
	slackFieldValueMaxLen = 500
	slackSectionTextMax   = 2900
	slackMaxBlocks        = 50
)

func truncateRunes(s string, limit int) string {
	if limit <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}

func runeLen(s string) int {
	return len([]rune(s))
}

// sectionFields creates section blocks with field pairs in deterministic order.
// Fields are split into chunks of slackMaxSectionFields to respect the Slack API limit.
// Long values are rendered as full-width sections instead of cramped field cells.
func sectionFields(fields map[string]string) []slack.Block {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var blocks []slack.Block
	var shortKeys []string
	for _, k := range keys {
		v := fields[k]
		if runeLen(v) > slackFieldValueMaxLen {
			budget := max(slackSectionTextMax-runeLen(k)-10, 1)
			blocks = append(blocks, section(fmt.Sprintf("*%s:*\n%s", k, truncateRunes(v, budget))))
			continue
		}
		shortKeys = append(shortKeys, k)
	}

	for i := 0; i < len(shortKeys); i += slackMaxSectionFields {
		end := min(i+slackMaxSectionFields, len(shortKeys))
		chunk := shortKeys[i:end]
		textFields := make([]*slack.TextBlockObject, 0, len(chunk))
		for _, k := range chunk {
			textFields = append(textFields, slack.NewTextBlockObject(
				slack.MarkdownType, fmt.Sprintf("*%s:*\n%s", k, fields[k]), false, false,
			))
		}
		blocks = append(blocks, slack.NewSectionBlock(nil, textFields, nil))
	}
	return blocks
}

// markdownMaxBlockLen keeps section text under Slack's 3000-character limit.
const markdownMaxBlockLen = slackSectionTextMax

func markdownTextBlocks(text string) []slack.Block {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	var blocks []slack.Block
	var chunk []string
	chunkLen := 0
	flush := func() {
		if len(chunk) == 0 {
			return
		}
		blocks = append(blocks, section(strings.Join(chunk, "\n")))
		chunk = nil
		chunkLen = 0
	}
	for _, line := range lines {
		if utils.IsMrkdwnHorizontalRule(line) {
			flush()
			blocks = append(blocks, divider())
			continue
		}
		for runeLen(line) > markdownMaxBlockLen {
			runes := []rune(line)
			part := string(runes[:markdownMaxBlockLen])
			if chunkLen+markdownMaxBlockLen+1 > markdownMaxBlockLen && len(chunk) > 0 {
				flush()
			}
			chunk = append(chunk, part)
			flush()
			line = string(runes[markdownMaxBlockLen:])
		}
		extra := runeLen(line) + 1
		if chunkLen+extra > markdownMaxBlockLen && len(chunk) > 0 {
			flush()
		}
		chunk = append(chunk, line)
		chunkLen += extra
	}
	flush()
	return blocks
}

// contextBlock creates a context block with multiple text elements.
func contextBlock(elements ...string) *slack.ContextBlock {
	var mixed []slack.MixedElement
	for _, e := range elements {
		mixed = append(mixed, slack.NewTextBlockObject(slack.MarkdownType, e, false, false))
	}
	return slack.NewContextBlock("", mixed...)
}

// divider creates a divider block.
func divider() *slack.DividerBlock {
	return slack.NewDividerBlock()
}

// actionButtons creates an actions block with multiple buttons.
func actionButtons(buttons ...ButtonDef) *slack.ActionBlock {
	var elements []slack.BlockElement
	for _, b := range buttons {
		btn := slack.NewButtonBlockElement(b.ActionID, b.Value,
			slack.NewTextBlockObject(slack.PlainTextType, b.Text, true, false),
		)
		if b.Style != "" {
			btn.Style = b.Style
		}
		if b.URL != "" {
			btn.URL = b.URL
		}
		if b.Confirm != nil {
			btn.Confirm = b.Confirm
		}
		elements = append(elements, btn)
	}
	return slack.NewActionBlock("", elements...)
}

// ButtonDef describes a button for actionButtons.
type ButtonDef struct {
	Text     string
	ActionID string
	Value    string
	Style    slack.Style // slack.StylePrimary or slack.StyleDanger
	URL      string
	Confirm  *slack.ConfirmationBlockObject
}

// FormFieldDef describes a field in a form segment.
type FormFieldDef struct {
	Label       string
	Key         string
	Type        string
	Placeholder string
	Options     []string
	Optional    bool
	InitialVal  string
}

// imageBlock creates an image block with title and alt text.
func imageBlock(url, altText, title string) *slack.ImageBlock {
	var titleObj *slack.TextBlockObject
	if title != "" {
		titleObj = slack.NewTextBlockObject(slack.PlainTextType, title, false, false)
	}
	return slack.NewImageBlock(url, altText, "", titleObj)
}

// imageSection creates a section with image accessory.
func imageSection(text, imageURL, altText string) *slack.SectionBlock {
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
		nil,
		slack.NewAccessory(slack.NewImageBlockElement(imageURL, altText)),
	)
}

// ──────────────────────────────────────────
// Chart rendering helpers (text-based)
// ──────────────────────────────────────────

const (
	barFull  = "█"
	barEmpty = "░"
)

// renderBarChart builds a text-based horizontal bar chart suitable for Slack mrkdwn.
func renderBarChart(title, subtitle string, labels []string, values []float64) []slack.Block {
	var blocks []slack.Block

	if title != "" {
		blocks = append(blocks, header(title))
	}
	if subtitle != "" {
		blocks = append(blocks, contextBlock(fmt.Sprintf("_%s_", subtitle)))
	}

	if len(labels) == 0 || len(values) == 0 {
		return blocks
	}

	// find max for scaling
	maxVal := 0.0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// find max label width for alignment
	maxLabelLen := 0
	for _, l := range labels {
		if len(l) > maxLabelLen {
			maxLabelLen = len(l)
		}
	}

	const barWidth = 20
	var lines []string
	for i, label := range labels {
		if i >= len(values) {
			break
		}
		v := values[i]
		filled := int(math.Round(v / maxVal * barWidth))
		bar := strings.Repeat(barFull, filled) + strings.Repeat(barEmpty, barWidth-filled)
		paddedLabel := fmt.Sprintf("%-*s", maxLabelLen, label)
		lines = append(lines, fmt.Sprintf("`%s` %s  *%.1f*", paddedLabel, bar, v))
	}

	blocks = append(blocks, section(strings.Join(lines, "\n")))
	return blocks
}

// renderPieChart builds a text-based pie chart representation using proportional indicators.
func renderPieChart(title string, labels []string, values []float64) []slack.Block {
	var blocks []slack.Block

	if title != "" {
		blocks = append(blocks, header(title))
	}

	if len(labels) == 0 || len(values) == 0 {
		return blocks
	}

	total := 0.0
	for _, v := range values {
		total += v
	}
	if total == 0 {
		total = 1
	}

	pieMarks := []string{"(1)", "(2)", "(3)", "(4)", "(5)", "(6)", "(7)", "(8)"}

	var lines []string
	for i, label := range labels {
		if i >= len(values) {
			break
		}
		pct := values[i] / total * 100
		mark := pieMarks[i%len(pieMarks)]
		// proportional bar
		fillCount := int(math.Round(pct / 5)) // each block = 5%
		bar := strings.Repeat("■", fillCount) + strings.Repeat("□", 20-fillCount)
		lines = append(lines, fmt.Sprintf("%s *%s*  `%s`  %.1f%%", mark, label, bar, pct))
	}

	blocks = append(blocks, section(strings.Join(lines, "\n")))
	return blocks
}

// ──────────────────────────────────────────
// Status / Thinking indicator
// ──────────────────────────────────────────

// statusBlocks returns blocks showing a "thinking" / processing indicator.
func statusBlocks(statusText string) []slack.Block {
	return []slack.Block{
		contextBlock(fmt.Sprintf("_%s_", statusText)),
	}
}

// ──────────────────────────────────────────
// Action card builder
// ──────────────────────────────────────────

// ActionCardDef describes a rich action card.
type ActionCardDef struct {
	Title       string
	Description string
	Fields      map[string]string // key-value fields displayed in the card
	ImageURL    string            // optional thumbnail
	Buttons     []ButtonDef
	Footer      string
}

const actionCardMaxBlockLen = slackSectionTextMax

// descriptionBlocks renders action-card body text as Slack mrkdwn sections.
func descriptionBlocks(description string) []slack.Block {
	return markdownTextBlocks(description)
}

// buildActionCard builds an action card from the definition.
func buildActionCard(card ActionCardDef) []slack.Block {
	var blocks []slack.Block

	if card.Title != "" {
		blocks = append(blocks, header(card.Title))
	}
	blocks = append(blocks, descriptionBlocks(card.Description)...)
	if card.ImageURL != "" {
		blocks = append(blocks, imageBlock(card.ImageURL, card.Title, card.Title))
	}
	if len(card.Fields) > 0 {
		blocks = append(blocks, divider())
		blocks = append(blocks, sectionFields(card.Fields)...)
	}
	if len(card.Buttons) > 0 {
		blocks = append(blocks, divider(), actionButtons(card.Buttons...))
	}
	if card.Footer != "" {
		blocks = append(blocks, contextBlock(card.Footer))
	}

	return blocks
}

// ──────────────────────────────────────────
// Table builder
// ──────────────────────────────────────────

// Slack table block limits (see Block Kit table-block docs).
const (
	slackTableMaxRows = 100
	slackTableMaxCols = 20
)

// buildTableBlocks renders a table with Slack's native table block.
func buildTableBlocks(title string, headers []string, rows [][]any) []slack.Block {
	var blocks []slack.Block
	if title != "" {
		blocks = append(blocks, header(title))
	}
	if len(headers) == 0 && len(rows) == 0 {
		return blocks
	}

	colCount := tableColumnCount(headers, rows)
	if colCount == 0 {
		return blocks
	}

	table := slack.NewTableBlock("")
	if len(headers) > 0 {
		table.AddRow(tableHeaderRow(headers, colCount)...)
	}
	for _, row := range clampTableRows(rows, len(headers) > 0) {
		table.AddRow(tableDataRow(row, colCount)...)
	}
	table.WithColumnSettings(tableColumnSettings(colCount)...)
	return append(blocks, table)
}

func tableColumnCount(headers []string, rows [][]any) int {
	colCount := len(headers)
	if colCount == 0 {
		for _, row := range rows {
			if len(row) > colCount {
				colCount = len(row)
			}
		}
	}
	if colCount > slackTableMaxCols {
		return slackTableMaxCols
	}
	return colCount
}

func clampTableRows(rows [][]any, hasHeader bool) [][]any {
	maxDataRows := slackTableMaxRows
	if hasHeader {
		maxDataRows--
	}
	if maxDataRows < 0 {
		maxDataRows = 0
	}
	if len(rows) > maxDataRows {
		return rows[:maxDataRows]
	}
	return rows
}

func tableHeaderRow(headers []string, colCount int) []slack.TableCell {
	cells := make([]slack.TableCell, 0, colCount)
	for i := range colCount {
		text := ""
		if i < len(headers) {
			text = headers[i]
		}
		cells = append(cells, tableHeaderCell(text))
	}
	return cells
}

func tableDataRow(row []any, colCount int) []slack.TableCell {
	cells := make([]slack.TableCell, 0, colCount)
	for i := range colCount {
		text := ""
		if i < len(row) && row[i] != nil {
			text = fmt.Sprintf("%v", row[i])
		}
		cells = append(cells, tableDataCell(text))
	}
	return cells
}

func tableColumnSettings(colCount int) []slack.ColumnSetting {
	settings := make([]slack.ColumnSetting, colCount)
	for i := range settings {
		settings[i] = slack.ColumnSetting{
			Align:     slack.ColumnAlignmentLeft,
			IsWrapped: true,
		}
	}
	return settings
}

func tableHeaderCell(text string) slack.TableCell {
	return slack.NewTableRichTextCell(
		&slack.RichTextSection{
			Type: slack.RTESection,
			Elements: []slack.RichTextSectionElement{
				slack.NewRichTextSectionTextElement(text, &slack.RichTextSectionTextStyle{Bold: true}),
			},
		},
	)
}

const slackTableCellMaxLen = 256

func tableDataCell(text string) slack.TableCell {
	text = truncateRunes(text, slackTableCellMaxLen)
	if looksLikeHTTPURL(text) {
		return slack.NewTableRichTextCell(
			&slack.RichTextSection{
				Type: slack.RTESection,
				Elements: []slack.RichTextSectionElement{
					slack.NewRichTextSectionLinkElement(text, text, nil),
				},
			},
		)
	}
	return slack.NewTableRawTextCell(text)
}

func looksLikeHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

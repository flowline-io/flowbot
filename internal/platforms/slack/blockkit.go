package slack

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/slack-go/slack"
)

// ──────────────────────────────────────────
// Block Kit builder helpers
// ──────────────────────────────────────────

// header creates a header block.
func header(text string) *slack.HeaderBlock {
	return slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, text, false, false),
	)
}

// section creates a section block with markdown text.
func section(text string) *slack.SectionBlock {
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
		nil, nil,
	)
}

// sectionWithButton creates a section block with a button accessory.
func sectionWithButton(text, btnText, actionID, value string, style slack.Style) *slack.SectionBlock {
	btn := slack.NewButtonBlockElement(actionID, value,
		slack.NewTextBlockObject(slack.PlainTextType, btnText, true, false),
	)
	if style != "" {
		btn.Style = style
	}
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
		nil, slack.NewAccessory(btn),
	)
}

// sectionFields creates a section block with field pairs in deterministic order.
func sectionFields(fields map[string]string) *slack.SectionBlock {
	// Sort keys for deterministic rendering order
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	textFields := make([]*slack.TextBlockObject, 0, len(keys))
	for _, k := range keys {
		textFields = append(textFields, slack.NewTextBlockObject(
			slack.MarkdownType, fmt.Sprintf("*%s:*\n%s", k, fields[k]), false, false,
		))
	}
	return slack.NewSectionBlock(nil, textFields, nil)
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

// confirmDialog creates a confirmation dialog for destructive actions.
func confirmDialog(title, text, confirm, deny string) *slack.ConfirmationBlockObject {
	return slack.NewConfirmationBlockObject(
		slack.NewTextBlockObject(slack.PlainTextType, title, false, false),
		slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
		slack.NewTextBlockObject(slack.PlainTextType, confirm, false, false),
		slack.NewTextBlockObject(slack.PlainTextType, deny, false, false),
	)
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

	pieEmojis := []string{"🔵", "🟢", "🟡", "🟠", "🔴", "🟣", "⚫", "⚪"}

	var lines []string
	for i, label := range labels {
		if i >= len(values) {
			break
		}
		pct := values[i] / total * 100
		emoji := pieEmojis[i%len(pieEmojis)]
		// proportional bar
		fillCount := int(math.Round(pct / 5)) // each block = 5%
		bar := strings.Repeat("■", fillCount) + strings.Repeat("□", 20-fillCount)
		lines = append(lines, fmt.Sprintf("%s *%s*  `%s`  %.1f%%", emoji, label, bar, pct))
	}

	blocks = append(blocks, section(strings.Join(lines, "\n")))
	return blocks
}

// ──────────────────────────────────────────
// Form / Modal builder
// ──────────────────────────────────────────

// FormFieldDef describes a field in a modal form.
type FormFieldDef struct {
	Label       string
	Key         string
	Type        string // text, number, email, url, date, time, select, radio, checkbox, textarea
	Placeholder string
	Options     []string // for select / radio / checkbox
	Optional    bool
	InitialVal  string
}

// buildModalView creates a Slack modal view from form fields.
func buildModalView(callbackID, title, submitLabel string, fields []FormFieldDef) slack.ModalViewRequest {
	var inputBlocks slack.Blocks

	for _, f := range fields {
		var element slack.BlockElement

		switch f.Type {
		case "select":
			var opts []*slack.OptionBlockObject
			for _, o := range f.Options {
				opts = append(opts, slack.NewOptionBlockObject(o,
					slack.NewTextBlockObject(slack.PlainTextType, o, false, false), nil))
			}
			sel := slack.NewOptionsSelectBlockElement(
				slack.OptTypeStatic, // static_select
				slack.NewTextBlockObject(slack.PlainTextType, f.Placeholder, false, false),
				f.Key, opts...,
			)
			if f.InitialVal != "" {
				sel.InitialOption = slack.NewOptionBlockObject(f.InitialVal,
					slack.NewTextBlockObject(slack.PlainTextType, f.InitialVal, false, false), nil)
			}
			element = sel

		case "radio":
			var opts []*slack.OptionBlockObject
			for _, o := range f.Options {
				opts = append(opts, slack.NewOptionBlockObject(o,
					slack.NewTextBlockObject(slack.PlainTextType, o, false, false), nil))
			}
			radio := slack.NewRadioButtonsBlockElement(f.Key, opts...)
			if f.InitialVal != "" {
				radio.InitialOption = slack.NewOptionBlockObject(f.InitialVal,
					slack.NewTextBlockObject(slack.PlainTextType, f.InitialVal, false, false), nil)
			}
			element = radio

		case "checkbox":
			var opts []*slack.OptionBlockObject
			for _, o := range f.Options {
				opts = append(opts, slack.NewOptionBlockObject(o,
					slack.NewTextBlockObject(slack.PlainTextType, o, false, false), nil))
			}
			element = slack.NewCheckboxGroupsBlockElement(f.Key, opts...)

		case "date":
			dp := slack.NewDatePickerBlockElement(f.Key)
			if f.InitialVal != "" {
				dp.InitialDate = f.InitialVal
			}
			if f.Placeholder != "" {
				dp.Placeholder = slack.NewTextBlockObject(slack.PlainTextType, f.Placeholder, false, false)
			}
			element = dp

		case "time":
			tp := slack.NewTimePickerBlockElement(f.Key)
			if f.InitialVal != "" {
				tp.InitialTime = f.InitialVal
			}
			element = tp

		case "textarea":
			input := slack.NewPlainTextInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, f.Placeholder, false, false),
				f.Key,
			)
			input.Multiline = true
			if f.InitialVal != "" {
				input.InitialValue = f.InitialVal
			}
			element = input

		case "number":
			input := slack.NewNumberInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, f.Placeholder, false, false),
				f.Key, false,
			)
			if f.InitialVal != "" {
				input.InitialValue = f.InitialVal
			}
			element = input

		case "url":
			input := slack.NewURLTextInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, f.Placeholder, false, false),
				f.Key,
			)
			if f.InitialVal != "" {
				input.InitialValue = f.InitialVal
			}
			element = input

		case "email":
			input := slack.NewEmailTextInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, f.Placeholder, false, false),
				f.Key,
			)
			if f.InitialVal != "" {
				input.InitialValue = f.InitialVal
			}
			element = input

		default: // text, password, etc.
			input := slack.NewPlainTextInputBlockElement(
				slack.NewTextBlockObject(slack.PlainTextType, f.Placeholder, false, false),
				f.Key,
			)
			if f.InitialVal != "" {
				input.InitialValue = f.InitialVal
			}
			element = input
		}

		inputBlock := slack.NewInputBlock(
			f.Key,
			slack.NewTextBlockObject(slack.PlainTextType, f.Label, false, false),
			nil,
			element,
		)
		inputBlock.Optional = f.Optional
		inputBlocks.BlockSet = append(inputBlocks.BlockSet, inputBlock)
	}

	if submitLabel == "" {
		submitLabel = "Submit"
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: callbackID,
		Title:      slack.NewTextBlockObject(slack.PlainTextType, truncate(title, 24), false, false),
		Submit:     slack.NewTextBlockObject(slack.PlainTextType, submitLabel, false, false),
		Close:      slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		Blocks:     inputBlocks,
	}
}

// ──────────────────────────────────────────
// Status / Thinking indicator
// ──────────────────────────────────────────

// statusBlocks returns blocks showing a "thinking" / processing indicator.
func statusBlocks(statusText string) []slack.Block {
	return []slack.Block{
		contextBlock(fmt.Sprintf("⏳ _%s_", statusText)),
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

// buildActionCard builds an action card from the definition.
func buildActionCard(card ActionCardDef) []slack.Block {
	var blocks []slack.Block

	if card.Title != "" {
		blocks = append(blocks, header(card.Title))
	}
	if card.Description != "" {
		blocks = append(blocks, section(card.Description))
	}
	if card.ImageURL != "" {
		blocks = append(blocks, imageBlock(card.ImageURL, card.Title, card.Title))
	}
	if len(card.Fields) > 0 {
		blocks = append(blocks, divider())
		blocks = append(blocks, sectionFields(card.Fields))
	}
	if len(card.Buttons) > 0 {
		blocks = append(blocks, divider())
		blocks = append(blocks, actionButtons(card.Buttons...))
	}
	if card.Footer != "" {
		blocks = append(blocks, contextBlock(card.Footer))
	}

	return blocks
}

// ──────────────────────────────────────────
// Table builder
// ──────────────────────────────────────────

// buildTableBlocks renders a table using mrkdwn in section blocks.
func buildTableBlocks(title string, headers []string, rows [][]any) []slack.Block {
	var blocks []slack.Block

	if title != "" {
		blocks = append(blocks, header(title))
	}

	if len(headers) == 0 && len(rows) == 0 {
		return blocks
	}

	// calculate column widths
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				s := fmt.Sprintf("%v", cell)
				if len(s) > colWidths[i] {
					colWidths[i] = len(s)
				}
			}
		}
	}

	// build lines
	var lines []string

	// header row
	var headerParts []string
	for i, h := range headers {
		headerParts = append(headerParts, fmt.Sprintf("*%-*s*", colWidths[i], h))
	}
	lines = append(lines, strings.Join(headerParts, " │ "))

	// separator
	var sepParts []string
	for _, w := range colWidths {
		sepParts = append(sepParts, strings.Repeat("─", w))
	}
	lines = append(lines, strings.Join(sepParts, "─┼─"))

	// data rows
	for _, row := range rows {
		var parts []string
		for i, cell := range row {
			w := 0
			if i < len(colWidths) {
				w = colWidths[i]
			}
			parts = append(parts, fmt.Sprintf("%-*v", w, cell))
		}
		lines = append(lines, strings.Join(parts, " │ "))
	}

	// Slack has a 3000 char limit per text block, so chunk by row if needed
	const maxBlockLen = 2900
	var chunk []string
	chunkLen := 0
	flush := func() {
		if len(chunk) > 0 {
			blocks = append(blocks, section("```\n"+strings.Join(chunk, "\n")+"\n```"))
			chunk = nil
			chunkLen = 0
		}
	}
	for _, line := range lines {
		if chunkLen+len(line)+1 > maxBlockLen && len(chunk) > 0 {
			flush()
		}
		chunk = append(chunk, line)
		chunkLen += len(line) + 1
	}
	flush()

	return blocks
}

// ──────────────────────────────────────────
// Utility
// ──────────────────────────────────────────

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

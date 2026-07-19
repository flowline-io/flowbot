package partials

// StatusGlyph identifies which SVG shape a status icon renders.
type StatusGlyph string

const (
	// StatusGlyphCheck is a checkmark for positive / completed states.
	StatusGlyphCheck StatusGlyph = "check"
	// StatusGlyphX is a cross for disabled / failed states.
	StatusGlyphX StatusGlyph = "x"
	// StatusGlyphPause is a pause mark inside a circle for paused states.
	StatusGlyphPause StatusGlyph = "pause"
	// StatusGlyphPlay is a play mark for active / running states.
	StatusGlyphPlay StatusGlyph = "play"
	// StatusGlyphStop is a stop square inside a circle for closed states.
	StatusGlyphStop StatusGlyph = "stop"
	// StatusGlyphAlert is a warning triangle for unknown or attention states.
	StatusGlyphAlert StatusGlyph = "alert"
	// StatusGlyphInfo is an info mark for terminal non-error states.
	StatusGlyphInfo StatusGlyph = "info"
	// StatusGlyphMinus is a dash for muted or unspecified states.
	StatusGlyphMinus StatusGlyph = "minus"
)

// StatusIconSpec describes a status value rendered as a colored icon with tooltip.
type StatusIconSpec struct {
	// Label is the tooltip and accessible name for the status.
	Label string
	// ToneClass is a Tailwind text color utility for the icon.
	ToneClass string
	// Glyph selects the SVG path set rendered inside the icon.
	Glyph StatusGlyph
}

// EnabledStatusIcon maps an enabled flag to icon styling.
func EnabledStatusIcon(enabled bool) StatusIconSpec {
	if enabled {
		return StatusIconSpec{Label: enabledText(enabled), ToneClass: "text-success", Glyph: StatusGlyphCheck}
	}
	return StatusIconSpec{Label: enabledText(enabled), ToneClass: "text-base-content/40", Glyph: StatusGlyphX}
}

// SessionStateStatusIcon maps an agent session display state to icon styling.
func SessionStateStatusIcon(state string) StatusIconSpec {
	switch state {
	case "Active":
		return StatusIconSpec{Label: state, ToneClass: "text-success", Glyph: StatusGlyphCheck}
	case "Closed":
		return StatusIconSpec{Label: state, ToneClass: "text-base-content/40", Glyph: StatusGlyphStop}
	default:
		return StatusIconSpec{Label: state, ToneClass: "text-warning", Glyph: StatusGlyphAlert}
	}
}

// ScheduledTaskStateStatusIcon maps a scheduled task lifecycle state to icon styling.
func ScheduledTaskStateStatusIcon(state string) StatusIconSpec {
	switch state {
	case "active":
		return StatusIconSpec{Label: state, ToneClass: "text-success", Glyph: StatusGlyphPlay}
	case "paused":
		return StatusIconSpec{Label: state, ToneClass: "text-warning", Glyph: StatusGlyphPause}
	case "completed":
		return StatusIconSpec{Label: state, ToneClass: "text-info", Glyph: StatusGlyphCheck}
	case "failed", "cancelled", "missed":
		return StatusIconSpec{Label: state, ToneClass: "text-error", Glyph: StatusGlyphX}
	default:
		return StatusIconSpec{Label: state, ToneClass: "text-base-content/40", Glyph: StatusGlyphMinus}
	}
}

// ScheduledTaskRunStateStatusIcon maps a scheduled task run state to icon styling.
func ScheduledTaskRunStateStatusIcon(state string) StatusIconSpec {
	switch state {
	case "completed":
		return StatusIconSpec{Label: state, ToneClass: "text-success", Glyph: StatusGlyphCheck}
	case "running":
		return StatusIconSpec{Label: state, ToneClass: "text-warning", Glyph: StatusGlyphPlay}
	case "failed":
		return StatusIconSpec{Label: state, ToneClass: "text-error", Glyph: StatusGlyphX}
	default:
		return StatusIconSpec{Label: state, ToneClass: "text-base-content/40", Glyph: StatusGlyphMinus}
	}
}

// PipelineRuntimeStatusIcon maps a published pipeline enabled flag to Active/Paused icon styling.
func PipelineRuntimeStatusIcon(enabled bool) StatusIconSpec {
	if enabled {
		return StatusIconSpec{Label: "Active", ToneClass: "text-success", Glyph: StatusGlyphPlay}
	}
	return StatusIconSpec{Label: "Paused", ToneClass: "text-warning", Glyph: StatusGlyphPause}
}

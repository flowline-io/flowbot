package permission

// Result is the outcome of one permission evaluation.
type Result struct {
	Action            Action
	PermissionKey     string
	Pattern           string
	SuggestedPattern  string
	SuggestAlways     bool
	DoomLoopTriggered bool
	ExternalChecked   bool
}

// Evaluator resolves tool permissions from merged config and session state.
type Evaluator struct {
	config Config
}

// NewEvaluator builds an evaluator from merged permission config.
func NewEvaluator(config Config) *Evaluator {
	if config == nil {
		config = DefaultConfig()
	}
	return &Evaluator{config: config}
}

// Config returns the evaluator's rule set.
func (e *Evaluator) Config() Config {
	return e.config
}

// ResolveDoomLoop returns the configured doom_loop action for a loop-detection critical hit.
// Session always-grants for KeyDoomLoop are honored when session is non-nil.
func (e *Evaluator) ResolveDoomLoop(pattern string, session *SessionState) Result {
	result := Result{
		PermissionKey:     KeyDoomLoop,
		Pattern:           pattern,
		DoomLoopTriggered: true,
	}
	if session != nil && pattern != "" && session.MatchesGrant(KeyDoomLoop, pattern) {
		result.Action = ActionAllow
		return result
	}
	result.Action = e.resolveKey(KeyDoomLoop, "")
	if result.Action == ActionAsk && pattern != "" {
		if suggested, ok := SuggestedPattern(KeyDoomLoop, pattern, ParseBashCommand{}); ok {
			result.SuggestedPattern = suggested
			result.SuggestAlways = true
		}
	}
	return result
}

// EvaluateDenyOnly reports ActionDeny when a deny rule wins for the tool or
// external path. Session grants, bash-complex→ask, and external→ask
// escalations are skipped. Ask/allow resolutions become ActionAllow (not denied).
func (e *Evaluator) EvaluateDenyOnly(req Request) Result {
	inputs := ExtractInputs(req)
	result := Result{
		Action:        ActionAllow,
		PermissionKey: inputs.PermissionKey,
		Pattern:       inputs.Primary,
	}

	if e.resolveKey(inputs.PermissionKey, inputs.Primary) == ActionDeny {
		result.Action = ActionDeny
		return result
	}

	for _, path := range inputs.ExternalPaths {
		if e.resolveKey(KeyExternalDirectory, path) == ActionDeny {
			result.Action = ActionDeny
			result.PermissionKey = KeyExternalDirectory
			result.Pattern = path
			result.ExternalChecked = true
			return result
		}
	}
	if req.ExternalPath {
		if e.resolveKey(KeyExternalDirectory, inputs.Primary) == ActionDeny {
			result.Action = ActionDeny
			result.PermissionKey = KeyExternalDirectory
			result.Pattern = inputs.Primary
			result.ExternalChecked = true
			return result
		}
	}
	return result
}

// Evaluate resolves the action for one tool invocation.
func (e *Evaluator) Evaluate(req Request, session *SessionState) Result {
	inputs := ExtractInputs(req)
	result := Result{
		PermissionKey: inputs.PermissionKey,
		Pattern:       inputs.Primary,
	}

	if session != nil && session.MatchesGrant(inputs.PermissionKey, inputs.Primary) {
		result.Action = ActionAllow
		return result
	}

	if e.evaluateExternal(req, inputs, &result) {
		attachSuggestion(&result, inputs)
		return result
	}

	toolAction := e.resolveKey(inputs.PermissionKey, inputs.Primary)
	if inputs.Bash.HasChain {
		toolAction = Stricter(toolAction, ActionAsk)
	}
	if inputs.Bash.Complex {
		toolAction = Stricter(toolAction, ActionAsk)
	}
	result.Action = Stricter(result.Action, toolAction)
	result.PermissionKey = inputs.PermissionKey
	result.Pattern = inputs.Primary
	attachSuggestion(&result, inputs)
	return result
}

func (e *Evaluator) evaluateExternal(req Request, inputs ExtractedInputs, result *Result) bool {
	if len(inputs.ExternalPaths) == 0 && !req.ExternalPath {
		return false
	}
	result.ExternalChecked = true
	for _, path := range inputs.ExternalPaths {
		e.applyExternalRule(result, path)
	}
	if req.ExternalPath {
		e.applyExternalRule(result, inputs.Primary)
	}
	return result.Action == ActionDeny || result.Action == ActionAsk
}

func (e *Evaluator) applyExternalRule(result *Result, path string) {
	action := e.resolveKey(KeyExternalDirectory, path)
	result.Action = Stricter(result.Action, action)
	if action != ActionAllow {
		result.PermissionKey = KeyExternalDirectory
		result.Pattern = path
	}
}

func attachSuggestion(result *Result, inputs ExtractedInputs) {
	if result.Action != ActionAsk {
		return
	}
	pattern, ok := SuggestedPattern(result.PermissionKey, result.Pattern, inputs.Bash)
	if !ok {
		result.SuggestAlways = false
		return
	}
	if IsOverlyBroadPattern(pattern) {
		result.SuggestAlways = false
		return
	}
	result.SuggestedPattern = pattern
	result.SuggestAlways = true
}

func (e *Evaluator) resolveKey(key, input string) Action {
	rs, ok := e.config[key]
	if !ok {
		rs = e.config[KeyWildcard]
	}
	if len(rs.Patterns) == 0 {
		if rs.Default.Valid() {
			return rs.Default
		}
		return ActionAsk
	}
	var matched Action
	found := false
	for _, rule := range rs.Patterns {
		if MatchGlob(rule.Pattern, input) {
			matched = rule.Action
			found = true
		}
	}
	if found {
		return matched
	}
	if rs.Default.Valid() {
		return rs.Default
	}
	return ActionAsk
}

// EffectiveConfig merges defaults with user overrides for API responses.
func EffectiveConfig(user Config) Config {
	return Merge(DefaultConfig(), user)
}

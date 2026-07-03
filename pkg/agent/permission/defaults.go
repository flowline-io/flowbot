package permission

// KeyDoomLoop is the permission key for repeated identical tool calls.
const KeyDoomLoop = "doom_loop"

// KeyExternalDirectory is the permission key for workspace-external path access.
const KeyExternalDirectory = "external_directory"

// KeyWildcard applies to all permission keys when no specific rule matches.
const KeyWildcard = "*"

// KeyDelegate is the permission key for subagent delegation via the task tool.
const KeyDelegate = "delegate"

// KeySchedule is the permission key for scheduled task write operations.
const KeySchedule = "schedule"

// KeyScheduleRead is the permission key for listing scheduled tasks.
const KeyScheduleRead = "schedule_read"

// DefaultConfig returns OpenCode-style baseline rules used when the user has no overrides.
func DefaultConfig() Config {
	return Config{
		KeyWildcard: {Default: ActionAsk},
		"read": {
			Patterns: []PatternRule{
				{Pattern: "*", Action: ActionAllow},
				{Pattern: "*.env", Action: ActionDeny},
				{Pattern: "*.env.*", Action: ActionDeny},
				{Pattern: "*.env.example", Action: ActionAllow},
			},
		},
		"edit": {
			Patterns: []PatternRule{
				{Pattern: "*", Action: ActionAsk},
			},
		},
		"bash": {
			Patterns: []PatternRule{
				{Pattern: "*", Action: ActionAsk},
			},
		},
		"websearch":     {Default: ActionAsk},
		"skill":         {Default: ActionAllow},
		KeyDelegate:     {Default: ActionAsk},
		KeySchedule:     {Default: ActionAsk},
		KeyScheduleRead: {Default: ActionAllow},
		KeyDoomLoop:     {Default: ActionAsk},
		KeyExternalDirectory: {
			Patterns: []PatternRule{
				{Pattern: "*", Action: ActionAsk},
			},
		},
	}
}

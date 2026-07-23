package permission

// KeyDoomLoop is the permission key for repeated identical tool calls.
const KeyDoomLoop = "doom_loop"

// KeyExternalDirectory is the permission key for workspace-external path access.
const KeyExternalDirectory = "external_directory"

// KeyWildcard applies to all permission keys when no specific rule matches.
const KeyWildcard = "*"

// KeyDelegate is the permission key for subagent delegation via the delegate_subagent tool.
const KeyDelegate = "delegate"

// KeySchedule is the permission key for scheduled task write operations.
const KeySchedule = "schedule"

// KeyScheduleRead is the permission key for listing scheduled tasks.
const KeyScheduleRead = "schedule_read"

// KeyMemory is the permission key for memory fact and session-summary tools.
const KeyMemory = "memory"

// KeyKnowledge is the permission key for knowledge base search/read tools.
const KeyKnowledge = "knowledge"

// KeyTodo is the permission key for session todo checklist tools.
const KeyTodo = "todo"

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
		KeyKnowledge:    {Default: ActionAllow},
		KeyDelegate:     {Default: ActionAsk},
		KeySchedule:     {Default: ActionAsk},
		KeyScheduleRead: {Default: ActionAllow},
		KeyMemory: {
			Patterns: []PatternRule{
				{Pattern: "read", Action: ActionAllow},
				{Pattern: "list", Action: ActionAllow},
				{Pattern: "write", Action: ActionAsk},
			},
			Default: ActionAsk,
		},
		KeyTodo:     {Default: ActionAllow},
		KeyDoomLoop: {Default: ActionAsk},
		KeyExternalDirectory: {
			Patterns: []PatternRule{
				{Pattern: "*", Action: ActionAsk},
			},
		},
	}
}

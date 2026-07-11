package permission

// ScheduledRunOverlay returns stricter permission rules for autonomous scheduled task runs.
func ScheduledRunOverlay() Config {
	return Config{
		"bash":               {Default: ActionDeny},
		"edit":               {Default: ActionDeny},
		"websearch":          {Default: ActionDeny},
		KeyDelegate:          {Default: ActionDeny},
		KeySchedule:          {Default: ActionDeny},
		KeyExternalDirectory: {Default: ActionDeny},
		KeyMemory:            {Default: ActionAllow},
	}
}

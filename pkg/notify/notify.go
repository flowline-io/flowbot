package notify

var handlers map[string]Notifyer

func Register(id string, notifyer Notifyer) {
	if handlers == nil {
		handlers = make(map[string]Notifyer)
	}

	if notifyer == nil {
		panic("Register: notifyer is nil")
	}
	if _, dup := handlers[id]; dup {
		panic("Register: called twice for notifyer " + id)
	}
	handlers[id] = notifyer
}

func List() map[string]Notifyer {
	return handlers
}

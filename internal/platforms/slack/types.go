package slack

import "sync"

const ID = "slack"

var (
	threadContext = make(map[string]string)
	threadMutex   sync.RWMutex
)

func setThreadContext(channel, threadTs string) {
	threadMutex.Lock()
	defer threadMutex.Unlock()
	threadContext[channel] = threadTs
}

func getThreadContext(channel string) string {
	threadMutex.RLock()
	defer threadMutex.RUnlock()
	return threadContext[channel]
}

func clearThreadContext(channel string) {
	threadMutex.Lock()
	defer threadMutex.Unlock()
	delete(threadContext, channel)
}

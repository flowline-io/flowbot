package store

import "github.com/flowline-io/flowbot/internal/store/ent/gen"

// Query result caps used by domain stores (mirrors former postgres adapter limits).
var (
	defaultQueryMaxResults        = 1024
	defaultQueryMaxMessageResults = 100
	activeQueryMaxResults         = defaultQueryMaxResults
	activeQueryMaxMessageResults  = defaultQueryMaxMessageResults
)

// SetQueryLimits updates package-level query caps (called from adapter Open).
func SetQueryLimits(maxResults, maxMessageResults int) {
	if maxResults > 0 {
		activeQueryMaxResults = maxResults
	} else {
		activeQueryMaxResults = defaultQueryMaxResults
	}
	if maxMessageResults > 0 {
		activeQueryMaxMessageResults = maxMessageResults
	} else {
		activeQueryMaxMessageResults = defaultQueryMaxMessageResults
	}
}

func queryMaxResults() int {
	return activeQueryMaxResults
}

func queryMaxMessageResults() int {
	return activeQueryMaxMessageResults
}

// ClientFromDB returns the ent client from the global Database adapter.
// Prefer GetDB(*gen.Client) so BDD stubs that only implement GetDB work; calling
// GetClient on a nil embedded Adapter panics. Fall back to GetClient otherwise.
func ClientFromDB() *gen.Client {
	if Database == nil {
		return nil
	}
	if c, ok := Database.GetDB().(*gen.Client); ok && c != nil {
		return c
	}
	return Database.GetClient()
}

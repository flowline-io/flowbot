// Package traefik implements the Traefik reverse-proxy API provider.
package traefik

// Overview is Traefik /api/overview statistics.
type Overview struct {
	HTTP     *ProtocolStats `json:"http"`
	TCP      *ProtocolStats `json:"tcp"`
	UDP      *ProtocolStats `json:"udp"`
	Features map[string]any `json:"features"`
}

// ProtocolStats holds counts for a protocol family.
type ProtocolStats struct {
	Routers     map[string]int `json:"routers"`
	Services    map[string]int `json:"services"`
	Middlewares map[string]int `json:"middlewares"`
}

// Router is an HTTP router from Traefik.
type Router struct {
	Name        string   `json:"name"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	Status      string   `json:"status"`
	Provider    string   `json:"provider"`
	EntryPoints []string `json:"entryPoints"`
	Priority    int64    `json:"priority"`
}

// Service is an HTTP service from Traefik.
type Service struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Provider string `json:"provider"`
}

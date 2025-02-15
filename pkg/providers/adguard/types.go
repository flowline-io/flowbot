package adguard

// ServerStatus AdGuard Home server status and configuration
type ServerStatus struct {
	DnsAddresses               []string `json:"dns_addresses"`
	DnsPort                    int32    `json:"dns_port"`
	HttpPort                   int32    `json:"http_port"`
	ProtectionEnabled          bool     `json:"protection_enabled"`
	ProtectionDisabledDuration *int64   `json:"protection_disabled_duration,omitempty"`
	DhcpAvailable              *bool    `json:"dhcp_available,omitempty"`
	Running                    bool     `json:"running"`
	Version                    string   `json:"version"`
	Language                   string   `json:"language"`
}

// Stats Server statistics data
type Stats struct {
	// Time units
	TimeUnits *string `json:"time_units,omitempty"`
	// Total number of DNS queries
	NumDnsQueries *int32 `json:"num_dns_queries,omitempty"`
	// Number of requests blocked by filtering rules
	NumBlockedFiltering *int32 `json:"num_blocked_filtering,omitempty"`
	// Number of requests blocked by safebrowsing module
	NumReplacedSafebrowsing *int32 `json:"num_replaced_safebrowsing,omitempty"`
	// Number of requests blocked by safesearch module
	NumReplacedSafesearch *int32 `json:"num_replaced_safesearch,omitempty"`
	// Number of blocked adult websites
	NumReplacedParental *int32 `json:"num_replaced_parental,omitempty"`
	// Average time in seconds on processing a DNS request
	AvgProcessingTime *float32        `json:"avg_processing_time,omitempty"`
	TopQueriedDomains []TopArrayEntry `json:"top_queried_domains,omitempty"`
	TopClients        []TopArrayEntry `json:"top_clients,omitempty"`
	TopBlockedDomains []TopArrayEntry `json:"top_blocked_domains,omitempty"`
	// Total number of responses from each upstream.
	TopUpstreamsResponses []TopArrayEntry `json:"top_upstreams_responses,omitempty"`
	// Average processing time in seconds of requests from each upstream.
	TopUpstreamsAvgTime  []TopArrayEntry `json:"top_upstreams_avg_time,omitempty"`
	DnsQueries           []int32         `json:"dns_queries,omitempty"`
	BlockedFiltering     []int32         `json:"blocked_filtering,omitempty"`
	ReplacedSafebrowsing []int32         `json:"replaced_safebrowsing,omitempty"`
	ReplacedParental     []int32         `json:"replaced_parental,omitempty"`
}

// TopArrayEntry Represent the number of hits or time duration per key (url, domain, or client IP).
type TopArrayEntry struct {
	DomainOrIp           *float32 `json:"domain_or_ip,omitempty"`
	AdditionalProperties map[string]interface{}
}

package adguard

type StatusResponse struct {
	DnsAddresses               []string `json:"dns_addresses"`
	DnsPort                    int      `json:"dns_port"`
	HttpPort                   int      `json:"http_port"`
	ProtectionEnabled          bool     `json:"protection_enabled"`
	ProtectionDisabledDuration int      `json:"protection_disabled_duration"`
	DhcpAvailable              bool     `json:"dhcp_available"`
	Running                    bool     `json:"running"`
	Version                    string   `json:"version"`
	Language                   string   `json:"language"`
}

type StatisticsResponse struct {
	TimeUnits               string  `json:"time_units"`
	NumDnsQueries           int     `json:"num_dns_queries"`
	NumBlockedFiltering     int     `json:"num_blocked_filtering"`
	NumReplacedSafebrowsing int     `json:"num_replaced_safebrowsing"`
	NumReplacedSafesearch   int     `json:"num_replaced_safesearch"`
	NumReplacedParental     int     `json:"num_replaced_parental"`
	AvgProcessingTime       float64 `json:"avg_processing_time"`
	TopQueriedDomains       []struct {
		DomainOrIp      int `json:"domain_or_ip"`
		AdditionalProp1 int `json:"additionalProp1"`
		AdditionalProp2 int `json:"additionalProp2"`
		AdditionalProp3 int `json:"additionalProp3"`
	} `json:"top_queried_domains"`
	TopClients []struct {
		DomainOrIp      int `json:"domain_or_ip"`
		AdditionalProp1 int `json:"additionalProp1"`
		AdditionalProp2 int `json:"additionalProp2"`
		AdditionalProp3 int `json:"additionalProp3"`
	} `json:"top_clients"`
	TopBlockedDomains []struct {
		DomainOrIp      int `json:"domain_or_ip"`
		AdditionalProp1 int `json:"additionalProp1"`
		AdditionalProp2 int `json:"additionalProp2"`
		AdditionalProp3 int `json:"additionalProp3"`
	} `json:"top_blocked_domains"`
	TopUpstreamsResponses []struct {
		DomainOrIp      int `json:"domain_or_ip"`
		AdditionalProp1 int `json:"additionalProp1"`
		AdditionalProp2 int `json:"additionalProp2"`
		AdditionalProp3 int `json:"additionalProp3"`
	} `json:"top_upstreams_responses"`
	TopUpstreamsAvgTime []struct {
		DomainOrIp      int `json:"domain_or_ip"`
		AdditionalProp1 int `json:"additionalProp1"`
		AdditionalProp2 int `json:"additionalProp2"`
		AdditionalProp3 int `json:"additionalProp3"`
	} `json:"top_upstreams_avg_time"`
	DnsQueries           []int `json:"dns_queries"`
	BlockedFiltering     []int `json:"blocked_filtering"`
	ReplacedSafebrowsing []int `json:"replaced_safebrowsing"`
	ReplacedParental     []int `json:"replaced_parental"`
}

package adguard

import (
	"github.com/go-resty/resty/v2"
	"time"
)

const (
	ID          = "adguard_home"
	EndpointKey = "endpoint"
	UsernameKey = "username"
	PasswordKey = "password"
)

type AdGuardHome struct {
	c *resty.Client
}

func NewAdGuardHome(endpoint string, username string, password string) (*AdGuardHome, error) {
	v := &AdGuardHome{}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)
	v.c.SetBasicAuth(username, password)

	return v, nil
}

func (v *AdGuardHome) GetStatus() (*StatusResponse, error) {
	resp, err := v.c.R().
		SetResult(&StatusResponse{}).
		Get("/status")
	if err != nil {
		return nil, err
	}
	return resp.Result().(*StatusResponse), nil
}

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

func (v *AdGuardHome) GetStats() (*StatisticsResponse, error) {
	resp, err := v.c.R().
		SetResult(&StatisticsResponse{}).
		Get("/stats")
	if err != nil {
		return nil, err
	}
	return resp.Result().(*StatisticsResponse), nil
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

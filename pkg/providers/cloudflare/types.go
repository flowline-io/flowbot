package cloudflare

import "time"

type AnalyticResponse struct {
	Data struct {
		Viewer struct {
			Zones []struct {
				FirewallEventsAdaptive []struct {
					Action                string     `json:"action"`
					ClientRequestHTTPHost string     `json:"clientRequestHTTPHost"`
					Datetime              *time.Time `json:"datetime"`
					RayName               string     `json:"rayName"`
					UserAgent             string     `json:"userAgent"`
				} `json:"firewallEventsAdaptive"`
			} `json:"zones"`
		} `json:"viewer"`
	} `json:"data"`
	Errors interface{} `json:"errors"`
}

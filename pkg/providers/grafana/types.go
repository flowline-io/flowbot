// Package grafana implements the Grafana HTTP API provider.
package grafana

// Health is the Grafana /api/health response.
type Health struct {
	Commit   string `json:"commit"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

// Datasource is a Grafana data source.
type Datasource struct {
	ID   int64  `json:"id"`
	UID  string `json:"uid"`
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// DashboardHit is a Grafana search result for a dashboard.
type DashboardHit struct {
	UID       string   `json:"uid"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	Type      string   `json:"type"`
	Tags      []string `json:"tags"`
	FolderUID string   `json:"folderUid"`
}

// Package netalertx implements the NetAlertX network device monitoring provider.
package netalertx

// DevicesResponse is returned by GET /devices.
type DevicesResponse struct {
	Success bool     `json:"success"`
	Devices []Device `json:"devices"`
	Message string   `json:"message,omitzero"`
}

// Device is a NetAlertX network device record.
type Device struct {
	Name           string `json:"devName"`
	MAC            string `json:"devMAC"`
	MacAlt         string `json:"devMac"`
	IP             string `json:"devIP"`
	LastIP         string `json:"devLastIP"`
	Type           string `json:"devType"`
	Vendor         string `json:"devVendor"`
	Owner          string `json:"devOwner"`
	Status         string `json:"devStatus"`
	Favorite       int    `json:"devFavorite"`
	LastConnection string `json:"devLastConnection"`
}

// SearchResponse is returned by POST /devices/search.
type SearchResponse struct {
	Success bool     `json:"success"`
	Devices []Device `json:"devices"`
}

// Totals holds device counts: [all, connected, favorites, new, down, archived].
type Totals struct {
	All       int `json:"all"`
	Connected int `json:"connected"`
	Favorites int `json:"favorites"`
	New       int `json:"new"`
	Down      int `json:"down"`
	Archived  int `json:"archived"`
}

// Topology is the network topology graph.
type Topology struct {
	Nodes []TopologyNode `json:"nodes"`
	Links []TopologyLink `json:"links"`
}

// TopologyNode is a node in the network topology.
type TopologyNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Vendor string `json:"vendor"`
}

// TopologyLink is an edge in the network topology.
type TopologyLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Port   string `json:"port"`
}

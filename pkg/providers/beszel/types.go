// Package beszel implements the Beszel host-monitoring provider (PocketBase API).
package beszel

// AuthResponse is returned by PocketBase password authentication.
type AuthResponse struct {
	Token  string      `json:"token"`
	Record *UserRecord `json:"record"`
}

// UserRecord is the authenticated user record from PocketBase.
type UserRecord struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// SystemList is a paginated list of Beszel systems.
type SystemList struct {
	Page       int      `json:"page"`
	PerPage    int      `json:"perPage"`
	TotalItems int      `json:"totalItems"`
	TotalPages int      `json:"totalPages"`
	Items      []System `json:"items"`
}

// System is a Beszel monitored host.
type System struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Host   string `json:"host"`
	Port   string `json:"port"`
	Info   any    `json:"info"`
}

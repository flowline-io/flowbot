// Package trilium implements the Trilium Notes ETAPI provider.
package trilium

// MaxPageSize is the maximum number of items per page for list requests.
const MaxPageSize = 100

// Note represents a note in Trilium.
type Note struct {
	NoteID          string      `json:"noteId"`
	Title           string      `json:"title,omitempty"`
	Type            string      `json:"type,omitempty"`
	Mime            string      `json:"mime,omitempty"`
	IsProtected     bool        `json:"isProtected,omitempty"`
	BlobID          string      `json:"blobId,omitempty"`
	Attributes      []Attribute `json:"attributes,omitempty"`
	ParentNoteIDs   []string    `json:"parentNoteIds,omitempty"`
	ChildNoteIDs    []string    `json:"childNoteIds,omitempty"`
	ParentBranchIDs []string    `json:"parentBranchIds,omitempty"`
	ChildBranchIDs  []string    `json:"childBranchIds,omitempty"`
	DateCreated     string      `json:"dateCreated,omitempty"`
	DateModified    string      `json:"dateModified,omitempty"`
	UtcDateCreated  string      `json:"utcDateCreated,omitempty"`
	UtcDateModified string      `json:"utcDateModified,omitempty"`
}

// CreateNoteDef is the request body for creating a note via POST /create-note.
type CreateNoteDef struct {
	ParentNoteID   string `json:"parentNoteId"`
	Title          string `json:"title"`
	Type           string `json:"type"`
	Mime           string `json:"mime,omitempty"`
	Content        string `json:"content"`
	NotePosition   int    `json:"notePosition,omitempty"`
	Prefix         string `json:"prefix,omitempty"`
	IsExpanded     bool   `json:"isExpanded,omitempty"`
	NoteID         string `json:"noteId,omitempty"`
	BranchID       string `json:"branchId,omitempty"`
	DateCreated    string `json:"dateCreated,omitempty"`
	UtcDateCreated string `json:"utcDateCreated,omitempty"`
}

// Branch represents a branch (note placement in the tree).
type Branch struct {
	BranchID        string `json:"branchId,omitempty"`
	NoteID          string `json:"noteId,omitempty"`
	ParentNoteID    string `json:"parentNoteId,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	NotePosition    int    `json:"notePosition,omitempty"`
	IsExpanded      bool   `json:"isExpanded,omitempty"`
	UtcDateModified string `json:"utcDateModified,omitempty"`
}

// NoteWithBranch is the response from creating a note.
type NoteWithBranch struct {
	Note   Note   `json:"note"`
	Branch Branch `json:"branch"`
}

// Attribute represents a label or relation attached to a note.
type Attribute struct {
	AttributeID     string `json:"attributeId,omitempty"`
	NoteID          string `json:"noteId,omitempty"`
	Type            string `json:"type,omitempty"`
	Name            string `json:"name,omitempty"`
	Value           string `json:"value,omitempty"`
	Position        int    `json:"position,omitempty"`
	IsInheritable   bool   `json:"isInheritable,omitempty"`
	UtcDateModified string `json:"utcDateModified,omitempty"`
}

// CreateAttribute is the request body for creating an attribute.
type CreateAttribute struct {
	NoteID        string `json:"noteId"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Value         string `json:"value,omitempty"`
	Position      int    `json:"position,omitempty"`
	IsInheritable bool   `json:"isInheritable,omitempty"`
}

// SearchResponse is the response from searching notes.
type SearchResponse struct {
	Results   []Note     `json:"results"`
	DebugInfo *DebugInfo `json:"debugInfo,omitempty"`
}

// DebugInfo contains search query parsing debug information.
type DebugInfo struct {
	Query       string `json:"query,omitempty"`
	ParsedQuery string `json:"parsedQuery,omitempty"`
}

// AppInfo contains information about the running Trilium instance.
type AppInfo struct {
	AppVersion    string `json:"appVersion"`
	DBVersion     int    `json:"dbVersion"`
	SyncVersion   int    `json:"syncVersion"`
	BuildDate     string `json:"buildDate"`
	BuildRevision string `json:"buildRevision"`
	DataDirectory string `json:"dataDirectory"`
	InstanceName  string `json:"instanceName,omitempty"`
}

// ErrorResponse represents an ETAPI error.
type ErrorResponse struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// LoginRequest is the request body for POST /auth/login.
type LoginRequest struct {
	Password string `json:"password"`
}

// LoginResponse is the response from POST /auth/login.
type LoginResponse struct {
	AuthToken string `json:"authToken"`
}

// SearchParams holds optional query parameters for searching notes.
type SearchParams struct {
	// Search is the search query string (required).
	Search string
	// FastSearch enables fast search (fulltext doesn't look into content).
	FastSearch bool
	// IncludeArchivedNotes includes archived notes in results.
	IncludeArchivedNotes bool
	// AncestorNoteID limits search to a subtree.
	AncestorNoteID string
	// AncestorDepth defines depth constraint (e.g., "eq1", "lt4", "gt2").
	AncestorDepth string
	// OrderBy specifies the property/label to order by.
	OrderBy string
	// OrderDirection is "asc" or "desc".
	OrderDirection string
	// Limit is the maximum number of results.
	Limit int
	// Debug enables debug info in response.
	Debug bool
}

// BranchRequest is the request body for creating/updating a branch.
type BranchRequest struct {
	NoteID       string `json:"noteId"`
	ParentNoteID string `json:"parentNoteId"`
	Prefix       string `json:"prefix,omitempty"`
	NotePosition int    `json:"notePosition,omitempty"`
	IsExpanded   bool   `json:"isExpanded,omitempty"`
}

// PatchNoteRequest is the subset of Note fields that can be patched.
type PatchNoteRequest struct {
	Title          string `json:"title,omitempty"`
	Type           string `json:"type,omitempty"`
	Mime           string `json:"mime,omitempty"`
	BlobID         string `json:"blobId,omitempty"`
	DateCreated    string `json:"dateCreated,omitempty"`
	UtcDateCreated string `json:"utcDateCreated,omitempty"`
}

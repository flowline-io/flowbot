package nocodb

// PageInfo holds NocoDB list pagination metadata.
type PageInfo struct {
	TotalRows   int  `json:"totalRows"`
	Page        int  `json:"page"`
	PageSize    int  `json:"pageSize"`
	IsFirstPage bool `json:"isFirstPage"`
	IsLastPage  bool `json:"isLastPage"`
}

// Base is a NocoDB project/base.
type Base struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// BaseList is the envelope returned by list bases.
type BaseList struct {
	List     []Base   `json:"list"`
	PageInfo PageInfo `json:"pageInfo"`
}

// Column is a table column definition.
type Column struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	UIDT       string `json:"uidt"`
	ColumnName string `json:"column_name"`
}

// Table is a NocoDB table with optional columns.
type Table struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	BaseID  string   `json:"base_id"`
	Columns []Column `json:"columns"`
}

// TableList is the envelope returned by list tables.
type TableList struct {
	List     []Table  `json:"list"`
	PageInfo PageInfo `json:"pageInfo"`
}

// Record is a single table row. Field keys are column titles; Id is the primary key.
type Record map[string]any

// RecordList is the envelope returned by list records.
type RecordList struct {
	List     []Record `json:"list"`
	PageInfo PageInfo `json:"pageInfo"`
}

// ListRecordsQuery holds optional query parameters for listing records.
type ListRecordsQuery struct {
	Limit  int
	Offset int
	Where  string
	Sort   string
	Fields string
}

package nocodb

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNocoDB(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		token    string
		wantNil  bool
	}{
		{name: "constructor creates client", endpoint: "https://nocodb.example.com", token: "tok"},
		{name: "empty endpoint returns nil", endpoint: "", token: "tok", wantNil: true},
		{name: "endpoint with empty token", endpoint: "https://nocodb.example.com", token: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewNocoDB(tt.endpoint, tt.token)
			if tt.wantNil {
				assert.Nil(t, client)
				return
			}
			require.NotNil(t, client)
			assert.NotNil(t, client.c)
		})
	}
}

func TestNocoDB_ListBases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       any
		wantLen    int
		wantErr    bool
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			body: BaseList{
				List:     []Base{{ID: "b1", Title: "Home"}},
				PageInfo: PageInfo{TotalRows: 1, PageSize: 25, IsLastPage: true},
			},
			wantLen: 1,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       map[string]string{"msg": "error"},
			wantErr:    true,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       BaseList{List: []Base{}, PageInfo: PageInfo{IsLastPage: true}},
			wantLen:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/meta/bases", r.URL.Path)
				assert.Equal(t, "test-token", r.Header.Get("xc-token"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(tt.body)
			}))
			defer server.Close()

			client := NewNocoDB(server.URL, "test-token")
			got, err := client.ListBases(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Len(t, got.List, tt.wantLen)
		})
	}
}

func TestNocoDB_ListTables(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		baseID   string
		wantErr  bool
		status   int
		wantPath string
	}{
		{name: "successful list", baseID: "b1", status: http.StatusOK, wantPath: "/api/v2/meta/bases/b1/tables"},
		{name: "missing base id", baseID: "", wantErr: true},
		{name: "invalid base id chars", baseID: "b/1", wantErr: true},
		{name: "path escaped id", baseID: "b 1", status: http.StatusOK, wantPath: "/api/v2/meta/bases/b%201/tables"},
		{name: "not found", baseID: "missing", status: http.StatusNotFound, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantPath != "" {
					assert.Equal(t, tt.wantPath, r.URL.EscapedPath())
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(TableList{
					List: []Table{{ID: "t1", Title: "Tasks", BaseID: tt.baseID}},
				})
			}))
			defer server.Close()

			client := NewNocoDB(server.URL, "tok")
			got, err := client.ListTables(context.Background(), tt.baseID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.List, 1)
			assert.Equal(t, "t1", got.List[0].ID)
		})
	}
}

func TestNocoDB_GetTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tableID string
		wantErr bool
		status  int
	}{
		{name: "successful get", tableID: "t1", status: http.StatusOK},
		{name: "missing table id", tableID: "", wantErr: true},
		{name: "not found", tableID: "missing", status: http.StatusNotFound, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/meta/tables/"+tt.tableID, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(Table{
					ID: tt.tableID, Title: "Tasks",
					Columns: []Column{{ID: "c1", Title: "Name", UIDT: "SingleLineText"}},
				})
			}))
			defer server.Close()

			client := NewNocoDB(server.URL, "tok")
			got, err := client.GetTable(context.Background(), tt.tableID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.status == http.StatusNotFound {
					se, ok := AsStatusError(err)
					require.True(t, ok)
					assert.Equal(t, http.StatusNotFound, se.Status)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "Tasks", got.Title)
			assert.Len(t, got.Columns, 1)
		})
	}
}

func TestNocoDB_ListRecords(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tableID string
		query   ListRecordsQuery
		wantErr bool
		status  int
	}{
		{
			name:    "successful list with filters",
			tableID: "t1",
			query:   ListRecordsQuery{Limit: 10, Offset: 5, Where: "(Name,eq,a)", Sort: "Name", Fields: "Name"},
			status:  http.StatusOK,
		},
		{name: "missing table id", tableID: "", wantErr: true},
		{name: "negative limit", tableID: "t1", query: ListRecordsQuery{Limit: -1}, wantErr: true},
		{name: "limit too large", tableID: "t1", query: ListRecordsQuery{Limit: maxListLimit + 1}, wantErr: true},
		{name: "server error", tableID: "t1", status: http.StatusBadGateway, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/tables/"+tt.tableID+"/records", r.URL.Path)
				if tt.query.Limit > 0 {
					assert.Equal(t, "10", r.URL.Query().Get("limit"))
				}
				if tt.query.Offset > 0 {
					assert.Equal(t, "5", r.URL.Query().Get("offset"))
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(RecordList{
					List: []Record{{"Id": 1, "Name": "a"}},
				})
			}))
			defer server.Close()

			client := NewNocoDB(server.URL, "tok")
			got, err := client.ListRecords(context.Background(), tt.tableID, tt.query)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.List, 1)
		})
	}
}

func TestNocoDB_GetRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		tableID  string
		recordID string
		wantErr  bool
		status   int
	}{
		{name: "successful get", tableID: "t1", recordID: "1", status: http.StatusOK},
		{name: "missing table id", tableID: "", recordID: "1", wantErr: true},
		{name: "missing record id", tableID: "t1", recordID: "", wantErr: true},
		{name: "not found", tableID: "t1", recordID: "99", status: http.StatusNotFound, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/tables/"+tt.tableID+"/records/"+tt.recordID, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_ = sonic.ConfigDefault.NewEncoder(w).Encode(Record{"Id": 1, "Name": "row"})
			}))
			defer server.Close()

			client := NewNocoDB(server.URL, "tok")
			got, err := client.GetRecord(context.Background(), tt.tableID, tt.recordID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "row", got["Name"])
		})
	}
}

func TestNocoDB_CreateRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tableID string
		fields  map[string]any
		body    string
		status  int
		wantID  string
		wantErr bool
	}{
		{
			name:    "create returns object",
			tableID: "t1",
			fields:  map[string]any{"Name": "new"},
			body:    `{"Id":9,"Name":"new"}`,
			status:  http.StatusOK,
			wantID:  "9",
		},
		{
			name:    "create returns array",
			tableID: "t1",
			fields:  map[string]any{"Name": "new"},
			body:    `[{"Id":10,"Name":"new"}]`,
			status:  http.StatusOK,
			wantID:  "10",
		},
		{
			name:    "empty array is error",
			tableID: "t1",
			fields:  map[string]any{"Name": "new"},
			body:    `[]`,
			status:  http.StatusOK,
			wantErr: true,
		},
		{name: "missing fields", tableID: "t1", fields: nil, wantErr: true},
		{name: "missing table id", tableID: "", fields: map[string]any{"Name": "a"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v2/tables/"+tt.tableID+"/records", r.URL.Path)
				raw, _ := io.ReadAll(r.Body)
				var req map[string]any
				_ = sonic.Unmarshal(raw, &req)
				assert.Equal(t, tt.fields["Name"], req["Name"])
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewNocoDB(server.URL, "tok")
			got, err := client.CreateRecord(context.Background(), tt.tableID, tt.fields)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, recordID(got))
		})
	}
}

func TestNocoDB_UpdateAndDeleteRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		op         string
		tableID    string
		fields     map[string]any
		id         string
		wantErr    bool
		status     int
		wantIDType string // "number" or "string"
	}{
		{name: "update success numeric id", op: "update", tableID: "t1", fields: map[string]any{"Id": "1", "Name": "x"}, status: http.StatusOK, wantIDType: "number"},
		{name: "update missing id", op: "update", tableID: "t1", fields: map[string]any{"Name": "x"}, wantErr: true},
		{name: "delete success numeric id", op: "delete", tableID: "t1", id: "1", status: http.StatusOK, wantIDType: "number"},
		{name: "delete custom string id", op: "delete", tableID: "t1", id: "abc-uuid", status: http.StatusOK, wantIDType: "string"},
		{name: "delete missing id", op: "delete", tableID: "t1", id: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/tables/"+tt.tableID+"/records", r.URL.Path)
				raw, _ := io.ReadAll(r.Body)
				switch tt.op {
				case "update":
					var body map[string]any
					if assert.NoError(t, sonic.Unmarshal(raw, &body)) {
						assertIDType(t, body["Id"], tt.wantIDType)
					}
				case "delete":
					var body []map[string]any
					if assert.NoError(t, sonic.Unmarshal(raw, &body)) && assert.Len(t, body, 1) {
						assertIDType(t, body[0]["Id"], tt.wantIDType)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"Id":1,"Name":"x"}`))
			}))
			defer server.Close()

			client := NewNocoDB(server.URL, "tok")
			var err error
			switch tt.op {
			case "update":
				_, err = client.UpdateRecord(context.Background(), tt.tableID, tt.fields)
			case "delete":
				err = client.DeleteRecord(context.Background(), tt.tableID, tt.id)
			}
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestEncodeRecordID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		id      string
		want    any
		wantErr bool
	}{
		{name: "numeric", id: "42", want: int64(42)},
		{name: "uuid string", id: "abc-uuid", want: "abc-uuid"},
		{name: "empty", id: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := encodeRecordID(tt.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func assertIDType(t *testing.T, v any, wantType string) {
	t.Helper()
	switch wantType {
	case "number":
		_, ok := v.(float64)
		assert.True(t, ok, "expected JSON number, got %T", v)
	case "string":
		_, ok := v.(string)
		assert.True(t, ok, "expected JSON string, got %T", v)
	}
}

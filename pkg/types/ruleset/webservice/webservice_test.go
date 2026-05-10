package webservice

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestRule_ID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{
			name:   "GET method",
			method: "GET",
			path:   "/example",
			want:   "GET_/example",
		},
		{
			name:   "POST method",
			method: "POST",
			path:   "/upload",
			want:   "POST_/upload",
		},
		{
			name:   "complex path",
			method: "GET",
			path:   "/api/v1/users/:id/posts",
			want:   "GET_/api/v1/users/:id/posts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := Rule{Method: tt.method, Path: tt.path}
			assert.Equal(t, tt.want, r.ID())
		})
	}
}

func TestRule_TYPE(t *testing.T) {
	t.Parallel()
	t.Run("rule type", func(t *testing.T) {
		t.Parallel()
		r := Rule{Method: "GET", Path: "/example"}
		assert.Equal(t, types.WebserviceRule, r.TYPE())
	})
}

func TestGet(t *testing.T) {
	t.Parallel()
	t.Run("Get helper", func(t *testing.T) {
		t.Parallel()
		r := Get("/api/data", nil)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/data", r.Path)
		assert.Nil(t, r.Function)
		assert.Empty(t, r.Option)
	})
}

func TestPost(t *testing.T) {
	t.Parallel()
	t.Run("Post helper", func(t *testing.T) {
		t.Parallel()
		r := Post("/api/submit", nil)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/submit", r.Path)
	})
}

func TestPut(t *testing.T) {
	t.Parallel()
	t.Run("Put helper", func(t *testing.T) {
		t.Parallel()
		r := Put("/api/update", nil)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/update", r.Path)
	})
}

func TestPatch(t *testing.T) {
	t.Parallel()
	t.Run("Patch helper", func(t *testing.T) {
		t.Parallel()
		r := Patch("/api/archive", nil)
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/api/archive", r.Path)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()
	t.Run("Delete helper", func(t *testing.T) {
		t.Parallel()
		r := Delete("/api/remove", nil)
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/remove", r.Path)
	})
}

func TestRuleset_Creation(t *testing.T) {
	t.Parallel()
	t.Run("ruleset creation", func(t *testing.T) {
		t.Parallel()
		rules := Ruleset{
			Get("/query", nil),
			Post("/submit", nil),
			Put("/update", nil),
			Patch("/patch", nil),
			Delete("/delete", nil),
		}
		assert.Len(t, rules, 5)
		assert.Equal(t, "GET_/query", rules[0].ID())
		assert.Equal(t, "POST_/submit", rules[1].ID())
		assert.Equal(t, "PUT_/update", rules[2].ID())
		assert.Equal(t, "PATCH_/patch", rules[3].ID())
		assert.Equal(t, "DELETE_/delete", rules[4].ID())
	})
}

func TestRuleset_Empty(t *testing.T) {
	t.Parallel()
	t.Run("empty ruleset", func(t *testing.T) {
		t.Parallel()
		rules := Ruleset{}
		assert.Empty(t, rules)
	})
}

func TestRule_AllMethodTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		create func(string, any, ...any) Rule
		method string
	}{
		{func(p string, _ any, _ ...any) Rule { return Get(p, nil) }, "GET"},
		{func(p string, _ any, _ ...any) Rule { return Post(p, nil) }, "POST"},
		{func(p string, _ any, _ ...any) Rule { return Put(p, nil) }, "PUT"},
		{func(p string, _ any, _ ...any) Rule { return Patch(p, nil) }, "PATCH"},
		{func(p string, _ any, _ ...any) Rule { return Delete(p, nil) }, "DELETE"},
	}

	for _, m := range tests {
		t.Run(m.method, func(t *testing.T) {
			t.Parallel()
			r := m.create("/test", nil)
			assert.Equal(t, m.method, r.Method)
			assert.Equal(t, types.WebserviceRule, r.TYPE())
		})
	}
}

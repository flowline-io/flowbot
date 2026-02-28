package webservice

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRule_ID(t *testing.T) {
	r := Rule{Method: "GET", Path: "/example"}
	assert.Equal(t, "GET_/example", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	r := Rule{Method: "GET", Path: "/example"}
	assert.Equal(t, types.WebserviceRule, r.TYPE())
}

func TestRule_ID_Post(t *testing.T) {
	r := Rule{Method: "POST", Path: "/upload"}
	assert.Equal(t, "POST_/upload", r.ID())
}

func TestGet(t *testing.T) {
	r := Get("/api/data", nil)
	assert.Equal(t, "GET", r.Method)
	assert.Equal(t, "/api/data", r.Path)
	assert.Nil(t, r.Function)
	assert.Empty(t, r.Option)
}

func TestPost(t *testing.T) {
	r := Post("/api/submit", nil)
	assert.Equal(t, "POST", r.Method)
	assert.Equal(t, "/api/submit", r.Path)
}

func TestPut(t *testing.T) {
	r := Put("/api/update", nil)
	assert.Equal(t, "PUT", r.Method)
	assert.Equal(t, "/api/update", r.Path)
}

func TestDelete(t *testing.T) {
	r := Delete("/api/remove", nil)
	assert.Equal(t, "DELETE", r.Method)
	assert.Equal(t, "/api/remove", r.Path)
}

func TestRuleset_Creation(t *testing.T) {
	rules := Ruleset{
		Get("/query", nil),
		Post("/submit", nil),
		Put("/update", nil),
		Delete("/delete", nil),
	}
	assert.Len(t, rules, 4)
	assert.Equal(t, "GET_/query", rules[0].ID())
	assert.Equal(t, "POST_/submit", rules[1].ID())
	assert.Equal(t, "PUT_/update", rules[2].ID())
	assert.Equal(t, "DELETE_/delete", rules[3].ID())
}

func TestRuleset_Empty(t *testing.T) {
	rules := Ruleset{}
	assert.Len(t, rules, 0)
}

func TestRule_ID_ComplexPath(t *testing.T) {
	r := Rule{Method: "GET", Path: "/api/v1/users/:id/posts"}
	assert.Equal(t, "GET_/api/v1/users/:id/posts", r.ID())
}

func TestRule_AllMethodTypes(t *testing.T) {
	methods := []struct {
		create func(string, interface{}, ...interface{}) Rule
		method string
	}{
		{func(p string, _ interface{}, _ ...interface{}) Rule { return Get(p, nil) }, "GET"},
		{func(p string, _ interface{}, _ ...interface{}) Rule { return Post(p, nil) }, "POST"},
		{func(p string, _ interface{}, _ ...interface{}) Rule { return Put(p, nil) }, "PUT"},
		{func(p string, _ interface{}, _ ...interface{}) Rule { return Delete(p, nil) }, "DELETE"},
	}

	for _, m := range methods {
		t.Run(m.method, func(t *testing.T) {
			r := m.create("/test", nil)
			assert.Equal(t, m.method, r.Method)
			assert.Equal(t, types.WebserviceRule, r.TYPE())
		})
	}
}

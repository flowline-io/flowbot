package flows

import (
	"testing"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestSimpleTemplateRenderer_Render(t *testing.T) {
	r := NewSimpleTemplateRenderer()
	vars := types.KV{"name": "Alice", "n": 3}

	require.Equal(t, "hi Alice", r.Render("hi {{name}}", vars))
	require.Equal(t, "n=3", r.Render("n={{n}}", vars))
	require.Equal(t, "nochange", r.Render("nochange", vars))
}

func TestEngine_prepareParameters_RenderNested(t *testing.T) {
	e := &Engine{rnd: NewSimpleTemplateRenderer()}

	params := model.JSON{
		"text": "hello {{name}}",
		"obj": map[string]any{
			"inner": "n={{n}}",
		},
		"arr": []any{"{{name}}", 1},
	}

	out := e.prepareParameters(params, types.KV{"name": "Bob", "n": 7})
	require.Equal(t, "hello Bob", out["text"])

	obj, ok := out["obj"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "n=7", obj["inner"])

	arr, ok := out["arr"].([]any)
	require.True(t, ok)
	require.Equal(t, "Bob", arr[0])
	require.Equal(t, 1.0, arr[1]) // JSON roundtrip turns ints into float64
}

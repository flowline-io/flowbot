package partials

import "testing"

func TestFunctionCallURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		fn     string
		origin string
		want   string
	}{
		{name: "path only", fn: "demo", origin: "", want: "/service/functions/call/demo"},
		{name: "absolute origin", fn: "demo", origin: "http://127.0.0.1:6060", want: "http://127.0.0.1:6060/service/functions/call/demo"},
		{name: "origin trailing slash", fn: "demo", origin: "http://127.0.0.1:6060/", want: "http://127.0.0.1:6060/service/functions/call/demo"},
		{name: "empty name", fn: "  ", origin: "http://x", want: ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FunctionCallURL(tt.fn, tt.origin)
			if got != tt.want {
				t.Fatalf("FunctionCallURL(%q, %q) = %q, want %q", tt.fn, tt.origin, got, tt.want)
			}
		})
	}
}

func TestFunctionCallVersionURL(t *testing.T) {
	t.Parallel()
	got := FunctionCallVersionURL("demo", 2, "http://127.0.0.1:6060")
	want := "http://127.0.0.1:6060/service/functions/call/demo/v/2"
	if got != want {
		t.Fatalf("FunctionCallVersionURL = %q, want %q", got, want)
	}
}

func TestFunctionCallVersionPath(t *testing.T) {
	t.Parallel()
	got := FunctionCallVersionPath("demo", 2)
	want := "/service/functions/call/demo/v/2"
	if got != want {
		t.Fatalf("FunctionCallVersionPath = %q, want %q", got, want)
	}
}

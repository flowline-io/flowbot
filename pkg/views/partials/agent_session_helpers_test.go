package partials

import (
	"strings"
	"testing"
)

func TestEntryPayloadPreview(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "empty payload returns empty", payload: "", want: ""},
		{name: "short payload unchanged", payload: `{"role":"user"}`, want: `{"role":"user"}`},
		{name: "long payload truncated", payload: `{"content":"` + strings.Repeat("x", 200) + `"}`, want: "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entryPayloadPreview(tt.payload)
			if tt.name == "long payload truncated" {
				if len(got) <= 120 || got[len(got)-3:] != "..." {
					t.Fatalf("want truncated preview ending with ..., got len=%d %q", len(got), got)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

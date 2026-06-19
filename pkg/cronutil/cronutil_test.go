package cronutil

import (
	"testing"
	"time"
)

func TestValidateExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{name: "valid daily", spec: "0 9 * * *", wantErr: false},
		{name: "valid descriptor", spec: "@daily", wantErr: false},
		{name: "empty", spec: "", wantErr: true},
		{name: "invalid", spec: "not a cron", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateExpr(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateExpr() error = %v", err)
			}
		})
	}
}

func TestNextRun(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, 6, 20, 8, 30, 0, 0, time.UTC)
	next, err := NextRun("0 9 * * *", from)
	if err != nil {
		t.Fatalf("NextRun() error = %v", err)
	}
	want := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("NextRun() = %v, want %v", next, want)
	}
}

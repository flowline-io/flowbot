package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNocodbCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "nocodb command has expected subcommands"},
		{name: "nocodb subcommands are wired"},
		{name: "nocodb records has nested commands"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := NocodbCommand()
			require.Equal(t, "nocodb", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "bases")
			require.Contains(t, subNames, "tables")
			require.Contains(t, subNames, "table")
			require.Contains(t, subNames, "records")
			require.Contains(t, subNames, "health")

			records := findSubcommand(cmd, "records")
			require.NotNil(t, records)
			recordSubs := subcommandNames(records)
			require.Contains(t, recordSubs, "list")
			require.Contains(t, recordSubs, "get")
			require.Contains(t, recordSubs, "create")
			require.Contains(t, recordSubs, "update")
			require.Contains(t, recordSubs, "delete")
		})
	}
}

func TestNocodbRequiredFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		parent   string
		child    string
		flagName string
	}{
		{name: "tables requires base-id", parent: "tables", flagName: "base-id"},
		{name: "table requires table-id", parent: "table", flagName: "table-id"},
		{name: "records list requires table-id", parent: "records", child: "list", flagName: "table-id"},
		{name: "records create requires fields", parent: "records", child: "create", flagName: "fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := NocodbCommand()
			cmd := findSubcommand(root, tt.parent)
			require.NotNil(t, cmd)
			if tt.child != "" {
				cmd = findSubcommand(cmd, tt.child)
				require.NotNil(t, cmd)
			}
			require.NotNil(t, cmd.Flags().Lookup(tt.flagName))
		})
	}
}

func TestParseNocoFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		wantKey string
	}{
		{name: "valid object", raw: `{"Name":"Alice"}`, wantKey: "Name"},
		{name: "empty string", raw: "", wantErr: true},
		{name: "invalid json", raw: `{`, wantErr: true},
		{name: "empty object", raw: `{}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseNocoFields(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Contains(t, got, tt.wantKey)
		})
	}
}

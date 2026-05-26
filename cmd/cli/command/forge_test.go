package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForgeCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "forge command has correct use and subcommands"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ForgeCommand()
			require.Equal(t, "forge", cmd.Use)
			require.True(t, cmd.HasSubCommands())
			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "user")
			require.Contains(t, subNames, "repo")
			require.Contains(t, subNames, "issues")
			require.Contains(t, subNames, "issue")
			require.Contains(t, subNames, "diff")
			require.Contains(t, subNames, "file")
		})
	}
}

func TestForgeRepoArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "zero args rejected", args: nil, want: false},
		{name: "one arg rejected", args: []string{"owner"}, want: false},
		{name: "two args accepted", args: []string{"owner", "repo"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ForgeCommand()
			repoCmd := findSubcommand(cmd, "repo")
			require.NotNil(t, repoCmd)
			err := repoCmd.Args(repoCmd, tt.args)
			if tt.want {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestForgeIssueArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "zero args rejected", args: nil, want: false},
		{name: "two args rejected", args: []string{"owner", "repo"}, want: false},
		{name: "three args accepted", args: []string{"owner", "repo", "1"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ForgeCommand()
			issueCmd := findSubcommand(cmd, "issue")
			require.NotNil(t, issueCmd)
			err := issueCmd.Args(issueCmd, tt.args)
			if tt.want {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestForgeDiffArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "zero args rejected", args: nil, want: false},
		{name: "two args rejected", args: []string{"owner", "repo"}, want: false},
		{name: "three args accepted", args: []string{"owner", "repo", "abc123"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ForgeCommand()
			diffCmd := findSubcommand(cmd, "diff")
			require.NotNil(t, diffCmd)
			err := diffCmd.Args(diffCmd, tt.args)
			if tt.want {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestForgeFileArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "zero args rejected", args: nil, want: false},
		{name: "three args rejected", args: []string{"owner", "repo", "abc123"}, want: false},
		{name: "four args accepted", args: []string{"owner", "repo", "abc123", "main.go"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ForgeCommand()
			fileCmd := findSubcommand(cmd, "file")
			require.NotNil(t, fileCmd)
			err := fileCmd.Args(fileCmd, tt.args)
			if tt.want {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestForgeOutputFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		subcommand string
	}{
		{name: "user has output flag", subcommand: "user"},
		{name: "repo has output flag", subcommand: "repo"},
		{name: "issues has output flag", subcommand: "issues"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ForgeCommand()
			sub := findSubcommand(cmd, tt.subcommand)
			require.NotNil(t, sub)
			flag := sub.Flags().Lookup("output")
			require.NotNil(t, flag)
		})
	}
}

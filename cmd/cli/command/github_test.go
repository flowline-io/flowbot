package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGithubCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "github command has correct use and subcommands"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := GithubCommand()
			require.Equal(t, "github", cmd.Use)
			require.True(t, cmd.HasSubCommands())
			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "user")
			require.Contains(t, subNames, "user-by-login")
			require.Contains(t, subNames, "repo")
			require.Contains(t, subNames, "issues")
			require.Contains(t, subNames, "issue")
			require.Contains(t, subNames, "diff")
			require.Contains(t, subNames, "file")
			require.Contains(t, subNames, "notifications")
			require.Contains(t, subNames, "releases")
		})
	}
}

func TestGithubRepoArgs(t *testing.T) {
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
			cmd := GithubCommand()
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

func TestGithubReleasesArgs(t *testing.T) {
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
			cmd := GithubCommand()
			releasesCmd := findSubcommand(cmd, "releases")
			require.NotNil(t, releasesCmd)
			err := releasesCmd.Args(releasesCmd, tt.args)
			if tt.want {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGithubUserByLoginArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "zero args rejected", args: nil, want: false},
		{name: "login accepted", args: []string{"someone"}, want: true},
		{name: "two args rejected", args: []string{"a", "b"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := GithubCommand()
			userByLoginCmd := findSubcommand(cmd, "user-by-login")
			require.NotNil(t, userByLoginCmd)
			err := userByLoginCmd.Args(userByLoginCmd, tt.args)
			if tt.want {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGithubOutputFlags(t *testing.T) {
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
			cmd := GithubCommand()
			sub := findSubcommand(cmd, tt.subcommand)
			require.NotNil(t, sub)
			flag := sub.Flags().Lookup("output")
			require.NotNil(t, flag)
		})
	}
}

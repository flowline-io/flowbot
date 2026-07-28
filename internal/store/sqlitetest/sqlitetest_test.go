package sqlitetest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenClientAppliesSchema(t *testing.T) {
	t.Parallel()
	client := OpenClient(t, t.Name())
	n, err := client.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, n)

	// Second open must reuse cached DDL (and remain isolated).
	client2 := OpenClient(t, t.Name()+"_b")
	n, err = client2.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, n)
}

func TestSplitSQLStatements(t *testing.T) {
	t.Parallel()
	got := splitSQLStatements("CREATE TABLE a(id integer);\nCREATE TABLE b(id integer);\n")
	require.Equal(t, []string{
		"CREATE TABLE a(id integer)",
		"CREATE TABLE b(id integer)",
	}, got)
}

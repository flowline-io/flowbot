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

	// Same logical name must still isolate (unique suffix); cached DDL is reused.
	client2 := OpenClient(t, t.Name())
	n, err = client2.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, n)
}

func TestOpenClientSameNameDoesNotCollide(t *testing.T) {
	t.Parallel()
	const shared = "ent"
	c1 := OpenClient(t, shared)
	c2 := OpenClient(t, shared)
	require.NoError(t, c1.User.Create().SetID(1).SetFlag("a").SetName("a").Exec(context.Background()))
	n, err := c2.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, n, "second OpenClient with same hint must be a separate database")
}

func TestSplitSQLStatements(t *testing.T) {
	t.Parallel()
	got := splitSQLStatements("CREATE TABLE a(id integer);\nCREATE TABLE b(id integer);\n")
	require.Equal(t, []string{
		"CREATE TABLE a(id integer)",
		"CREATE TABLE b(id integer)",
	}, got)
}

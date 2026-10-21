package mcp

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/confluence"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/capability/devops"
	"github.com/flowline-io/flowbot/pkg/capability/drone"
	"github.com/flowline-io/flowbot/pkg/capability/email"
	"github.com/flowline-io/flowbot/pkg/capability/fireflyiii"
	"github.com/flowline-io/flowbot/pkg/capability/functions"
	"github.com/flowline-io/flowbot/pkg/capability/gitea"
	"github.com/flowline-io/flowbot/pkg/capability/github"
	"github.com/flowline-io/flowbot/pkg/capability/kanboard"
	"github.com/flowline-io/flowbot/pkg/capability/karakeep"
	"github.com/flowline-io/flowbot/pkg/capability/memos"
	"github.com/flowline-io/flowbot/pkg/capability/miniflux"
	"github.com/flowline-io/flowbot/pkg/capability/nocodb"
	"github.com/flowline-io/flowbot/pkg/capability/transmission"
	"github.com/flowline-io/flowbot/pkg/capability/trello"
	"github.com/flowline-io/flowbot/pkg/capability/trilium"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

func TestInputSchemaContract(t *testing.T) {
	t.Parallel()
	specs := advertisedCatalogSpecs()
	got := make([]string, 0, len(specs))
	descs := make([]hub.Descriptor, 0, len(specs))
	for _, spec := range specs {
		got = append(got, string(spec.Type))
		descs = append(descs, capability.DescriptorFromSpec(spec))
	}
	slices.Sort(got)
	assert.Equal(t, catalogSpecDirs(t), got)

	tools := BuildCatalog(descs, nil, nil)
	require.NotEmpty(t, tools)
	for _, tool := range tools {
		err := pkgmcp.ValidateInputSchema(schemaFromParams(tool.Input))
		assert.NoError(t, err, tool.Name)
	}
}

func advertisedCatalogSpecs() []capability.Spec {
	return []capability.Spec{
		confluence.CatalogSpec(),
		core.CatalogSpec(),
		devops.CatalogSpec(),
		drone.CatalogSpec(),
		email.CatalogSpec(),
		fireflyiii.CatalogSpec(),
		functions.CatalogSpec(),
		gitea.CatalogSpec(),
		github.CatalogSpec(),
		kanboard.CatalogSpec(),
		karakeep.CatalogSpec(),
		memos.CatalogSpec(),
		miniflux.CatalogSpec(),
		nocodb.CatalogSpec(),
		transmission.CatalogSpec(),
		trello.CatalogSpec(),
		trilium.CatalogSpec(),
	}
}

func catalogSpecDirs(t *testing.T) []string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "pkg", "capability")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, entry.Name(), "register.go"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			require.NoError(t, err)
		}
		if bytes.Contains(body, []byte("func CatalogSpec(")) {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names
}

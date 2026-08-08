package skills

import (
	"cmp"
	"slices"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/capability/devops"
	"github.com/flowline-io/flowbot/pkg/capability/email"
	"github.com/flowline-io/flowbot/pkg/capability/fireflyiii"
	"github.com/flowline-io/flowbot/pkg/capability/gateway"
	"github.com/flowline-io/flowbot/pkg/capability/gitea"
	"github.com/flowline-io/flowbot/pkg/capability/github"
	"github.com/flowline-io/flowbot/pkg/capability/kanboard"
	"github.com/flowline-io/flowbot/pkg/capability/karakeep"
	"github.com/flowline-io/flowbot/pkg/capability/memos"
	"github.com/flowline-io/flowbot/pkg/capability/miniflux"
	"github.com/flowline-io/flowbot/pkg/capability/nocodb"
	"github.com/flowline-io/flowbot/pkg/capability/transmission"
	"github.com/flowline-io/flowbot/pkg/capability/trilium"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// capOpDoc is one capability operation for workflow documentation.
type capOpDoc struct {
	Action      string
	Name        string
	Description string
	Mutation    bool
	Inputs      []hub.ParamDef
}

// capDoc is one capability type for workflow documentation.
type capDoc struct {
	Type        string
	Description string
	Ops         []capOpDoc
}

// workflowCapabilityCatalog returns CapCore + provider catalogs for workflow skill docs.
func workflowCapabilityCatalog() []capDoc {
	specs := []capability.Spec{
		core.CatalogSpec(),
		gateway.CatalogSpec(),
		karakeep.CatalogSpec(),
		kanboard.CatalogSpec(),
		miniflux.CatalogSpec(),
		memos.CatalogSpec(),
		trilium.CatalogSpec(),
		fireflyiii.CatalogSpec(),
		transmission.CatalogSpec(),
		email.CatalogSpec(),
		nocodb.CatalogSpec(),
		devops.CatalogSpec(),
		gitea.CatalogSpec(),
		github.CatalogSpec(),
	}
	docs := make([]capDoc, 0, len(specs))
	for _, s := range specs {
		ops := make([]capOpDoc, 0, len(s.Ops))
		for _, op := range s.Ops {
			ops = append(ops, capOpDoc{
				Action:      "capability:" + string(s.Type) + "." + op.Name,
				Name:        op.Name,
				Description: op.Description,
				Mutation:    op.Mutation,
				Inputs:      op.Input,
			})
		}
		slices.SortFunc(ops, func(a, b capOpDoc) int {
			return cmp.Compare(a.Name, b.Name)
		})
		docs = append(docs, capDoc{
			Type:        string(s.Type),
			Description: s.Description,
			Ops:         ops,
		})
	}
	slices.SortFunc(docs, func(a, b capDoc) int {
		return cmp.Compare(a.Type, b.Type)
	})
	return docs
}

package web

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var commandPaletteWebserviceRules = []webservice.Rule{
	webservice.Get("/command-palette/search", commandPaletteSearch, route.WithNotAuth()),
}

const commandPaletteSessionLimit = 50

// commandPaletteSearch returns JSON jump targets matching query q.
func commandPaletteSearch(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	var pipelines []*gen.PipelineDefinition
	if s := getPipelineDefStore(); s != nil {
		defs, err := s.ListDefinitions(ctx.Context())
		if err != nil {
			return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
		}
		pipelines = defs
	}

	var sessions []chatagent.SessionSummary
	if pkgconfig.ChatAgentEnabled() {
		uid, err := webUID(ctx)
		if err == nil {
			rows, _, listErr := chatagent.ListUserActiveSessions(ctx.Context(), uid, commandPaletteSessionLimit, "")
			if listErr != nil {
				return types.Errorf(types.ErrInternal, "list sessions: %v", listErr)
			}
			sessions = rows
		}
	}

	apps := homelab.DefaultRegistry.List()
	results := buildCommandPaletteResults(ctx.Query("q"), commandPaletteNavPages(), pipelines, sessions, apps)
	return ctx.Status(http.StatusOK).JSON(results)
}

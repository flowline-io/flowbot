package workflow

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	workflowpkg "github.com/flowline-io/flowbot/pkg/workflow"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Post("/run", runWorkflow),
}

type runWorkflowRequest struct {
	File   string         `json:"file" validate:"required"`
	Params map[string]any `json:"params"`
}

func runWorkflow(ctx fiber.Ctx) error {
	var body runWorkflowRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if body.File == "" {
		return protocol.ErrBadParam.New("file path is required")
	}

	wf, err := workflowpkg.LoadFile(body.File)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	runner := workflowpkg.NewRunner()
	if err := runner.Execute(context.Background(), *wf, body.Params); err != nil {
		return fmt.Errorf("workflow execution: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]string{
		"message": fmt.Sprintf("workflow %s completed successfully", wf.Name),
	}))
}

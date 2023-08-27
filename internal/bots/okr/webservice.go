package okr

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/route"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "GET",
		Path:          "/objectives",
		Function:      objectiveList,
		Documentation: "objective list",
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:        "GET",
		Path:          "/objective/{sequence}",
		Function:      objectiveDetail,
		Documentation: "objective detail",
		Option: []route.Option{
			route.WithAuth(),
			route.WithPathParam("sequence", "id", "integer"),
		},
	},
	{
		Method:        "POST",
		Path:          "/objective",
		Function:      objectiveCreate,
		Documentation: "objective create",
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:        "PUT",
		Path:          "/objective/{sequence}",
		Function:      objectiveUpdate,
		Documentation: "objective update",
		Option: []route.Option{
			route.WithAuth(),
			route.WithPathParam("sequence", "id", "integer"),
		},
	},
	{
		Method:        "DELETE",
		Path:          "/objective/{sequence}",
		Function:      objectiveDelete,
		Documentation: "objective delete",
		Option: []route.Option{
			route.WithAuth(),
			route.WithPathParam("sequence", "id", "integer"),
		},
	},
}

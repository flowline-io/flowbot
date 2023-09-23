package okr

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "GET",
		Path:          "/objectives",
		Function:      objectiveList,
		Documentation: "objective list",
	},
	{
		Method:        "GET",
		Path:          "/objective/:sequence",
		Function:      objectiveDetail,
		Documentation: "objective detail",
	},
	{
		Method:        "POST",
		Path:          "/objective",
		Function:      objectiveCreate,
		Documentation: "objective create",
	},
	{
		Method:        "PUT",
		Path:          "/objective/:sequence",
		Function:      objectiveUpdate,
		Documentation: "objective update",
	},
	{
		Method:        "DELETE",
		Path:          "/objective/:sequence",
		Function:      objectiveDelete,
		Documentation: "objective delete",
	},
}

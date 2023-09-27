package okr

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/route"
)

var webserviceRules = []webservice.Rule{
	{
		Method:   "GET",
		Path:     "/objectives",
		Function: objectiveList,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "GET",
		Path:     "/objective/:sequence",
		Function: objectiveDetail,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "POST",
		Path:     "/objective",
		Function: objectiveCreate,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "PUT",
		Path:     "/objective/:sequence",
		Function: objectiveUpdate,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "DELETE",
		Path:     "/objective/:sequence",
		Function: objectiveDelete,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "POST",
		Path:     "/key_result",
		Function: keyResultCreate,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "PUT",
		Path:     "/key_result/:sequence",
		Function: keyResultUpdate,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "DELETE",
		Path:     "/key_result/:sequence",
		Function: keyResultDelete,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "GET",
		Path:     "/key_result/:id/values",
		Function: keyResultValueList,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "POST",
		Path:     "/key_result/:id/value",
		Function: keyResultValueCreate,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "DELETE",
		Path:     "/key_result_value/:id",
		Function: keyResultValueDelete,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
	{
		Method:   "GET",
		Path:     "/key_result_value/:id",
		Function: keyResultValue,
		Option: []route.Option{
			route.WithAuth(),
		},
	},
}

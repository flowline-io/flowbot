package okr

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/route"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/objectives", objectiveList, route.WithAuth()),
	webservice.Get("/objective/:sequence", objectiveDetail, route.WithAuth()),
	webservice.Post("/objective", objectiveCreate, route.WithAuth()),
	webservice.Put("/objective/:sequence", objectiveUpdate, route.WithAuth()),
	webservice.Delete("/objective/:sequence", objectiveDelete, route.WithAuth()),
	webservice.Post("/key_result", keyResultCreate, route.WithAuth()),
	webservice.Put("/key_result/:sequence", keyResultUpdate, route.WithAuth()),
	webservice.Delete("/key_result/:sequence", keyResultDelete, route.WithAuth()),
	webservice.Get("/key_result/:id/values", keyResultValueList, route.WithAuth()),
	webservice.Post("/key_result/:id/value", keyResultValueCreate, route.WithAuth()),
	webservice.Delete("/key_result_value/:id", keyResultValueDelete, route.WithAuth()),
	webservice.Get("/key_result_value/:id", keyResultValue, route.WithAuth()),
}

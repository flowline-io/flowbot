package okr

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/objectives", objectiveList),
	webservice.Get("/objective/:sequence", objectiveDetail),
	webservice.Post("/objective", objectiveCreate),
	webservice.Put("/objective/:sequence", objectiveUpdate),
	webservice.Delete("/objective/:sequence", objectiveDelete),
	webservice.Post("/key_result", keyResultCreate),
	webservice.Put("/key_result/:sequence", keyResultUpdate),
	webservice.Delete("/key_result/:sequence", keyResultDelete),
	webservice.Get("/key_result/:id/values", keyResultValueList),
	webservice.Post("/key_result/:id/value", keyResultValueCreate),
	webservice.Delete("/key_result_value/:id", keyResultValueDelete),
	webservice.Get("/key_result_value/:id", keyResultValue),
}

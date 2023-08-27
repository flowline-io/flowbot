package meta

import (
	"github.com/sysatom/flowbot/internal/store/model"
	"time"
)

type Step struct {
	Name              string
	UID               string
	WorkerUID         string
	ResourceVersion   string
	Generation        int
	Finalizers        interface{}
	DeletionTimestamp *time.Time

	DagUID       string
	NodeId       string
	DependNodeId []string
	State        model.StepState
}

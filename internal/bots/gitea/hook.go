package gitea

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func hookIssueOpened(ctx types.Context) {

}

func hookIssueCreated(ctx types.Context) {
	utils.PrettyPrint(ctx)
}

package gitea

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func hookIssueOpened(ctx types.Context, issue *gitea.IssuePayload) {

}

func hookIssueCreated(ctx types.Context, issue *gitea.IssuePayload) {
	utils.PrettyPrint(ctx)
}

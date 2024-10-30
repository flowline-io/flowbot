package bookmark

import (
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"

	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
)

var cronRules = []cron.Rule{
	{
		Name: "bookmarks_tag",
		When: "*/10 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			resp, err := client.GetAllBookmarks(hoarder.MaxPageSize)
			if err != nil {
				flog.Error(err)
			}

			for _, bookmark := range resp.Bookmarks {
				if len(bookmark.Tags) > 0 {
					continue
				}
				tags, err := extractTags(ctx.Context(), bookmark.Title)
				if err != nil {
					flog.Error(err)
				}
				if len(tags) == 0 {
					continue
				}

				resp, err := client.AttachTagsToBookmark(bookmark.Id, tags)
				if err != nil {
					flog.Error(err)
				}
				flog.Info("[bookmark] bookmark %s attach tags %v,esult %v", bookmark.Id, tags, resp.Attached)
			}

			return nil
		},
	},
}

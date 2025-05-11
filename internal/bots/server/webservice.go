package server

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

var webserviceRules = []webservice.Rule{
	webservice.Post("/upload", upload),
}

// upload PicGO upload api
//
//	@Summary	upload PicGO upload api
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Security	ApiKeyAuth
//	@Router		/server/upload [post]
func upload(ctx *fiber.Ctx) error {
	result := make([]string, 0)
	if form, err := ctx.MultipartForm(); err == nil {
		for _, file := range form.File {
			for _, part := range file {
				mimeType := part.Header.Get("Content-Type")
				if !utils.ValidImageContentType(mimeType) {
					continue
				}

				f, err := part.Open()
				if err != nil {
					flog.Error(fmt.Errorf("error opening file: %s, %w", part.Filename, err))
					continue
				}

				url, _, err := store.FileSystem.Upload(&types.FileDef{
					ObjHeader: types.ObjHeader{
						Id: types.Id(),
					},
					MimeType: mimeType,
					Size:     part.Size,
					Location: "/image",
				}, f)
				if err != nil {
					flog.Error(fmt.Errorf("error uploading file: %s, %w", part.Filename, err))
					continue
				}

				result = append(result, url)
			}
		}
	}

	return ctx.JSON(types.KV{
		"success": len(result) > 0,
		"result":  result,
	})
}

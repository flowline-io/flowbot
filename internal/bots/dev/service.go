package dev

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
)

// example show example data
//
//	@Summary	Show example
//	@Tags		dev
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=types.KV}
//	@Router		/dev/example [get]
func example(ctx *fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(types.KV{
		"title": "example",
	}))
}

// upload PicGO upload api
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
					flog.Error(errors.Wrapf(err, "error opening file: %s", part.Filename))
					continue
				}

				url, _, err := store.FS.Upload(&types.FileDef{
					ObjHeader: types.ObjHeader{
						Id: types.Id(),
					},
					MimeType: mimeType,
					Size:     part.Size,
					Location: "/image",
				}, f)
				if err != nil {
					flog.Error(errors.Wrapf(err, "error uploading file: %s", part.Filename))
					continue
				}

				result = append(result, url)
			}
		}
	}

	return ctx.JSON(map[string]interface{}{
		"success": len(result) > 0,
		"result":  result,
	})
}

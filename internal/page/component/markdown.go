package component

import (
	"bytes"
	"fmt"
	"github.com/flowline-io/flowbot/internal/page/library"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type Markdown struct {
	app.Compo
	Page   model.Page
	Schema types.MarkdownMsg
}

func (c *Markdown) Render() app.UI {
	var buf bytes.Buffer
	source := utils.StringToBytes(c.Schema.Raw)

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)
	err := md.Convert(source, &buf)
	if err != nil {
		buf.WriteString("error markdown")
	}

	return app.Raw(fmt.Sprintf(`
<link rel="stylesheet" href="%s" />
<div class="markdown-body" style="margin-top: 20px;">%s</div>`, library.GithubMarkdownCss, buf.String()))
}

package page

import (
	"fmt"
	"html"
	"strings"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/page/component"
	"github.com/flowline-io/flowbot/pkg/page/library"
	jsoniter "github.com/json-iterator/go"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const Layout = `
<!DOCTYPE html>
<html>
    <head>
        <title>%s</title>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
     	<link rel="stylesheet" href="%s" />
		<script src="%s"></script>
		<script src="%s"></script>
		%s
    </head>
    <body>
        %s
		%s
    </body>
</html>
`

func RenderForm(page model.Page, form model.Form) app.UI {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	d, err := json.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.FormMsg
	err = json.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Form{
		Page:   page,
		Form:   form,
		Schema: msg,
	}
	return comp
}

func RenderTable(page model.Page) app.UI {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	d, err := json.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.TableMsg
	err = json.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Table{
		Page:   page,
		Schema: msg,
	}
	return comp
}

func RenderShare(page model.Page) app.UI {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	d, err := json.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.TextMsg
	err = json.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Share{
		Page:   page,
		Schema: msg,
	}
	return comp
}

func RenderJson(page model.Page) app.UI {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	d, err := json.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.TextMsg
	err = json.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Json{
		Page:   page,
		Schema: msg,
	}
	return comp
}

func RenderHtml(page model.Page) app.UI {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	d, err := json.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.HtmlMsg
	err = json.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Html{
		Page:   page,
		Schema: msg,
	}
	return comp
}

func RenderMarkdown(page model.Page) app.UI {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	d, err := json.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.MarkdownMsg
	err = json.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Markdown{
		Page:   page,
		Schema: msg,
	}
	return comp
}

func Render(comp *types.UI) string {
	stylesStr := strings.Builder{}
	for _, style := range comp.CSS {
		stylesStr.WriteString(app.HTMLString(style))
	}
	scriptsStr := strings.Builder{}
	if len(comp.Global) > 0 {
		scriptsStr.WriteString("<script>")
		scriptsStr.WriteString("let Global = {};")
		for key, value := range comp.Global {
			switch v := value.(type) {
			case string:
				scriptsStr.WriteString(fmt.Sprintf(`Global.%s = "%s";`, key, v))
			case int, uint, int32, uint32, int64, uint64:
				scriptsStr.WriteString(fmt.Sprintf(`Global.%s = %d;`, key, v))
			case float32, float64:
				scriptsStr.WriteString(fmt.Sprintf(`Global.%s = %f;`, key, v))
			case map[string]interface{}:
				var json = jsoniter.ConfigCompatibleWithStandardLibrary
				j, err := json.Marshal(v)
				if err != nil {
					flog.Error(err)
					continue
				}
				scriptsStr.WriteString(fmt.Sprintf(`Global.%s = %s;`, key, string(j)))
			}
		}
		scriptsStr.WriteString("</script>")
	}
	for _, script := range comp.JS {
		scriptsStr.WriteString(html.UnescapeString(app.HTMLString(script)))
	}
	return fmt.Sprintf(Layout,
		comp.Title,
		library.UIKitCss, library.UIKitJs, library.UIKitIconJs,
		stylesStr.String(), app.HTMLString(comp.App), scriptsStr.String())
}

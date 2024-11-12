package page

import (
	_ "embed"
	"fmt"
	"html"
	"strings"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/page/component"
	"github.com/flowline-io/flowbot/pkg/page/library"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/flowline-io/flowbot/pkg/types"
	jsoniter "github.com/json-iterator/go"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

//go:embed styles.css
var stylesCss string

func RenderForm(page model.Page, form model.Form) app.UI {
	d, err := jsoniter.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.FormMsg
	err = jsoniter.Unmarshal(d, &msg)
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
	d, err := jsoniter.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.TableMsg
	err = jsoniter.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Table{
		Page:   page,
		Schema: msg,
	}
	return comp
}

func RenderJson(page model.Page) app.UI {
	d, err := jsoniter.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.TextMsg
	err = jsoniter.Unmarshal(d, &msg)
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
	d, err := jsoniter.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.HtmlMsg
	err = jsoniter.Unmarshal(d, &msg)
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
	d, err := jsoniter.Marshal(page.Schema)
	if err != nil {
		return nil
	}
	var msg types.MarkdownMsg
	err = jsoniter.Unmarshal(d, &msg)
	if err != nil {
		return nil
	}

	comp := &component.Markdown{
		Page:   page,
		Schema: msg,
	}
	return comp
}

func scripts(comp *types.UI) string {
	scriptsStr := strings.Builder{}
	if len(comp.Global) > 0 {
		_, _ = scriptsStr.WriteString("<script>")
		_, _ = scriptsStr.WriteString("let Global = {};")
		for key, value := range comp.Global {
			switch v := value.(type) {
			case string:
				_, _ = scriptsStr.WriteString(fmt.Sprintf(`Global.%s = "%s";`, key, v))
			case int, uint, int32, uint32, int64, uint64:
				_, _ = scriptsStr.WriteString(fmt.Sprintf(`Global.%s = %d;`, key, v))
			case float32, float64:
				_, _ = scriptsStr.WriteString(fmt.Sprintf(`Global.%s = %f;`, key, v))
			case map[string]interface{}:
				j, err := jsoniter.Marshal(v)
				if err != nil {
					flog.Error(err)
					continue
				}
				_, _ = scriptsStr.WriteString(fmt.Sprintf(`Global.%s = %s;`, key, string(j)))
			}
		}
		_, _ = scriptsStr.WriteString("</script>")
	}
	for _, script := range comp.JS {
		_, _ = scriptsStr.WriteString(html.UnescapeString(app.HTMLString(script)))
	}

	return scriptsStr.String()
}

func Render(comp *types.UI) string {
	const layout = `
<!DOCTYPE html>
<html>
    %s
    <body>
        %s
		%s
    </body>
</html>
`

	headUIs := []app.UI{
		app.Title().Text(comp.Title),
		app.Meta().Charset("utf-8"),
		app.Meta().Name("viewport").Content("width=device-width, initial-scale=1"),
		app.Style().Text(stylesCss),
		app.Link().Rel("stylesheet").Href(library.UIKitCss),
		app.Script().Src(library.UIKitJs),
		app.Script().Src(library.UIKitIconJs),
	}
	if len(comp.CSS) > 0 {
		headUIs = append(headUIs, comp.CSS...)
	}
	head := app.Head().Body(headUIs...)

	b := comp.App
	if !comp.ExpiredAt.IsZero() {
		b = app.Div().Body(
			comp.App,
			uikit.Countdown(comp.ExpiredAt),
		)
	}

	return fmt.Sprintf(layout, app.HTMLString(head), app.HTMLString(b), scripts(comp))
}

func RenderComponent(title string, a app.UI) string {
	return Render(&types.UI{
		Title: title,
		App:   uikit.Container(a),
	})
}

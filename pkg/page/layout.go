package page

import (
	_ "embed"
	"fmt"
	"html"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/page/component"
	"github.com/flowline-io/flowbot/pkg/page/library"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

//go:embed styles.css
var stylesCss string

// Define HTML layout template constant
const htmlLayout = `
<!DOCTYPE html>
<html>
    %s
    <body>
        %s
        %s
    </body>
</html>
`

// unmarshalPageSchema parses specific types of messages from Page.Schema
func unmarshalPageSchema(page model.Page, target any) error {
	data, err := sonic.Marshal(page.Schema)
	if err != nil {
		flog.Error(fmt.Errorf("failed to marshal page schema: %w", err))
		return err
	}

	err = sonic.Unmarshal(data, target)
	if err != nil {
		flog.Error(fmt.Errorf("failed to unmarshal page schema: %w", err))
		return err
	}

	return nil
}

// RenderForm renders the form page
func RenderForm(page model.Page, form model.Form) app.UI {
	var msg types.FormMsg
	if err := unmarshalPageSchema(page, &msg); err != nil {
		return nil
	}

	comp := &component.Form{
		Page:   page,
		Form:   form,
		Schema: msg,
	}
	return comp
}

// RenderTable renders the table page
func RenderTable(page model.Page) app.UI {
	var msg types.TableMsg
	if err := unmarshalPageSchema(page, &msg); err != nil {
		return nil
	}

	comp := &component.Table{
		Page:   page,
		Schema: msg,
	}
	return comp
}

// RenderHtml renders the HTML page
func RenderHtml(page model.Page) app.UI {
	var msg types.HtmlMsg
	if err := unmarshalPageSchema(page, &msg); err != nil {
		return nil
	}

	comp := &component.Html{
		Page:   page,
		Schema: msg,
	}
	return comp
}

// scripts generates the JavaScript scripts required for the page
func scripts(comp *types.UI) string {
	var scriptsStr strings.Builder

	// Handle global variables
	if len(comp.Global) > 0 {
		_, _ = scriptsStr.WriteString("<script>\nlet Global = {};\n")

		for key, value := range comp.Global {
			switch v := value.(type) {
			case string:
				_, _ = fmt.Fprintf(&scriptsStr, "Global.%s = \"%s\";\n", key, v)
			case int, uint, int32, uint32, int64, uint64:
				_, _ = fmt.Fprintf(&scriptsStr, "Global.%s = %d;\n", key, v)
			case float32, float64:
				_, _ = fmt.Fprintf(&scriptsStr, "Global.%s = %f;\n", key, v)
			case map[string]any:
				j, err := sonic.Marshal(v)
				if err != nil {
					flog.Error(fmt.Errorf("failed to marshal global variable %s: %w", key, err))
					continue
				}
				_, _ = fmt.Fprintf(&scriptsStr, "Global.%s = %s;\n", key, string(j))
			}
		}

		_, _ = scriptsStr.WriteString("</script>\n")
	}

	// Add custom scripts
	for _, script := range comp.JS {
		_, _ = scriptsStr.WriteString(html.UnescapeString(app.HTMLString(script)))
	}

	return scriptsStr.String()
}

// Render renders the complete HTML page
func Render(comp *types.UI) string {
	// Build head elements
	headUIs := []app.UI{
		app.Title().Text(comp.Title),
		app.Meta().Charset("utf-8"),
		app.Meta().Name("viewport").Content("width=device-width, initial-scale=1"),
		app.Style().Text(stylesCss),
		app.Link().Rel("stylesheet").Href(library.UIKitCss),
		app.Script().Src(library.UIKitJs),
		app.Script().Src(library.UIKitIconJs),
	}

	// Add custom CSS
	if len(comp.CSS) > 0 {
		headUIs = append(headUIs, comp.CSS...)
	}

	head := app.Head().Body(headUIs...)

	// Handle body content
	body := comp.App
	if !comp.ExpiredAt.IsZero() {
		body = app.Div().Body(
			comp.App,
			uikit.Margin(
				uikit.Text(fmt.Sprintf("Expired at: %s", comp.ExpiredAt.Format("2006-01-02 15:04:05"))),
			).Class(uikit.FlexClass, uikit.FlexCenterClass),
		)
	}

	// Assemble final HTML
	return fmt.Sprintf(
		htmlLayout,
		app.HTMLString(head),
		app.HTMLString(body),
		scripts(comp),
	)
}

// RenderComponent renders a simple component page
func RenderComponent(title string, a app.UI) string {
	return Render(&types.UI{
		Title: title,
		App:   uikit.Container(a),
	})
}

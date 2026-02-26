package component

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type BotRuleOption struct {
	Bot   string
	Rule  string
	Label string
}

func optionWithValue(value, label string, selected bool) app.HTMLOption {
	o := app.Option().Value(value).Text(label)
	if selected {
		o.Selected(true)
	}
	return o
}

func FlowListView(flag string, flows []*model.Flow) app.UI {
	return FlowListViewWithSummary(flag, flows, nil)
}

type FlowListSummary struct {
	Trigger string
	Action  string
}

func FlowListViewWithSummary(flag string, flows []*model.Flow, summaryByID map[int64]FlowListSummary) app.UI {
	var items []app.UI
	for _, flow := range flows {
		statusClass := "uk-label-default"
		if flow.Enabled {
			statusClass = "uk-label-success"
		}

		triggerText := "-"
		actionText := "-"
		if summaryByID != nil {
			if s, ok := summaryByID[flow.ID]; ok {
				if strings.TrimSpace(s.Trigger) != "" {
					triggerText = s.Trigger
				}
				if strings.TrimSpace(s.Action) != "" {
					actionText = s.Action
				}
			}
		}
		triggerUI := app.Span().Text(triggerText)
		if triggerText != "-" {
			triggerUI = uikit.Label(triggerText).Class("uk-label")
		}
		actionUI := app.Span().Text(actionText)
		if actionText != "-" {
			actionUI = uikit.Label(actionText).Class("uk-label uk-label-default")
		}

		actionsUI := app.Div().Class("uk-button-group").Body(
			uikit.Button("Execute").Class("uk-button uk-button-small uk-button-primary").Attr("onclick", fmt.Sprintf("executeFlow(%d)", flow.ID)),
			uikit.Button("Edit").Class("uk-button uk-button-small uk-button-default").Attr("onclick", fmt.Sprintf("location.href='/page/flows_edit/%s?flow_id=%d'", flag, flow.ID)),
			uikit.Button("Executions").Class("uk-button uk-button-small uk-button-default").Attr("onclick", fmt.Sprintf("location.href='/page/executions/%s?flow_id=%d'", flag, flow.ID)),
			uikit.Button("Delete").Class("uk-button uk-button-small uk-button-danger").Attr("onclick", fmt.Sprintf("deleteFlow(%d)", flow.ID)),
		)

		items = append(items, uikit.Tr(
			uikit.Td(uikit.Text(fmt.Sprintf("%d", flow.ID))),
			uikit.Td(
				app.Div().Body(
					uikit.Link(flow.Name, fmt.Sprintf("/page/flows_edit/%s?flow_id=%d", flag, flow.ID)).Class("uk-link-heading"),
					app.If(strings.TrimSpace(flow.Description) != "", func() app.UI {
						return app.Div().Class("uk-text-small uk-text-muted").Text(flow.Description)
					}),
				),
			),
			uikit.Td(triggerUI),
			uikit.Td(actionUI),
			uikit.Td(uikit.Label(func() string {
				if flow.State == 1 {
					return "Active"
				}
				return "Inactive"
			}()).Class(statusClass)),
			uikit.Td(uikit.Text(flow.CreatedAt.Format("2006-01-02 15:04:05"))),
			uikit.Td(actionsUI),
		))
	}
	if len(items) == 0 {
		items = append(items, uikit.Tr(uikit.Td(uikit.Text("No flows found.").Class(uikit.TextCenterClass)).ColSpan(8)))
	}

	return uikit.App(
		app.Div().Class("uk-flex uk-flex-between uk-flex-middle uk-margin").Body(
			uikit.H2("Flows").Class("uk-margin-remove"),
			uikit.Button("New Flow").Class("uk-button uk-button-primary").Attr("onclick", fmt.Sprintf("location.href='/page/flows_edit/%s'", flag)),
		),
		app.Div().Class("uk-card uk-card-default uk-card-body uk-padding-small").Body(
			app.Div().Class("uk-overflow-auto").Body(
				uikit.Table(
					uikit.THead(uikit.Tr(
						uikit.Th(uikit.Text("ID")),
						uikit.Th(uikit.Text("Name")),
						uikit.Th(uikit.Text("Trigger")),
						uikit.Th(uikit.Text("Action")),
						uikit.Th(uikit.Text("Status")),
						uikit.Th(uikit.Text("Created")),
						uikit.Th(uikit.Text("Actions")),
					)),
					uikit.TBody(items...),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass, "uk-table-small", "uk-table-middle"),
			),
		),
	)
}

func ExecutionsView(execs []*model.Execution) app.UI {
	var items []app.UI
	for _, exec := range execs {
		stateClass := "uk-label-default"
		switch exec.State {
		case 1:
			stateClass = "uk-label-warning"
		case 2:
			stateClass = "uk-label-primary"
		case 3:
			stateClass = "uk-label-success"
		case 4:
			stateClass = "uk-label-danger"
		}

		items = append(items, uikit.Tr(
			uikit.Td(uikit.Text(exec.ExecutionID)),
			uikit.Td(uikit.Text(exec.TriggerType)),
			uikit.Td(uikit.Label(execState(exec.State)).Class(stateClass)),
			uikit.Td(uikit.Text(exec.CreatedAt.Format("2006-01-02 15:04:05"))),
			uikit.Td(uikit.Text(func() string {
				if exec.FinishedAt != nil {
					return exec.FinishedAt.Format("2006-01-02 15:04:05")
				}
				return "-"
			}())),
			uikit.Td(uikit.Text(func() string {
				if exec.Error != "" {
					return exec.Error
				}
				return "-"
			}())),
		))
	}
	if len(items) == 0 {
		items = append(items, uikit.Tr(uikit.Td(uikit.Text("No executions found.").Class(uikit.TextCenterClass)).ColSpan(6)))
	}

	return uikit.App(
		uikit.H2("Executions").Class(uikit.TextCenterClass),
		uikit.Table(
			uikit.THead(uikit.Tr(
				uikit.Th(uikit.Text("Execution ID")),
				uikit.Th(uikit.Text("Trigger")),
				uikit.Th(uikit.Text("State")),
				uikit.Th(uikit.Text("Started")),
				uikit.Th(uikit.Text("Finished")),
				uikit.Th(uikit.Text("Error")),
			)),
			uikit.TBody(items...),
		).Class(uikit.TableDividerClass, uikit.TableHoverClass),
	)
}

func AppsView(apps []*model.App) app.UI {
	var items []app.UI
	for _, appItem := range apps {
		statusClass := "uk-label-default"
		switch appItem.Status {
		case "running":
			statusClass = "uk-label-success"
		case "stopped":
			statusClass = "uk-label-danger"
		case "paused":
			statusClass = "uk-label-warning"
		}

		items = append(items, uikit.Tr(
			uikit.Td(uikit.Text(appItem.Name)),
			uikit.Td(uikit.Text(appItem.Path)),
			uikit.Td(uikit.Label(string(appItem.Status)).Class(statusClass)),
			uikit.Td(uikit.Text(appItem.ContainerID)),
			uikit.Td(
				uikit.Button("Start").Class("uk-button uk-button-small uk-button-primary").Attr("onclick", fmt.Sprintf("appAction(%d, 'start')", appItem.ID)),
				uikit.Button("Stop").Class("uk-button uk-button-small uk-button-danger").Attr("onclick", fmt.Sprintf("appAction(%d, 'stop')", appItem.ID)),
				uikit.Button("Restart").Class("uk-button uk-button-small uk-button-default").Attr("onclick", fmt.Sprintf("appAction(%d, 'restart')", appItem.ID)),
			),
		))
	}
	if len(items) == 0 {
		items = append(items, uikit.Tr(uikit.Td(uikit.Text("No apps found.").Class(uikit.TextCenterClass)).ColSpan(5)))
	}

	return uikit.App(
		uikit.H2("Apps").Class(uikit.TextCenterClass),
		uikit.Button("Scan Apps").Class("uk-button uk-button-primary").Attr("onclick", "scanApps()"),
		uikit.Table(
			uikit.THead(uikit.Tr(
				uikit.Th(uikit.Text("Name")),
				uikit.Th(uikit.Text("Path")),
				uikit.Th(uikit.Text("Status")),
				uikit.Th(uikit.Text("Container ID")),
				uikit.Th(uikit.Text("Actions")),
			)),
			uikit.TBody(items...),
		).Class(uikit.TableDividerClass, uikit.TableHoverClass),
	)
}

func ConnectionsView(flag string, conns []*model.Connection) app.UI {
	var items []app.UI
	for _, conn := range conns {
		items = append(items, uikit.Tr(
			uikit.Td(uikit.Text(conn.Name)),
			uikit.Td(uikit.Text(conn.Type)),
			uikit.Td(uikit.Label(func() string {
				if conn.Enabled {
					return "Enabled"
				}
				return "Disabled"
			}()).Class(func() string {
				if conn.Enabled {
					return "uk-label-success"
				}
				return "uk-label-default"
			}())),
			uikit.Td(uikit.Text(conn.CreatedAt.Format("2006-01-02 15:04:05"))),
			uikit.Td(
				uikit.Button("Edit").Class("uk-button uk-button-small").Attr("onclick", fmt.Sprintf("location.href='/page/connection_edit/%s?id=%d'", flag, conn.ID)),
				uikit.Button("Delete").Class("uk-button uk-button-small uk-button-danger").Attr("onclick", fmt.Sprintf("deleteConnection(%d)", conn.ID)),
			),
		))
	}
	if len(items) == 0 {
		items = append(items, uikit.Tr(uikit.Td(uikit.Text("No connections found.").Class(uikit.TextCenterClass)).ColSpan(5)))
	}

	return uikit.App(
		uikit.H2("Connections").Class(uikit.TextCenterClass),
		uikit.Button("New Connection").Class("uk-button uk-button-primary").Attr("onclick", fmt.Sprintf("location.href='/page/connection_edit/%s'", flag)),
		uikit.Table(
			uikit.THead(uikit.Tr(
				uikit.Th(uikit.Text("Name")),
				uikit.Th(uikit.Text("Type")),
				uikit.Th(uikit.Text("Status")),
				uikit.Th(uikit.Text("Created")),
				uikit.Th(uikit.Text("Actions")),
			)),
			uikit.TBody(items...),
		).Class(uikit.TableDividerClass, uikit.TableHoverClass),
	)
}

func AuthenticationsView(flag string, auths []*model.Authentication) app.UI {
	var items []app.UI
	for _, auth := range auths {
		items = append(items, uikit.Tr(
			uikit.Td(uikit.Text(auth.Name)),
			uikit.Td(uikit.Text(auth.Type)),
			uikit.Td(uikit.Label(func() string {
				if auth.Enabled {
					return "Enabled"
				}
				return "Disabled"
			}()).Class(func() string {
				if auth.Enabled {
					return "uk-label-success"
				}
				return "uk-label-default"
			}())),
			uikit.Td(uikit.Text(auth.CreatedAt.Format("2006-01-02 15:04:05"))),
			uikit.Td(
				uikit.Button("Edit").Class("uk-button uk-button-small").Attr("onclick", fmt.Sprintf("location.href='/page/authentication_edit/%s?id=%d'", flag, auth.ID)),
				uikit.Button("Delete").Class("uk-button uk-button-small uk-button-danger").Attr("onclick", fmt.Sprintf("deleteAuth(%d)", auth.ID)),
			),
		))
	}
	if len(items) == 0 {
		items = append(items, uikit.Tr(uikit.Td(uikit.Text("No authentications found.").Class(uikit.TextCenterClass)).ColSpan(5)))
	}

	return uikit.App(
		uikit.H2("Authentications").Class(uikit.TextCenterClass),
		uikit.Button("New Authentication").Class("uk-button uk-button-primary").Attr("onclick", fmt.Sprintf("location.href='/page/authentication_edit/%s'", flag)),
		uikit.Table(
			uikit.THead(uikit.Tr(
				uikit.Th(uikit.Text("Name")),
				uikit.Th(uikit.Text("Type")),
				uikit.Th(uikit.Text("Status")),
				uikit.Th(uikit.Text("Created")),
				uikit.Th(uikit.Text("Actions")),
			)),
			uikit.TBody(items...),
		).Class(uikit.TableDividerClass, uikit.TableHoverClass),
	)
}

type FlowEditData struct {
	Flag           string
	FlowID         string
	Name           string
	Description    string
	Enabled        bool
	Trigger        string
	TriggerParams  string
	WebhookURL     string
	Action         string
	ActionParams   string
	RuleMetaJSON   string
	TriggerOptions []BotRuleOption
	ActionOptions  []BotRuleOption
}

func FlowEditView(d FlowEditData) app.UI {
	if strings.TrimSpace(d.TriggerParams) == "" {
		d.TriggerParams = "{}"
	}
	if strings.TrimSpace(d.ActionParams) == "" {
		d.ActionParams = "{}"
	}

	var triggerOptions []app.UI
	for _, opt := range d.TriggerOptions {
		value := opt.Bot + "|" + opt.Rule
		triggerOptions = append(triggerOptions, optionWithValue(value, opt.Label, d.Trigger == value))
	}
	triggerSelect := uikit.Select(triggerOptions...).Name("trigger")

	var actionOptions []app.UI
	actionOptions = append(actionOptions, optionWithValue("", "(none)", d.Action == ""))
	for _, opt := range d.ActionOptions {
		value := opt.Bot + "|" + opt.Rule
		actionOptions = append(actionOptions, optionWithValue(value, opt.Label, d.Action == value))
	}
	actionSelect := uikit.Select(actionOptions...).Name("action")

	return uikit.App(
		uikit.H2("Flow Editor").Class(uikit.TextCenterClass),
		app.If(strings.TrimSpace(d.RuleMetaJSON) != "", func() app.UI {
			// Use Raw to avoid HTML-escaping the JSON inside the script tag.
			return app.Raw(fmt.Sprintf(`<script type="application/json" id="flow_rule_meta">%s</script>`, d.RuleMetaJSON))
		}),
		uikit.Form().ID("flow_edit_form").Body(
			uikit.Input().Type("hidden").Name("flow_id").Value(d.FlowID),
			uikit.Input().Type("hidden").Name("flag").Value(d.Flag),

			uikit.Margin(uikit.Text("Name")),
			uikit.Input().Name("name").Value(d.Name).Class(uikit.WidthClass(1, 1)),

			uikit.Margin(uikit.Text("Description")),
			uikit.Input().Name("description").Value(d.Description).Class(uikit.WidthClass(1, 1)),

			uikit.Margin(
				app.Label().Body(
					uikit.Checkbox().Name("enabled").Checked(d.Enabled),
					app.Span().Text(" Enabled"),
				),
			),

			uikit.H3("Trigger"),
			triggerSelect,
			uikit.Margin(uikit.Text("Trigger Params (JSON)")),
			uikit.Textarea().ID("trigger_params").Text(d.TriggerParams).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(uikit.Text("Trigger Params Example (JSON)")),
			app.Pre().ID("trigger_params_example").Text("{}").Class(uikit.WidthClass(1, 1)),
			uikit.Margin(uikit.Text("Ingredients Variables")),
			app.Div().ID("trigger_ingredient_vars").Body(uikit.Text("")),
			app.If(d.WebhookURL != "", func() app.UI {
				return app.Div().Body(
					uikit.Margin(uikit.Text("Webhook URL")),
					uikit.Input().Attr("readonly", "readonly").Value(d.WebhookURL).Class(uikit.WidthClass(1, 1)),
				)
			}),

			uikit.H3("Action"),
			actionSelect,
			uikit.Margin(uikit.Text("Action Params (JSON)")),
			uikit.Textarea().ID("action_params").Text(d.ActionParams).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(uikit.Text("Action Params Example (JSON)")),
			app.Pre().ID("action_params_example").Text("{}").Class(uikit.WidthClass(1, 1)),

			uikit.Margin(
				uikit.Button("Save").Class("uk-button uk-button-primary").Attr("type", "button").Attr("onclick", "saveFlow('flow_edit_form')"),
				uikit.Button("Back").Class("uk-button uk-button-default").Attr("type", "button").Attr("onclick", fmt.Sprintf("location.href='/page/flows_list/%s'", d.Flag)),
			),
		),
	)
}

type ConnectionEditData struct {
	Flag    string
	ID      string
	Name    string
	Type    string
	Enabled bool
	Config  string
}

func ConnectionEditView(d ConnectionEditData) app.UI {
	if strings.TrimSpace(d.Config) == "" {
		d.Config = "{}"
	}
	return uikit.App(
		uikit.H2("Connection Editor").Class(uikit.TextCenterClass),
		uikit.Form().ID("connection_edit_form").Body(
			uikit.Input().Type("hidden").Name("id").Value(d.ID),
			uikit.Input().Type("hidden").Name("flag").Value(d.Flag),
			uikit.Margin(uikit.Text("Name")),
			uikit.Input().Name("name").Value(d.Name).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(uikit.Text("Type")),
			uikit.Input().Name("type").Value(d.Type).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(
				app.Label().Body(
					uikit.Checkbox().Name("enabled").Checked(d.Enabled),
					app.Span().Text(" Enabled"),
				),
			),
			uikit.Margin(uikit.Text("Config (JSON)")),
			uikit.Textarea().ID("conn_config").Text(d.Config).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(
				uikit.Button("Save").Class("uk-button uk-button-primary").Attr("type", "button").Attr("onclick", "saveConnection('connection_edit_form')"),
				uikit.Button("Back").Class("uk-button uk-button-default").Attr("type", "button").Attr("onclick", fmt.Sprintf("location.href='/page/connections/%s'", d.Flag)),
			),
		),
	)
}

type AuthenticationEditData struct {
	Flag        string
	ID          string
	Name        string
	Type        string
	Enabled     bool
	Credentials string
}

func AuthenticationEditView(d AuthenticationEditData) app.UI {
	if strings.TrimSpace(d.Credentials) == "" {
		d.Credentials = "{}"
	}
	return uikit.App(
		uikit.H2("Authentication Editor").Class(uikit.TextCenterClass),
		uikit.Form().ID("auth_edit_form").Body(
			uikit.Input().Type("hidden").Name("id").Value(d.ID),
			uikit.Input().Type("hidden").Name("flag").Value(d.Flag),
			uikit.Margin(uikit.Text("Name")),
			uikit.Input().Name("name").Value(d.Name).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(uikit.Text("Type")),
			uikit.Input().Name("type").Value(d.Type).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(
				app.Label().Body(
					uikit.Checkbox().Name("enabled").Checked(d.Enabled),
					app.Span().Text(" Enabled"),
				),
			),
			uikit.Margin(uikit.Text("Credentials (JSON)")),
			uikit.Textarea().ID("auth_credentials").Text(d.Credentials).Class(uikit.WidthClass(1, 1)),
			uikit.Margin(
				uikit.Button("Save").Class("uk-button uk-button-primary").Attr("type", "button").Attr("onclick", "saveAuthentication('auth_edit_form')"),
				uikit.Button("Back").Class("uk-button uk-button-default").Attr("type", "button").Attr("onclick", fmt.Sprintf("location.href='/page/authentications/%s'", d.Flag)),
			),
		),
	)
}

func execState(state model.ExecutionState) string {
	switch state {
	case 0:
		return "Unknown"
	case 1:
		return "Pending"
	case 2:
		return "Running"
	case 3:
		return "Succeeded"
	case 4:
		return "Failed"
	case 5:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

func JSONString(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "{}"
	}
	return s
}

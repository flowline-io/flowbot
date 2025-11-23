package flows

import (
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	flowsListPageId       = "flows_list"
	flowsEditPageId       = "flows_edit"
	executionsPageId      = "executions"
	appsPageId            = "apps"
	connectionsPageId     = "connections"
	authenticationsPageId = "authentications"
)

var pageRules = []page.Rule{
	{
		Id: flowsListPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			// Load flows from database
			flows, err := store.Database.GetFlows(ctx.AsUser, ctx.Topic)
			if err != nil {
				return nil, fmt.Errorf("failed to get flows: %w", err)
			}

			var items []app.UI
			for _, flow := range flows {
				statusClass := "uk-label-default"
				if flow.Enabled {
					statusClass = "uk-label-success"
				}

				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text(fmt.Sprintf("%d", flow.ID))),
					uikit.Td(uikit.Link(flow.Name, fmt.Sprintf("/page/%s/%s?flow_id=%d", flowsEditPageId, flag, flow.ID))),
					uikit.Td(uikit.Text(flow.Description)),
					uikit.Td(
						uikit.Label(func() string {
							if flow.State == 1 {
								return "Active"
							}
							return "Inactive"
						}()).Class(statusClass),
					),
					uikit.Td(uikit.Text(flow.CreatedAt.Format("2006-01-02 15:04:05"))),
					uikit.Td(
						uikit.Button("Execute").Class("uk-button uk-button-small uk-button-primary").
							Attr("onclick", fmt.Sprintf("executeFlow(%d)", flow.ID)),
						uikit.Button("Delete").Class("uk-button uk-button-small uk-button-danger").
							Attr("onclick", fmt.Sprintf("deleteFlow(%d)", flow.ID)),
					),
				))
			}

			if len(items) == 0 {
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text("No flows found.").Class(uikit.TextCenterClass)).ColSpan(6),
				))
			}

			js := fmt.Sprintf(`
				function executeFlow(id) {
					fetch('/service/flows/' + id + '/execute', {
						method: 'POST',
						headers: {'Content-Type': 'application/json'},
						body: JSON.stringify({trigger_type: 'manual', trigger_id: '', payload: {}})
					}).then(r => r.json()).then(d => {
						alert(d.message || 'Flow executed');
						location.reload();
					});
				}
				function deleteFlow(id) {
					if (confirm('Are you sure?')) {
						fetch('/service/flows/' + id, {method: 'DELETE'})
							.then(r => location.reload());
					}
				}
			`)

			appUI := uikit.App(
				uikit.H2("Flows").Class(uikit.TextCenterClass),
				uikit.Button("New Flow").Class("uk-button uk-button-primary").
					Attr("onclick", fmt.Sprintf("location.href='/page/%s/%s'", flowsEditPageId, flag)),
				uikit.Table(
					uikit.THead(
						uikit.Tr(
							uikit.Th(uikit.Text("ID")),
							uikit.Th(uikit.Text("Name")),
							uikit.Th(uikit.Text("Description")),
							uikit.Th(uikit.Text("Status")),
							uikit.Th(uikit.Text("Created")),
							uikit.Th(uikit.Text("Actions")),
						),
					),
					uikit.TBody(items...),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass),
			)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(js)},
			}, nil
		},
	},
	{
		Id: executionsPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			flowIDStr, _ := args.String("flow_id")
			if flowIDStr == "" {
				return nil, fmt.Errorf("flow_id is required")
			}

			var flowID int64
			if _, err := fmt.Sscanf(flowIDStr, "%d", &flowID); err != nil {
				return nil, fmt.Errorf("invalid flow_id: %s", flowIDStr)
			}

			executions, err := store.Database.GetExecutions(flowID, 50)
			if err != nil {
				return nil, fmt.Errorf("failed to get executions: %w", err)
			}

			var items []app.UI
			for _, exec := range executions {
				stateClass := "uk-label-default"
				switch exec.State {
				case 1: // Pending
					stateClass = "uk-label-warning"
				case 2: // Running
					stateClass = "uk-label-primary"
				case 3: // Succeeded
					stateClass = "uk-label-success"
				case 4: // Failed
					stateClass = "uk-label-danger"
				}

				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text(exec.ExecutionID)),
					uikit.Td(uikit.Text(exec.TriggerType)),
					uikit.Td(
						uikit.Label(func() string {
							switch exec.State {
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
							}
							return "Unknown"
						}()).Class(stateClass),
					),
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
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text("No executions found.").Class(uikit.TextCenterClass)).ColSpan(6),
				))
			}

			appUI := uikit.App(
				uikit.H2("Executions").Class(uikit.TextCenterClass),
				uikit.Table(
					uikit.THead(
						uikit.Tr(
							uikit.Th(uikit.Text("Execution ID")),
							uikit.Th(uikit.Text("Trigger")),
							uikit.Th(uikit.Text("State")),
							uikit.Th(uikit.Text("Started")),
							uikit.Th(uikit.Text("Finished")),
							uikit.Th(uikit.Text("Error")),
						),
					),
					uikit.TBody(items...),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass),
			)

			return &types.UI{
				App: appUI,
			}, nil
		},
	},
	{
		Id: appsPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			apps, err := store.Database.GetApps()
			if err != nil {
				return nil, fmt.Errorf("failed to get apps: %w", err)
			}

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
					uikit.Td(
						uikit.Label(string(appItem.Status)).Class(statusClass),
					),
					uikit.Td(uikit.Text(appItem.ContainerID)),
					uikit.Td(
						uikit.Button("Start").Class("uk-button uk-button-small uk-button-primary").
							Attr("onclick", fmt.Sprintf("appAction(%d, 'start')", appItem.ID)),
						uikit.Button("Stop").Class("uk-button uk-button-small uk-button-danger").
							Attr("onclick", fmt.Sprintf("appAction(%d, 'stop')", appItem.ID)),
						uikit.Button("Restart").Class("uk-button uk-button-small uk-button-default").
							Attr("onclick", fmt.Sprintf("appAction(%d, 'restart')", appItem.ID)),
					),
				))
			}

			if len(items) == 0 {
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text("No apps found.").Class(uikit.TextCenterClass)).ColSpan(5),
				))
			}

			js := fmt.Sprintf(`
				function appAction(id, action) {
					fetch('/service/apps/' + id + '/' + action, {method: 'POST'})
						.then(r => r.json())
						.then(d => {
							alert(d.message || 'Action completed');
							location.reload();
						});
				}
				function scanApps() {
					fetch('/service/apps/scan', {method: 'POST'})
						.then(r => r.json())
						.then(d => {
							alert(d.message || 'Scan completed');
							location.reload();
						});
				}
			`)

			appUI := uikit.App(
				uikit.H2("Apps").Class(uikit.TextCenterClass),
				uikit.Button("Scan Apps").Class("uk-button uk-button-primary").
					Attr("onclick", "scanApps()"),
				uikit.Table(
					uikit.THead(
						uikit.Tr(
							uikit.Th(uikit.Text("Name")),
							uikit.Th(uikit.Text("Path")),
							uikit.Th(uikit.Text("Status")),
							uikit.Th(uikit.Text("Container ID")),
							uikit.Th(uikit.Text("Actions")),
						),
					),
					uikit.TBody(items...),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass),
			)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(js)},
			}, nil
		},
	},
	{
		Id: connectionsPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			connections, err := store.Database.GetConnections(ctx.AsUser, ctx.Topic)
			if err != nil {
				return nil, fmt.Errorf("failed to get connections: %w", err)
			}

			var items []app.UI
			for _, conn := range connections {
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text(conn.Name)),
					uikit.Td(uikit.Text(conn.Type)),
					uikit.Td(
						uikit.Label(func() string {
							if conn.Enabled {
								return "Enabled"
							}
							return "Disabled"
						}()).Class(func() string {
							if conn.Enabled {
								return "uk-label-success"
							}
							return "uk-label-default"
						}()),
					),
					uikit.Td(uikit.Text(conn.CreatedAt.Format("2006-01-02 15:04:05"))),
					uikit.Td(
						uikit.Button("Edit").Class("uk-button uk-button-small"),
						uikit.Button("Delete").Class("uk-button uk-button-small uk-button-danger").
							Attr("onclick", fmt.Sprintf("deleteConnection(%d)", conn.ID)),
					),
				))
			}

			if len(items) == 0 {
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text("No connections found.").Class(uikit.TextCenterClass)).ColSpan(5),
				))
			}

			js := fmt.Sprintf(`
				function deleteConnection(id) {
					if (confirm('Are you sure?')) {
						fetch('/service/connections/' + id, {method: 'DELETE'})
							.then(r => location.reload());
					}
				}
			`)

			appUI := uikit.App(
				uikit.H2("Connections").Class(uikit.TextCenterClass),
				uikit.Button("New Connection").Class("uk-button uk-button-primary"),
				uikit.Table(
					uikit.THead(
						uikit.Tr(
							uikit.Th(uikit.Text("Name")),
							uikit.Th(uikit.Text("Type")),
							uikit.Th(uikit.Text("Status")),
							uikit.Th(uikit.Text("Created")),
							uikit.Th(uikit.Text("Actions")),
						),
					),
					uikit.TBody(items...),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass),
			)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(js)},
			}, nil
		},
	},
	{
		Id: authenticationsPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			auths, err := store.Database.GetAuthentications(ctx.AsUser, ctx.Topic)
			if err != nil {
				return nil, fmt.Errorf("failed to get authentications: %w", err)
			}

			var items []app.UI
			for _, auth := range auths {
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text(auth.Name)),
					uikit.Td(uikit.Text(auth.Type)),
					uikit.Td(
						uikit.Label(func() string {
							if auth.Enabled {
								return "Enabled"
							}
							return "Disabled"
						}()).Class(func() string {
							if auth.Enabled {
								return "uk-label-success"
							}
							return "uk-label-default"
						}()),
					),
					uikit.Td(uikit.Text(auth.CreatedAt.Format("2006-01-02 15:04:05"))),
					uikit.Td(
						uikit.Button("Edit").Class("uk-button uk-button-small"),
						uikit.Button("Delete").Class("uk-button uk-button-small uk-button-danger").
							Attr("onclick", fmt.Sprintf("deleteAuth(%d)", auth.ID)),
					),
				))
			}

			if len(items) == 0 {
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Text("No authentications found.").Class(uikit.TextCenterClass)).ColSpan(5),
				))
			}

			js := fmt.Sprintf(`
				function deleteAuth(id) {
					if (confirm('Are you sure?')) {
						fetch('/service/authentications/' + id, {method: 'DELETE'})
							.then(r => location.reload());
					}
				}
			`)

			appUI := uikit.App(
				uikit.H2("Authentications").Class(uikit.TextCenterClass),
				uikit.Button("New Authentication").Class("uk-button uk-button-primary"),
				uikit.Table(
					uikit.THead(
						uikit.Tr(
							uikit.Th(uikit.Text("Name")),
							uikit.Th(uikit.Text("Type")),
							uikit.Th(uikit.Text("Status")),
							uikit.Th(uikit.Text("Created")),
							uikit.Th(uikit.Text("Actions")),
						),
					),
					uikit.TBody(items...),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass),
			)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(js)},
			}, nil
		},
	},
}

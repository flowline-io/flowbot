package flows

import (
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/page/component"
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
	connectionEditPageId  = "connection_edit"
	authenticationEditId  = "authentication_edit"
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

			appUI := component.FlowListView(flag, flows)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(component.AdminJS())},
			}, nil
		},
	},
	{
		Id: flowsEditPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			flowIDStr, _ := args.String("flow_id")
			data := component.FlowEditData{
				Flag:        flag,
				FlowID:      "",
				Enabled:     true,
				TriggerType: "manual",
			}

			if flowIDStr != "" {
				var flowID int64
				if _, err := fmt.Sscanf(flowIDStr, "%d", &flowID); err != nil {
					return nil, fmt.Errorf("invalid flow_id: %s", flowIDStr)
				}

				flow, err := store.Database.GetFlow(flowID)
				if err != nil {
					return nil, fmt.Errorf("failed to get flow: %w", err)
				}
				nodes, _ := store.Database.GetFlowNodes(flowID)
				edges, _ := store.Database.GetFlowEdges(flowID)

				data.FlowID = fmt.Sprintf("%d", flow.ID)
				data.Name = flow.Name
				data.Description = flow.Description
				data.Enabled = flow.Enabled

				triggerNode, action1Node, action2Node := flowGraphToChain(nodes, edges)
				if triggerNode != nil {
					data.TriggerType = triggerNode.RuleID
					if triggerNode.Parameters != nil {
						if tok, ok := triggerNode.Parameters["token"].(string); ok {
							data.WebhookToken = tok
						}
						if spec, ok := triggerNode.Parameters["spec"].(string); ok {
							data.CronSpec = spec
						}
					}
				}
				if action1Node != nil {
					data.Action1 = action1Node.Bot + "|" + action1Node.RuleID
					data.Action1Params = component.JSONString(action1Node.Parameters)
				} else {
					data.Action1Params = "{}"
				}
				if action2Node != nil {
					data.Action2 = action2Node.Bot + "|" + action2Node.RuleID
					data.Action2Params = component.JSONString(action2Node.Parameters)
				} else {
					data.Action2Params = "{}"
				}
			}

			appUI := component.FlowEditView(data)
			return &types.UI{App: appUI, JS: []app.HTMLScript{uikit.Js(component.AdminJS())}}, nil
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

			appUI := component.ExecutionsView(executions)
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

			appUI := component.AppsView(apps)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(component.AdminJS())},
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

			appUI := component.ConnectionsView(flag, connections)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(component.AdminJS())},
			}, nil
		},
	},
	{
		Id: connectionEditPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			idStr, _ := args.String("id")
			data := component.ConnectionEditData{Flag: flag, Enabled: true}
			if idStr != "" {
				var id int64
				if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
					return nil, fmt.Errorf("invalid id: %s", idStr)
				}
				conn, err := store.Database.GetConnection(id)
				if err != nil {
					return nil, fmt.Errorf("failed to get connection: %w", err)
				}
				data.ID = fmt.Sprintf("%d", conn.ID)
				data.Name = conn.Name
				data.Type = conn.Type
				data.Enabled = conn.Enabled
				data.Config = component.JSONString(conn.Config)
			}
			appUI := component.ConnectionEditView(data)
			return &types.UI{App: appUI, JS: []app.HTMLScript{uikit.Js(component.AdminJS())}}, nil
		},
	},
	{
		Id: authenticationsPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			auths, err := store.Database.GetAuthentications(ctx.AsUser, ctx.Topic)
			if err != nil {
				return nil, fmt.Errorf("failed to get authentications: %w", err)
			}

			appUI := component.AuthenticationsView(flag, auths)

			return &types.UI{
				App: appUI,
				JS:  []app.HTMLScript{uikit.Js(component.AdminJS())},
			}, nil
		},
	},
	{
		Id: authenticationEditId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			idStr, _ := args.String("id")
			data := component.AuthenticationEditData{Flag: flag, Enabled: true}
			if idStr != "" {
				var id int64
				if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
					return nil, fmt.Errorf("invalid id: %s", idStr)
				}
				auth, err := store.Database.GetAuthentication(id)
				if err != nil {
					return nil, fmt.Errorf("failed to get authentication: %w", err)
				}
				data.ID = fmt.Sprintf("%d", auth.ID)
				data.Name = auth.Name
				data.Type = auth.Type
				data.Enabled = auth.Enabled
				data.Credentials = component.JSONString(auth.Credentials)
			}
			appUI := component.AuthenticationEditView(data)
			return &types.UI{App: appUI, JS: []app.HTMLScript{uikit.Js(component.AdminJS())}}, nil
		},
	},
}

func flowGraphToChain(nodes []*model.FlowNode, edges []*model.FlowEdge) (trigger *model.FlowNode, action1 *model.FlowNode, action2 *model.FlowNode) {
	if len(nodes) == 0 {
		return nil, nil, nil
	}
	nodeMap := make(map[string]*model.FlowNode, len(nodes))
	for _, n := range nodes {
		nodeMap[n.NodeID] = n
		if trigger == nil && n.Type == model.NodeTypeTrigger {
			trigger = n
		}
	}
	if trigger == nil {
		return nil, nil, nil
	}

	outgoing := make(map[string][]string)
	for _, e := range edges {
		outgoing[e.SourceNode] = append(outgoing[e.SourceNode], e.TargetNode)
	}

	firstTargets := outgoing[trigger.NodeID]
	for _, tid := range firstTargets {
		n := nodeMap[tid]
		if n != nil && n.Type == model.NodeTypeAction {
			action1 = n
			break
		}
	}
	if action1 == nil {
		return trigger, nil, nil
	}

	secondTargets := outgoing[action1.NodeID]
	for _, tid := range secondTargets {
		n := nodeMap[tid]
		if n != nil && n.Type == model.NodeTypeAction {
			action2 = n
			break
		}
	}
	return trigger, action1, action2
}

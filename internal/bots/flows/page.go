package flows

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/flows"
	"github.com/flowline-io/flowbot/pkg/page/component"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/action"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/trigger"
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

			summaryByID := make(map[int64]component.FlowListSummary)
			for _, f := range flows {
				if f == nil {
					continue
				}
				nodes, err1 := store.Database.GetFlowNodes(f.ID)
				edges, err2 := store.Database.GetFlowEdges(f.ID)
				if err1 != nil || err2 != nil {
					continue
				}
				tr, a1, a2 := flowGraphToChain(nodes, edges)

				triggerText := ""
				if tr != nil {
					bot := tr.Bot
					if bot == "system" {
						bot = "dev"
					}
					triggerText = bot + "|" + tr.RuleID
				}

				actionText := ""
				if a1 != nil {
					bot := a1.Bot
					if bot == "system" {
						bot = "dev"
					}
					actionText = bot + "|" + a1.RuleID
					if a2 != nil {
						bot2 := a2.Bot
						if bot2 == "system" {
							bot2 = "dev"
						}
						actionText = actionText + " -> " + bot2 + "|" + a2.RuleID
					}
				}

				summaryByID[f.ID] = component.FlowListSummary{Trigger: triggerText, Action: actionText}
			}

			appUI := component.FlowListViewWithSummary(flag, flows, summaryByID)

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
				Flag:          flag,
				FlowID:        "",
				Enabled:       true,
				Trigger:       "dev|manual",
				TriggerParams: "{}",
				ActionParams:  "{}",
			}

			// Build metadata for Flow Editor: params examples and ingredient-path roots.
			type ruleMeta struct {
				ParamsExample       types.KV            `json:"params_example"`
				IngredientPathRoots []string            `json:"ingredient_path_roots,omitempty"`
				Inputs              []flows.ParamSpec   `json:"inputs,omitempty"`
				Ingredients         []flows.Ingredient  `json:"ingredients,omitempty"`
				TriggerMode         trigger.Mode        `json:"trigger_mode,omitempty"`
				TriggerAbstraction  trigger.Abstraction `json:"trigger_abstraction,omitempty"`
				Title               string              `json:"title,omitempty"`
			}
			meta := struct {
				Triggers map[string]ruleMeta `json:"triggers"`
				Actions  map[string]ruleMeta `json:"actions"`
			}{
				Triggers: make(map[string]ruleMeta),
				Actions:  make(map[string]ruleMeta),
			}

			// Collect trigger/action options from registered bots.
			for botName, h := range chatbot.List() {
				if h == nil || !h.IsReady() {
					continue
				}
				for _, rs := range h.Rules() {
					switch v := rs.(type) {
					case []trigger.Rule:
						for _, r := range v {
							label := fmt.Sprintf("%s: %s (%s)", botName, r.Title, r.Id)
							data.TriggerOptions = append(data.TriggerOptions, component.BotRuleOption{Bot: botName, Rule: r.Id, Label: label})

							key := botName + "|" + r.Id
							roots := []string{"payload"}
							if r.Mode == trigger.ModePoll {
								roots = append(roots, "item")
							}
							meta.Triggers[key] = ruleMeta{
								ParamsExample:       defaultTriggerParamsExample(r),
								IngredientPathRoots: roots,
								Ingredients:         r.Ingredients,
								TriggerMode:         r.Mode,
								TriggerAbstraction:  r.Abstraction,
								Title:               r.Title,
							}
						}
					case []action.Rule:
						for _, r := range v {
							label := fmt.Sprintf("%s: %s (%s)", botName, r.Title, r.Id)
							data.ActionOptions = append(data.ActionOptions, component.BotRuleOption{Bot: botName, Rule: r.Id, Label: label})

							key := botName + "|" + r.Id
							meta.Actions[key] = ruleMeta{
								ParamsExample: defaultParamsExampleFromSpecs(r.Inputs),
								Inputs:        r.Inputs,
								Title:         r.Title,
							}
						}
					}
				}
			}
			// Embed metadata into page for JS.
			b, _ := json.Marshal(meta)
			data.RuleMetaJSON = strings.TrimSpace(string(b))

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
					botName := triggerNode.Bot
					if botName == "system" {
						botName = "dev"
					}
					data.Trigger = botName + "|" + triggerNode.RuleID
					data.TriggerParams = component.JSONString(triggerNode.Parameters)
					if triggerNode.RuleID == "webhook" {
						if triggerNode.Parameters != nil {
							if tok, ok := triggerNode.Parameters["token"].(string); ok && tok != "" {
								data.WebhookURL = fmt.Sprintf("/flows/webhook/%d/%s", flow.ID, tok)
							}
						}
					}
				}
				_ = action2Node // Editor currently supports a single action.
				if action1Node != nil {
					botName := action1Node.Bot
					if botName == "system" {
						botName = "dev"
					}
					data.Action = botName + "|" + action1Node.RuleID
					data.ActionParams = component.JSONString(action1Node.Parameters)
				} else {
					data.ActionParams = "{}"
				}
			}
			if strings.TrimSpace(data.TriggerParams) == "" {
				data.TriggerParams = "{}"
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

func defaultParamsExampleFromSpecs(specs []flows.ParamSpec) types.KV {
	out := make(types.KV)
	for _, spec := range specs {
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			continue
		}
		switch spec.Type {
		case flows.ParamTypeString:
			if len(spec.Enum) > 0 {
				out[name] = spec.Enum[0]
			} else {
				out[name] = ""
			}
		case flows.ParamTypeNumber:
			out[name] = 0
		case flows.ParamTypeBool:
			out[name] = false
		case flows.ParamTypeObject:
			out[name] = map[string]any{}
		case flows.ParamTypeArray:
			out[name] = []any{}
		default:
			out[name] = ""
		}
	}
	return out
}

func defaultTriggerParamsExample(r trigger.Rule) types.KV {
	// Keep this intentionally conservative: unknown triggers get an empty object.
	// A few built-in/dev-friendly examples are provided to guide users.
	switch r.Id {
	case "webhook":
		return types.KV{
			"ingredients": []any{
				map[string]any{"name": "example", "path": "payload.foo"},
			},
		}
	case "http_poll":
		return types.KV{
			"url":              "https://example.com/api/items",
			"items_path":       "data.items",
			"id_path":          "id",
			"status_path":      "status",
			"from_status":      "",
			"to_status":        "",
			"method":           "GET",
			"headers":          map[string]any{},
			"interval_seconds": 60,
			"ingredients": []any{
				map[string]any{"name": "id", "path": "payload.id"},
				map[string]any{"name": "status", "path": "item.status"},
			},
		}
	default:
		return types.KV{}
	}
}

func flowGraphToChain(nodes []*model.FlowNode, edges []*model.FlowEdge) (triggerNode *model.FlowNode, action1 *model.FlowNode, action2 *model.FlowNode) {
	if len(nodes) == 0 {
		return nil, nil, nil
	}
	nodeMap := make(map[string]*model.FlowNode, len(nodes))
	for _, n := range nodes {
		nodeMap[n.NodeID] = n
		if triggerNode == nil && n.Type == model.NodeTypeTrigger {
			triggerNode = n
		}
	}
	if triggerNode == nil {
		return nil, nil, nil
	}

	outgoing := make(map[string][]string)
	for _, e := range edges {
		outgoing[e.SourceNode] = append(outgoing[e.SourceNode], e.TargetNode)
	}

	firstTargets := outgoing[triggerNode.NodeID]
	for _, tid := range firstTargets {
		n := nodeMap[tid]
		if n != nil && n.Type == model.NodeTypeAction {
			action1 = n
			break
		}
	}
	if action1 == nil {
		return triggerNode, nil, nil
	}

	secondTargets := outgoing[action1.NodeID]
	for _, tid := range secondTargets {
		n := nodeMap[tid]
		if n != nil && n.Type == model.NodeTypeAction {
			action2 = n
			break
		}
	}
	return triggerNode, action1, action2
}

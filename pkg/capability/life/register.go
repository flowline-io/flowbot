package life

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Register registers hub.CapLife operations.
func Register(svc Service) error {
	if svc == nil {
		svc = NewLLM()
	}
	return capability.Register(capability.Spec{
		Type:        hub.CapLife,
		Description: "Life AI dungeon master: quest evaluation and item lore",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name:        OpEvaluateQuest,
				Description: "Evaluate a free-text quest prompt",
				Mutation:    false,
				Input: []hub.ParamDef{
					{Name: "prompt", Type: "string", Required: true, Description: "Quest description"},
					{Name: "ai_personality", Type: "string", Required: false, Description: "DM personality"},
					{Name: "completion_rate", Type: "number", Required: false, Description: "Historical completion rate"},
					{Name: "mood", Type: "object", Required: false, Description: "Recent mood JSON"},
					{Name: "privileges", Type: "object", Required: false, Description: "Equipped AI privileges"},
					{Name: "active_goals", Type: "array", Required: false, Description: "Active goal titles"},
					{Name: "breakdown_depth", Type: "string", Required: false, Description: "AI breakdown depth hint"},
				},
				Handler: invokeEvaluate(svc),
			},
			{
				Name:        OpGenerateInstanceLore,
				Description: "Generate instance lore for a dropped item",
				Mutation:    false,
				Input: []hub.ParamDef{
					{Name: "quest_title", Type: "string", Required: false, Description: "Source quest title"},
					{Name: "equipment_name", Type: "string", Required: true, Description: "Template equipment name"},
					{Name: "rarity", Type: "string", Required: false, Description: "Rarity label"},
				},
				Handler: invokeLore(svc),
			},
		},
	})
}

func invokeEvaluate(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		prompt, err := capability.RequiredString(params, "prompt")
		if err != nil {
			return nil, err
		}
		req := EvaluateQuestRequest{
			Prompt:         prompt,
			AIPersonality:  optionalStringParam(params, "ai_personality"),
			BreakdownDepth: optionalStringParam(params, "breakdown_depth"),
		}
		if v, ok := params["completion_rate"].(float64); ok {
			req.CompletionRate = v
		}
		if m, ok := params["mood"].(map[string]any); ok {
			req.Mood = m
		}
		if m, ok := params["privileges"].(map[string]any); ok {
			req.Privileges = m
		}
		if arr, ok := params["active_goals"].([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok && s != "" {
					req.ActiveGoals = append(req.ActiveGoals, s)
				}
			}
		}
		if arr, ok := params["active_goals"].([]string); ok {
			req.ActiveGoals = append(req.ActiveGoals, arr...)
		}
		ev, err := svc.EvaluateQuest(ctx, req)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ev}, nil
	}
}

func invokeLore(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		equip, err := capability.RequiredString(params, "equipment_name")
		if err != nil {
			return nil, err
		}
		questTitle := optionalStringParam(params, "quest_title")
		rarity := optionalStringParam(params, "rarity")
		if rarity == "" {
			rarity = "Common"
		}
		lore, err := svc.GenerateInstanceLore(ctx, questTitle, equip, rarity)
		if err != nil {
			return nil, fmt.Errorf("life capability: lore: %w", err)
		}
		return &capability.InvokeResult{Data: lore}, nil
	}
}

func optionalStringParam(params map[string]any, key string) string {
	v, ok := params[key].(string)
	if !ok {
		return ""
	}
	return v
}

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
				Name:        OpAdjudicateQuest,
				Description: "Adjudicate quest evidence into a suggested ruling",
				Mutation:    false,
				Input: []hub.ParamDef{
					{Name: "quest_title", Type: "string", Required: true, Description: "Quest title"},
					{Name: "quest_prompt", Type: "string", Required: false, Description: "Original quest prompt"},
					{Name: "quest_type", Type: "string", Required: false, Description: "Quest type"},
					{Name: "difficulty", Type: "string", Required: false, Description: "Quest difficulty"},
					{Name: "base_exp", Type: "number", Required: false, Description: "Quest base exp"},
					{Name: "base_gold", Type: "number", Required: false, Description: "Quest base gold"},
					{Name: "ai_personality", Type: "string", Required: false, Description: "DM personality"},
					{Name: "completion_rate", Type: "number", Required: false, Description: "Historical completion rate"},
					{Name: "mood", Type: "object", Required: false, Description: "Recent mood JSON"},
					{Name: "active_goals", Type: "array", Required: false, Description: "Active goal titles"},
					{Name: "recent_action_log", Type: "array", Required: false, Description: "Recent action log summaries"},
					{Name: "evidence", Type: "array", Required: false, Description: "Quest evidence items"},
				},
				Handler: invokeAdjudicate(svc),
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
			{
				Name:        OpBreakdownGoalTree,
				Description: "Suggest a structured life plan tree",
				Mutation:    false,
				Input: []hub.ParamDef{
					{Name: "root_title", Type: "string", Required: true, Description: "Root goal title"},
					{Name: "description", Type: "string", Required: false, Description: "Goal detail"},
					{Name: "ai_personality", Type: "string", Required: false, Description: "DM personality"},
					{Name: "privileges", Type: "object", Required: false, Description: "Equipped AI privileges"},
					{Name: "active_goals", Type: "array", Required: false, Description: "Active goal titles"},
					{Name: "breakdown_depth", Type: "string", Required: false, Description: "AI breakdown depth hint"},
				},
				Handler: invokeBreakdown(svc),
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

func invokeAdjudicate(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		title, err := capability.RequiredString(params, "quest_title")
		if err != nil {
			return nil, err
		}
		req := AdjudicateQuestRequest{
			QuestTitle:    title,
			QuestPrompt:   optionalStringParam(params, "quest_prompt"),
			QuestType:     optionalStringParam(params, "quest_type"),
			Difficulty:    optionalStringParam(params, "difficulty"),
			AIPersonality: optionalStringParam(params, "ai_personality"),
		}
		if v, ok := capability.IntParam(params, "base_exp"); ok {
			req.BaseExp = v
		}
		if v, ok := capability.IntParam(params, "base_gold"); ok {
			req.BaseGold = v
		}
		if v, ok := params["completion_rate"].(float64); ok {
			req.CompletionRate = v
		}
		if m, ok := params["mood"].(map[string]any); ok {
			req.Mood = m
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
		req.RecentActionLog = appendActionLogParam(req.RecentActionLog, params["recent_action_log"])
		req.Evidence = appendEvidenceParam(req.Evidence, params["evidence"])
		adjudication, err := svc.AdjudicateQuest(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("life capability: adjudicate: %w", err)
		}
		return &capability.InvokeResult{Data: adjudication}, nil
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

func invokeBreakdown(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		title, err := capability.RequiredString(params, "root_title")
		if err != nil {
			return nil, err
		}
		req := GoalBreakdownRequest{
			RootTitle:      title,
			Description:    optionalStringParam(params, "description"),
			AIPersonality:  optionalStringParam(params, "ai_personality"),
			BreakdownDepth: optionalStringParam(params, "breakdown_depth"),
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
		tree, err := svc.BreakdownGoalTree(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("life capability: breakdown: %w", err)
		}
		return &capability.InvokeResult{Data: tree}, nil
	}
}

func optionalStringParam(params map[string]any, key string) string {
	v, ok := params[key].(string)
	if !ok {
		return ""
	}
	return v
}

func appendActionLogParam(dst []map[string]any, raw any) []map[string]any {
	switch items := raw.(type) {
	case []map[string]any:
		return append(dst, items...)
	case []any:
		return appendActionLogMaps(dst, items)
	default:
		return dst
	}
}

func appendActionLogMaps(dst []map[string]any, items []any) []map[string]any {
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			dst = append(dst, m)
		}
	}
	return dst
}

func appendEvidenceParam(dst []QuestEvidence, raw any) []QuestEvidence {
	switch items := raw.(type) {
	case []map[string]any:
		for _, m := range items {
			dst = append(dst, questEvidenceFromMap(m))
		}
		return dst
	case []any:
		return appendEvidence(dst, items)
	default:
		return dst
	}
}

func appendEvidence(dst []QuestEvidence, items []any) []QuestEvidence {
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		dst = append(dst, questEvidenceFromMap(m))
	}
	return dst
}

func questEvidenceFromMap(m map[string]any) QuestEvidence {
	return QuestEvidence{
		SourceType: optionalStringFromMap(m, "source_type"),
		Content:    optionalStringFromMap(m, "content"),
		SourceURL:  optionalStringFromMap(m, "source_url"),
		Summary:    optionalStringFromMap(m, "summary"),
	}
}

func optionalStringFromMap(m map[string]any, key string) string {
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}

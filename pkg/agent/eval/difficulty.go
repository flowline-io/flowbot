package eval

import (
	"fmt"
	"strings"
)

// Difficulty levels for the capability ladder.
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// NormalizeDifficulty maps empty/unknown values to easy|medium|hard.
func NormalizeDifficulty(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", DifficultyEasy:
		return DifficultyEasy
	case DifficultyMedium:
		return DifficultyMedium
	case DifficultyHard:
		return DifficultyHard
	default:
		return DifficultyEasy
	}
}

// FilterByDifficulty keeps scenarios matching spec.
// Spec examples: "hard", "medium+", "easy,hard", empty (all).
func FilterByDifficulty(scenarios []Scenario, spec string) ([]Scenario, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return scenarios, nil
	}
	want, err := parseDifficultySpec(spec)
	if err != nil {
		return nil, err
	}
	out := make([]Scenario, 0, len(scenarios))
	for _, sc := range scenarios {
		if _, ok := want[NormalizeDifficulty(sc.Difficulty)]; ok {
			out = append(out, sc)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("eval: no cases matched --difficulty %q", spec)
	}
	return out, nil
}

func parseDifficultySpec(spec string) (map[string]struct{}, error) {
	want := make(map[string]struct{})
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		switch {
		case part == "easy+" || part == "all":
			want[DifficultyEasy] = struct{}{}
			want[DifficultyMedium] = struct{}{}
			want[DifficultyHard] = struct{}{}
		case part == "medium+":
			want[DifficultyMedium] = struct{}{}
			want[DifficultyHard] = struct{}{}
		case part == "hard+":
			want[DifficultyHard] = struct{}{}
		case part == DifficultyEasy, part == DifficultyMedium, part == DifficultyHard:
			want[part] = struct{}{}
		default:
			return nil, fmt.Errorf("eval: invalid --difficulty %q (use easy|medium|hard|medium+|hard+|easy,hard)", part)
		}
	}
	if len(want) == 0 {
		return nil, fmt.Errorf("eval: empty --difficulty spec")
	}
	return want, nil
}

package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/functiondefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/functionrun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/pkg/types"
)

// OwnedRunActivity returns daily pipeline run counts for definitions owned by uid.
func (s *PipelineStore) OwnedRunActivity(ctx context.Context, uid string, since, until time.Time) ([]types.ActivityStatsPoint, int64, error) {
	if s == nil || s.client == nil {
		return nil, 0, nil
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return fillActivityPoints(nil, since, until), 0, nil
	}
	names, err := s.client.PipelineDefinition.Query().
		Where(pipelinedefinition.CreatedByEQ(uid)).
		Select(pipelinedefinition.FieldName).
		Strings(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list owned pipelines: %w", err)
	}
	daily := map[string]int64{}
	var total int64
	if len(names) > 0 {
		rows, err := s.client.PipelineRun.Query().
			Where(
				pipelinerun.PipelineNameIn(names...),
				pipelinerun.StartedAtGTE(since),
				pipelinerun.StartedAtLTE(until),
			).
			All(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("store: list owned pipeline runs: %w", err)
		}
		total = int64(len(rows))
		for _, row := range rows {
			day := startOfUTCDayStore(row.StartedAt).Format("2006-01-02")
			daily[day]++
		}
	}
	return fillActivityPoints(daily, since, until), total, nil
}

// OwnedRunActivity returns daily function run counts for definitions owned by uid.
func (s *FunctionStore) OwnedRunActivity(ctx context.Context, uid string, since, until time.Time) ([]types.ActivityStatsPoint, int64, error) {
	if s == nil || s.client == nil {
		return nil, 0, nil
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return fillActivityPoints(nil, since, until), 0, nil
	}
	fnNames, err := s.client.FunctionDefinition.Query().
		Where(functiondefinition.CreatedByEQ(uid)).
		Select(functiondefinition.FieldName).
		Strings(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list owned functions: %w", err)
	}
	daily := map[string]int64{}
	var total int64
	if len(fnNames) > 0 {
		rows, err := s.client.FunctionRun.Query().
			Where(
				functionrun.FunctionNameIn(fnNames...),
				functionrun.CreatedAtGTE(since),
				functionrun.CreatedAtLTE(until),
			).
			All(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("store: list owned function runs: %w", err)
		}
		total = int64(len(rows))
		for _, row := range rows {
			day := startOfUTCDayStore(row.CreatedAt).Format("2006-01-02")
			daily[day]++
		}
	}
	return fillActivityPoints(daily, since, until), total, nil
}

func fillActivityPoints(daily map[string]int64, since, until time.Time) []types.ActivityStatsPoint {
	startDay := startOfUTCDayStore(since)
	endDay := startOfUTCDayStore(until)
	points := make([]types.ActivityStatsPoint, 0)
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		points = append(points, types.ActivityStatsPoint{Date: day, Daily: daily[day]})
	}
	return points
}

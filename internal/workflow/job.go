package workflow

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/utils"
	jsoniter "github.com/json-iterator/go"
)

const (
	jobListKey = "workflow:jobs"
)

func SyncJob(ctx context.Context, job *model.Job) error {
	return rdb.Client.HSet(ctx, jobListKey, strconv.FormatInt(job.ID, 10), job).Err()
}

func DeleteJob(ctx context.Context, job *model.Job) error {
	return rdb.Client.HDel(ctx, jobListKey, strconv.FormatInt(job.ID, 10)).Err()
}

func GetJobsByState(ctx context.Context, state model.JobState) ([]*model.Job, error) {
	res, err := rdb.Client.HGetAll(ctx, jobListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs from cache key %s, %w", jobListKey, err)
	}
	var jobs []*model.Job
	for _, v := range res {
		job := &model.Job{}
		err = jsoniter.Unmarshal(utils.StringToBytes(v), job)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal job %s, %w", v, err)
		}
		jobs = append(jobs, job)
	}

	var list []*model.Job
	for _, job := range jobs {
		if job.State == state {
			list = append(list, job)
		}
	}
	return list, nil
}

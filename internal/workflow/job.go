package workflow

import (
	"context"
	"strconv"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

const (
	jobListKey = "workflow:jobs"
)

func SyncJob(ctx context.Context, job *model.Job) error {
	return cache.DB.HSet(ctx, jobListKey, strconv.FormatInt(job.ID, 10), job).Err()
}

func DeleteJob(ctx context.Context, job *model.Job) error {
	return cache.DB.HDel(ctx, jobListKey, strconv.FormatInt(job.ID, 10)).Err()
}

func GetJobsByState(ctx context.Context, state model.JobState) ([]*model.Job, error) {
	res, err := cache.DB.HGetAll(ctx, jobListKey).Result()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get jobs from cache key %s", jobListKey)
	}
	var jobs []*model.Job
	for _, v := range res {
		job := &model.Job{}
		err = json.Unmarshal(utils.StringToBytes(v), job)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal job %s", v)
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

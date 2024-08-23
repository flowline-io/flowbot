package leetcode

import (
	"context"
	_ "embed"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	jsoniter "github.com/json-iterator/go"
)

//go:embed problems.json
var problems []byte

type ProblemResult struct {
	UserName        string    `json:"user_name"`
	NumSolved       int       `json:"num_solved"`
	NumTotal        int       `json:"num_total"`
	AcEasy          int       `json:"ac_easy"`
	AcMedium        int       `json:"ac_medium"`
	AcHard          int       `json:"ac_hard"`
	StatStatusPairs []Problem `json:"stat_status_pairs"`
	FrequencyHigh   int       `json:"frequency_high"`
	FrequencyMid    int       `json:"frequency_mid"`
	CategorySlug    string    `json:"category_slug"`
}

type Problem struct {
	Stat struct {
		QuestionID                      int         `json:"question_id"`
		QuestionArticleLive             interface{} `json:"question__article__live"`
		QuestionArticleSlug             interface{} `json:"question__article__slug"`
		QuestionArticleHasVideoSolution interface{} `json:"question__article__has_video_solution"`
		QuestionTitle                   string      `json:"question__title"`
		QuestionTitleSlug               string      `json:"question__title_slug"`
		QuestionHide                    bool        `json:"question__hide"`
		TotalAcs                        int         `json:"total_acs"`
		TotalSubmitted                  int         `json:"total_submitted"`
		FrontendQuestionID              int         `json:"frontend_question_id"`
		IsNewQuestion                   bool        `json:"is_new_question"`
	} `json:"stat"`
	Status     interface{} `json:"status"`
	Difficulty struct {
		Level int `json:"level"`
	} `json:"difficulty"`
	PaidOnly  bool `json:"paid_only"`
	IsFavor   bool `json:"is_favor"`
	Frequency int  `json:"frequency"`
	Progress  int  `json:"progress"`
}

func (i Problem) MarshalBinary() ([]byte, error) {
	return jsoniter.Marshal(i)
}

func importProblems() error {
	var result ProblemResult
	err := jsoniter.Unmarshal(problems, &result)
	if err != nil {
		return nil
	}

	var easy []Problem
	var medium []Problem
	var hard []Problem
	for i, problem := range result.StatStatusPairs {
		switch problem.Difficulty.Level {
		case 1:
			easy = append(easy, result.StatStatusPairs[i])
		case 2:
			medium = append(medium, result.StatStatusPairs[i])
		case 3:
			hard = append(hard, result.StatStatusPairs[i])
		}
	}

	ctx := context.Background()
	pipe := cache.DB.Pipeline()

	pipe.Del(ctx, "leetcode:problems:easy")
	for i := range easy {
		pipe.SAdd(ctx, "leetcode:problems:easy", easy[i])
	}
	pipe.Del(ctx, "leetcode:problems:medium")
	for i := range medium {
		pipe.SAdd(ctx, "leetcode:problems:medium", medium[i])
	}
	pipe.Del(ctx, "leetcode:problems:hard")
	for i := range hard {
		pipe.SAdd(ctx, "leetcode:problems:hard", hard[i])
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		flog.Error(err)
	}
	return nil
}

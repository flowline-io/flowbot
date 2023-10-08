// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package dao

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"gorm.io/gen"

	"gorm.io/plugin/dbresolver"
)

var (
	Q                = new(Query)
	Action           *action
	Behavior         *behavior
	Bot              *bot
	Channel          *channel
	Config           *config
	Counter          *counter
	CounterRecord    *counterRecord
	Cycle            *cycle
	Dag              *dag
	Data             *data
	Fileupload       *fileupload
	Form             *form
	Instruct         *instruct
	Job              *job
	KeyResult        *keyResult
	KeyResultValue   *keyResultValue
	Message          *message
	OAuth            *oAuth
	Objective        *objective
	Page             *page
	Parameter        *parameter
	Pipeline         *pipeline
	Platform         *platform
	PlatformBot      *platformBot
	PlatformChannel  *platformChannel
	PlatformUser     *platformUser
	Review           *review
	ReviewEvaluation *reviewEvaluation
	SchemaMigration  *schemaMigration
	Session          *session
	Step             *step
	Todo             *todo
	Topic            *topic
	Url              *url
	User             *user
	Workflow         *workflow
	WorkflowTrigger  *workflowTrigger
)

func SetDefault(db *gorm.DB, opts ...gen.DOOption) {
	*Q = *Use(db, opts...)
	Action = &Q.Action
	Behavior = &Q.Behavior
	Bot = &Q.Bot
	Channel = &Q.Channel
	Config = &Q.Config
	Counter = &Q.Counter
	CounterRecord = &Q.CounterRecord
	Cycle = &Q.Cycle
	Dag = &Q.Dag
	Data = &Q.Data
	Fileupload = &Q.Fileupload
	Form = &Q.Form
	Instruct = &Q.Instruct
	Job = &Q.Job
	KeyResult = &Q.KeyResult
	KeyResultValue = &Q.KeyResultValue
	Message = &Q.Message
	OAuth = &Q.OAuth
	Objective = &Q.Objective
	Page = &Q.Page
	Parameter = &Q.Parameter
	Pipeline = &Q.Pipeline
	Platform = &Q.Platform
	PlatformBot = &Q.PlatformBot
	PlatformChannel = &Q.PlatformChannel
	PlatformUser = &Q.PlatformUser
	Review = &Q.Review
	ReviewEvaluation = &Q.ReviewEvaluation
	SchemaMigration = &Q.SchemaMigration
	Session = &Q.Session
	Step = &Q.Step
	Todo = &Q.Todo
	Topic = &Q.Topic
	Url = &Q.Url
	User = &Q.User
	Workflow = &Q.Workflow
	WorkflowTrigger = &Q.WorkflowTrigger
}

func Use(db *gorm.DB, opts ...gen.DOOption) *Query {
	return &Query{
		db:               db,
		Action:           newAction(db, opts...),
		Behavior:         newBehavior(db, opts...),
		Bot:              newBot(db, opts...),
		Channel:          newChannel(db, opts...),
		Config:           newConfig(db, opts...),
		Counter:          newCounter(db, opts...),
		CounterRecord:    newCounterRecord(db, opts...),
		Cycle:            newCycle(db, opts...),
		Dag:              newDag(db, opts...),
		Data:             newData(db, opts...),
		Fileupload:       newFileupload(db, opts...),
		Form:             newForm(db, opts...),
		Instruct:         newInstruct(db, opts...),
		Job:              newJob(db, opts...),
		KeyResult:        newKeyResult(db, opts...),
		KeyResultValue:   newKeyResultValue(db, opts...),
		Message:          newMessage(db, opts...),
		OAuth:            newOAuth(db, opts...),
		Objective:        newObjective(db, opts...),
		Page:             newPage(db, opts...),
		Parameter:        newParameter(db, opts...),
		Pipeline:         newPipeline(db, opts...),
		Platform:         newPlatform(db, opts...),
		PlatformBot:      newPlatformBot(db, opts...),
		PlatformChannel:  newPlatformChannel(db, opts...),
		PlatformUser:     newPlatformUser(db, opts...),
		Review:           newReview(db, opts...),
		ReviewEvaluation: newReviewEvaluation(db, opts...),
		SchemaMigration:  newSchemaMigration(db, opts...),
		Session:          newSession(db, opts...),
		Step:             newStep(db, opts...),
		Todo:             newTodo(db, opts...),
		Topic:            newTopic(db, opts...),
		Url:              newUrl(db, opts...),
		User:             newUser(db, opts...),
		Workflow:         newWorkflow(db, opts...),
		WorkflowTrigger:  newWorkflowTrigger(db, opts...),
	}
}

type Query struct {
	db *gorm.DB

	Action           action
	Behavior         behavior
	Bot              bot
	Channel          channel
	Config           config
	Counter          counter
	CounterRecord    counterRecord
	Cycle            cycle
	Dag              dag
	Data             data
	Fileupload       fileupload
	Form             form
	Instruct         instruct
	Job              job
	KeyResult        keyResult
	KeyResultValue   keyResultValue
	Message          message
	OAuth            oAuth
	Objective        objective
	Page             page
	Parameter        parameter
	Pipeline         pipeline
	Platform         platform
	PlatformBot      platformBot
	PlatformChannel  platformChannel
	PlatformUser     platformUser
	Review           review
	ReviewEvaluation reviewEvaluation
	SchemaMigration  schemaMigration
	Session          session
	Step             step
	Todo             todo
	Topic            topic
	Url              url
	User             user
	Workflow         workflow
	WorkflowTrigger  workflowTrigger
}

func (q *Query) Available() bool { return q.db != nil }

func (q *Query) clone(db *gorm.DB) *Query {
	return &Query{
		db:               db,
		Action:           q.Action.clone(db),
		Behavior:         q.Behavior.clone(db),
		Bot:              q.Bot.clone(db),
		Channel:          q.Channel.clone(db),
		Config:           q.Config.clone(db),
		Counter:          q.Counter.clone(db),
		CounterRecord:    q.CounterRecord.clone(db),
		Cycle:            q.Cycle.clone(db),
		Dag:              q.Dag.clone(db),
		Data:             q.Data.clone(db),
		Fileupload:       q.Fileupload.clone(db),
		Form:             q.Form.clone(db),
		Instruct:         q.Instruct.clone(db),
		Job:              q.Job.clone(db),
		KeyResult:        q.KeyResult.clone(db),
		KeyResultValue:   q.KeyResultValue.clone(db),
		Message:          q.Message.clone(db),
		OAuth:            q.OAuth.clone(db),
		Objective:        q.Objective.clone(db),
		Page:             q.Page.clone(db),
		Parameter:        q.Parameter.clone(db),
		Pipeline:         q.Pipeline.clone(db),
		Platform:         q.Platform.clone(db),
		PlatformBot:      q.PlatformBot.clone(db),
		PlatformChannel:  q.PlatformChannel.clone(db),
		PlatformUser:     q.PlatformUser.clone(db),
		Review:           q.Review.clone(db),
		ReviewEvaluation: q.ReviewEvaluation.clone(db),
		SchemaMigration:  q.SchemaMigration.clone(db),
		Session:          q.Session.clone(db),
		Step:             q.Step.clone(db),
		Todo:             q.Todo.clone(db),
		Topic:            q.Topic.clone(db),
		Url:              q.Url.clone(db),
		User:             q.User.clone(db),
		Workflow:         q.Workflow.clone(db),
		WorkflowTrigger:  q.WorkflowTrigger.clone(db),
	}
}

func (q *Query) ReadDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Read))
}

func (q *Query) WriteDB() *Query {
	return q.ReplaceDB(q.db.Clauses(dbresolver.Write))
}

func (q *Query) ReplaceDB(db *gorm.DB) *Query {
	return &Query{
		db:               db,
		Action:           q.Action.replaceDB(db),
		Behavior:         q.Behavior.replaceDB(db),
		Bot:              q.Bot.replaceDB(db),
		Channel:          q.Channel.replaceDB(db),
		Config:           q.Config.replaceDB(db),
		Counter:          q.Counter.replaceDB(db),
		CounterRecord:    q.CounterRecord.replaceDB(db),
		Cycle:            q.Cycle.replaceDB(db),
		Dag:              q.Dag.replaceDB(db),
		Data:             q.Data.replaceDB(db),
		Fileupload:       q.Fileupload.replaceDB(db),
		Form:             q.Form.replaceDB(db),
		Instruct:         q.Instruct.replaceDB(db),
		Job:              q.Job.replaceDB(db),
		KeyResult:        q.KeyResult.replaceDB(db),
		KeyResultValue:   q.KeyResultValue.replaceDB(db),
		Message:          q.Message.replaceDB(db),
		OAuth:            q.OAuth.replaceDB(db),
		Objective:        q.Objective.replaceDB(db),
		Page:             q.Page.replaceDB(db),
		Parameter:        q.Parameter.replaceDB(db),
		Pipeline:         q.Pipeline.replaceDB(db),
		Platform:         q.Platform.replaceDB(db),
		PlatformBot:      q.PlatformBot.replaceDB(db),
		PlatformChannel:  q.PlatformChannel.replaceDB(db),
		PlatformUser:     q.PlatformUser.replaceDB(db),
		Review:           q.Review.replaceDB(db),
		ReviewEvaluation: q.ReviewEvaluation.replaceDB(db),
		SchemaMigration:  q.SchemaMigration.replaceDB(db),
		Session:          q.Session.replaceDB(db),
		Step:             q.Step.replaceDB(db),
		Todo:             q.Todo.replaceDB(db),
		Topic:            q.Topic.replaceDB(db),
		Url:              q.Url.replaceDB(db),
		User:             q.User.replaceDB(db),
		Workflow:         q.Workflow.replaceDB(db),
		WorkflowTrigger:  q.WorkflowTrigger.replaceDB(db),
	}
}

type queryCtx struct {
	Action           *actionDo
	Behavior         *behaviorDo
	Bot              *botDo
	Channel          *channelDo
	Config           *configDo
	Counter          *counterDo
	CounterRecord    *counterRecordDo
	Cycle            *cycleDo
	Dag              *dagDo
	Data             *dataDo
	Fileupload       *fileuploadDo
	Form             *formDo
	Instruct         *instructDo
	Job              *jobDo
	KeyResult        *keyResultDo
	KeyResultValue   *keyResultValueDo
	Message          *messageDo
	OAuth            *oAuthDo
	Objective        *objectiveDo
	Page             *pageDo
	Parameter        *parameterDo
	Pipeline         *pipelineDo
	Platform         *platformDo
	PlatformBot      *platformBotDo
	PlatformChannel  *platformChannelDo
	PlatformUser     *platformUserDo
	Review           *reviewDo
	ReviewEvaluation *reviewEvaluationDo
	SchemaMigration  *schemaMigrationDo
	Session          *sessionDo
	Step             *stepDo
	Todo             *todoDo
	Topic            *topicDo
	Url              *urlDo
	User             *userDo
	Workflow         *workflowDo
	WorkflowTrigger  *workflowTriggerDo
}

func (q *Query) WithContext(ctx context.Context) *queryCtx {
	return &queryCtx{
		Action:           q.Action.WithContext(ctx),
		Behavior:         q.Behavior.WithContext(ctx),
		Bot:              q.Bot.WithContext(ctx),
		Channel:          q.Channel.WithContext(ctx),
		Config:           q.Config.WithContext(ctx),
		Counter:          q.Counter.WithContext(ctx),
		CounterRecord:    q.CounterRecord.WithContext(ctx),
		Cycle:            q.Cycle.WithContext(ctx),
		Dag:              q.Dag.WithContext(ctx),
		Data:             q.Data.WithContext(ctx),
		Fileupload:       q.Fileupload.WithContext(ctx),
		Form:             q.Form.WithContext(ctx),
		Instruct:         q.Instruct.WithContext(ctx),
		Job:              q.Job.WithContext(ctx),
		KeyResult:        q.KeyResult.WithContext(ctx),
		KeyResultValue:   q.KeyResultValue.WithContext(ctx),
		Message:          q.Message.WithContext(ctx),
		OAuth:            q.OAuth.WithContext(ctx),
		Objective:        q.Objective.WithContext(ctx),
		Page:             q.Page.WithContext(ctx),
		Parameter:        q.Parameter.WithContext(ctx),
		Pipeline:         q.Pipeline.WithContext(ctx),
		Platform:         q.Platform.WithContext(ctx),
		PlatformBot:      q.PlatformBot.WithContext(ctx),
		PlatformChannel:  q.PlatformChannel.WithContext(ctx),
		PlatformUser:     q.PlatformUser.WithContext(ctx),
		Review:           q.Review.WithContext(ctx),
		ReviewEvaluation: q.ReviewEvaluation.WithContext(ctx),
		SchemaMigration:  q.SchemaMigration.WithContext(ctx),
		Session:          q.Session.WithContext(ctx),
		Step:             q.Step.WithContext(ctx),
		Todo:             q.Todo.WithContext(ctx),
		Topic:            q.Topic.WithContext(ctx),
		Url:              q.Url.WithContext(ctx),
		User:             q.User.WithContext(ctx),
		Workflow:         q.Workflow.WithContext(ctx),
		WorkflowTrigger:  q.WorkflowTrigger.WithContext(ctx),
	}
}

func (q *Query) Transaction(fc func(tx *Query) error, opts ...*sql.TxOptions) error {
	return q.db.Transaction(func(tx *gorm.DB) error { return fc(q.clone(tx)) }, opts...)
}

func (q *Query) Begin(opts ...*sql.TxOptions) *QueryTx {
	tx := q.db.Begin(opts...)
	return &QueryTx{Query: q.clone(tx), Error: tx.Error}
}

type QueryTx struct {
	*Query
	Error error
}

func (q *QueryTx) Commit() error {
	return q.db.Commit().Error
}

func (q *QueryTx) Rollback() error {
	return q.db.Rollback().Error
}

func (q *QueryTx) SavePoint(name string) error {
	return q.db.SavePoint(name).Error
}

func (q *QueryTx) RollbackTo(name string) error {
	return q.db.RollbackTo(name).Error
}

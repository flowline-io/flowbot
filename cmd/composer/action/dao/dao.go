package dao

import (
	"context"
	"io"
	"os"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/goccy/go-yaml"
	"github.com/urfave/cli/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type Querier interface {
	// GetByID
	//
	// SELECT * FROM @@table WHERE id=@id
	GetByID(id int) (gen.T, error)

	// TakeByUidAndTopic query data by uid and topic and return one
	//
	// where("uid=@uid AND topic=@topic")
	TakeByUidAndTopic(uid, topic string) (gen.T, error)

	// FindByUidAndTopic query data by uid and topic and return array
	//
	// where("uid=@uid AND topic=@topic")
	FindByUidAndTopic(uid, topic string) ([]*gen.T, error)
}

func GenerationAction(ctx context.Context, c *cli.Command) error {
	conffile := c.String("config")

	file, err := os.Open(conffile)
	if err != nil {
		flog.Panic(err.Error())
	}

	config := configType{}

	data, err := io.ReadAll(file)
	if err != nil {
		flog.Panic(err.Error())
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		flog.Panic(err.Error())
	}

	if config.StoreConfig.UseAdapter != "mysql" {
		flog.Panic("error adapter")
	}
	if config.StoreConfig.Adapters.Mysql.DSN == "" {
		flog.Panic("error adapter dsn")
	}
	dsn := config.StoreConfig.Adapters.Mysql.DSN

	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		flog.Panic(err.Error())
	}

	g := gen.NewGenerator(gen.Config{
		OutPath: "./internal/store/dao",
		Mode:    gen.WithoutContext | gen.WithDefaultQuery,
	})

	g.UseDB(db)

	// chatbot table
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("behavior", "Behavior",
		gen.FieldType("extra", "*JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("configs", "Config",
		gen.FieldType("value", "JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("counter_records", "CounterRecord"))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("counters", "Counter"))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("data", "Data",
		gen.FieldType("value", "JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("form", "Form",
		gen.FieldType("schema", "JSON"),
		gen.FieldType("values", "JSON"),
		gen.FieldType("extra", "JSON"),
		gen.FieldType("state", "FormState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("instruct", "Instruct",
		gen.FieldType("object", "InstructObject"),
		gen.FieldType("content", "JSON"),
		gen.FieldType("priority", "InstructPriority"),
		gen.FieldType("state", "InstructState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("oauth", "OAuth",
		gen.FieldType("extra", "JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("webhook", "Webhook",
		gen.FieldType("state", "WebhookState")))

	// OKR
	todos := g.GenerateModelAs("todos", "Todo", gen.FieldRelate(field.HasMany, "SubTodos",
		g.GenerateModelAs("todos", "Todo"), &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"parent_id"},
			},
		}))
	keyResultValues := g.GenerateModelAs("key_result_values", "KeyResultValue")
	reviewEvaluations := g.GenerateModelAs("review_evaluations", "ReviewEvaluation")
	reviews := g.GenerateModelAs("reviews", "Review",
		gen.FieldType("type", "ReviewType"),
		gen.FieldRelate(field.HasMany, "Evaluations", reviewEvaluations, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"review_id"},
			},
		}))
	cycles := g.GenerateModelAs("cycles", "Cycle", gen.FieldType("state", "CycleState"))
	keyResults := g.GenerateModelAs("key_results", "KeyResult",
		gen.FieldType("value_mode", "ValueModeType"),
		gen.FieldRelate(field.HasMany, "KeyResultValues", keyResultValues, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"key_result_id"},
			},
		}),
		gen.FieldRelate(field.HasMany, "Todos", todos, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"key_result_id"},
			},
		}))
	objectives := g.GenerateModelAs("objectives", "Objective",
		gen.FieldRelate(field.HasMany, "KeyResults", keyResults, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"objective_id"},
			},
		}),
		gen.FieldRelate(field.HasMany, "Reviews", reviews, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"objective_id"},
			},
		}),
	)
	g.ApplyInterface(func(Querier) {}, objectives, keyResults, keyResultValues, todos, cycles, reviews, reviewEvaluations)

	// workflow
	dag := g.GenerateModelAs("dag", "Dag",
		gen.FieldType("nodes", "[]*Node"),
		gen.FieldGORMTag("nodes", func(tag field.GormTag) field.GormTag {
			return map[string][]string{
				"column":     {"nodes"},
				"type":       {"json"},
				"serializer": {"json"},
				"not null":   nil,
			}
		}),
		gen.FieldType("edges", "[]*Edge"),
		gen.FieldGORMTag("edges", func(tag field.GormTag) field.GormTag {
			return map[string][]string{
				"column":     {"edges"},
				"type":       {"json"},
				"serializer": {"json"},
				"not null":   nil,
			}
		}))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("workflow_script", "WorkflowScript",
		gen.FieldType("lang", "WorkflowScriptLang")))
	workflowTriggers := g.GenerateModelAs("workflow_trigger", "WorkflowTrigger",
		gen.FieldType("type", "TriggerType"),
		gen.FieldType("rule", "JSON"),
		gen.FieldType("state", "WorkflowTriggerState"))
	workflows := g.GenerateModelAs("workflow", "Workflow",
		gen.FieldType("state", "WorkflowState"),
		gen.FieldRelate(field.HasOne, "Dag", dag, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"workflow_id"},
			},
		}),
		gen.FieldRelate(field.HasMany, "Triggers", workflowTriggers, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"workflow_id"},
			},
		}))
	steps := g.GenerateModelAs("steps", "Step",
		gen.FieldType("depend", "[]string"),
		gen.FieldGORMTag("depend", func(tag field.GormTag) field.GormTag {
			return map[string][]string{
				"column":     {"depend"},
				"type":       {"json"},
				"serializer": {"json"},
				"not null":   nil,
			}
		}),
		gen.FieldType("action", "JSON"),
		gen.FieldType("input", "JSON"),
		gen.FieldType("output", "JSON"),
		gen.FieldType("state", "StepState"),
		gen.FieldType("started_at", "*time.Time"),
		gen.FieldType("ended_at", "*time.Time"))
	jobs := g.GenerateModelAs("jobs", "Job",
		gen.FieldType("state", "JobState"),
		gen.FieldType("started_at", "*time.Time"),
		gen.FieldType("ended_at", "*time.Time"),
		gen.FieldRelate(field.HasMany, "Steps", steps, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"job_id"},
			},
		}))
	g.ApplyInterface(func(Querier) {}, workflows, dag, steps, jobs, workflowTriggers)

	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("pages", "Page",
		gen.FieldType("type", "PageType"),
		gen.FieldType("schema", "JSON"),
		gen.FieldType("state", "PageState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("parameter", "Parameter",
		gen.FieldType("params", "JSON")))

	// platform table
	g.ApplyBasic(g.GenerateModel("users",
		gen.FieldType("state", "UserState")))
	g.ApplyBasic(g.GenerateModel("bots",
		gen.FieldType("state", "BotState")))
	g.ApplyBasic(g.GenerateModel("topics",
		gen.FieldType("state", "TopicState")))
	g.ApplyBasic(g.GenerateModel("messages",
		gen.FieldType("content", "JSON"),
		gen.FieldType("state", "MessageState")))
	g.ApplyBasic(g.GenerateModel("platforms"))
	g.ApplyBasic(g.GenerateModel("platform_users"))
	g.ApplyBasic(g.GenerateModel("platform_bots"))
	g.ApplyBasic(g.GenerateModel("fileuploads",
		gen.FieldType("state", "FileState")))
	g.ApplyBasic(g.GenerateModel("channels",
		gen.FieldType("state", "ChannelState")))
	g.ApplyBasic(g.GenerateModel("platform_channels"))
	g.ApplyBasic(g.GenerateModel("platform_channel_users"))
	g.ApplyBasic(g.GenerateModel("agents"))

	g.ApplyBasic(g.GenerateModel("schema_migrations"))

	// execute
	g.Execute()

	return nil
}

type configType struct {
	StoreConfig struct {
		UseAdapter string `json:"use_adapter" yaml:"use_adapter"`
		Adapters   struct {
			Mysql struct {
				DSN string `json:"dsn" yaml:"dsn"`
			} `json:"mysql" yaml:"mysql"`
		} `json:"adapters" yaml:"adapters"`
	} `json:"store_config" yaml:"store_config"`
}

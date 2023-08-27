package dao

import (
	"encoding/json"
	"github.com/tinode/jsonco"
	"github.com/urfave/cli/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"os"
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

func GenerationAction(c *cli.Context) error {
	conffile := c.String("config")

	file, err := os.Open(conffile)
	if err != nil {
		panic(err)
	}

	config := configType{}
	jr := jsonco.New(file)
	if err = json.NewDecoder(jr).Decode(&config); err != nil {
		panic(err)
	}

	if config.StoreConfig.UseAdapter != "mysql" {
		panic("error adapter")
	}
	if config.StoreConfig.Adapters.Mysql.DSN == "" {
		panic("error adapter dsn")
	}
	dsn := config.StoreConfig.Adapters.Mysql.DSN

	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(err)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath: "./server/extra/store/dao",
		Mode:    gen.WithoutContext | gen.WithDefaultQuery,
	})

	g.UseDB(db)

	// chatbot table
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_action", "Action",
		gen.FieldType("state", "ActionState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_behavior", "Behavior",
		gen.FieldType("extra", "*JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_configs", "Config",
		gen.FieldType("value", "JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_counter_records", "CounterRecord"))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_counters", "Counter"))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_data", "Data",
		gen.FieldType("value", "JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_form", "Form",
		gen.FieldType("schema", "JSON"),
		gen.FieldType("values", "JSON"),
		gen.FieldType("extra", "JSON"),
		gen.FieldType("state", "FormState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_instruct", "Instruct",
		gen.FieldType("object", "InstructObject"),
		gen.FieldType("content", "JSON"),
		gen.FieldType("priority", "InstructPriority"),
		gen.FieldType("state", "InstructState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_oauth", "OAuth",
		gen.FieldType("extra", "JSON")))

	// OKR
	todos := g.GenerateModelAs("chatbot_todos", "Todo", gen.FieldRelate(field.HasMany, "SubTodos",
		g.GenerateModelAs("chatbot_todos", "Todo"), &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"parent_id"},
			},
		}))
	keyResultValues := g.GenerateModelAs("chatbot_key_result_values", "KeyResultValue")
	reviewEvaluations := g.GenerateModelAs("chatbot_review_evaluations", "ReviewEvaluation")
	reviews := g.GenerateModelAs("chatbot_reviews", "Review",
		gen.FieldType("type", "ReviewType"),
		gen.FieldRelate(field.HasMany, "Evaluations", reviewEvaluations, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"review_id"},
			},
		}))
	cycles := g.GenerateModelAs("chatbot_cycles", "Cycle", gen.FieldType("state", "CycleState"))
	keyResults := g.GenerateModelAs("chatbot_key_results", "KeyResult",
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
	objectives := g.GenerateModelAs("chatbot_objectives", "Objective",
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
		}))
	g.ApplyInterface(func(Querier) {}, objectives, keyResults, keyResultValues, todos, cycles, reviews, reviewEvaluations)

	// workflow
	dag := g.GenerateModelAs("chatbot_dag", "Dag",
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
	workflowTriggers := g.GenerateModelAs("chatbot_workflow_trigger", "WorkflowTrigger",
		gen.FieldType("type", "TriggerType"))
	workflows := g.GenerateModelAs("chatbot_workflow", "Workflow",
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
	steps := g.GenerateModelAs("chatbot_steps", "Step",
		gen.FieldType("depend", "IDList"),
		gen.FieldType("state", "StepState"))
	jobs := g.GenerateModelAs("chatbot_jobs", "Job",
		gen.FieldType("state", "JobState"),
		gen.FieldRelate(field.HasMany, "Steps", steps, &field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"job_id"},
			},
		}))
	g.ApplyInterface(func(Querier) {}, workflows, dag, steps, jobs, workflowTriggers)

	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_page", "Page",
		gen.FieldType("type", "PageType"),
		gen.FieldType("schema", "JSON"),
		gen.FieldType("state", "PageState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_parameter", "Parameter",
		gen.FieldType("params", "JSON")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_session", "Session",
		gen.FieldType("init", "JSON"),
		gen.FieldType("values", "JSON"),
		gen.FieldType("state", "SessionState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_url", "Url",
		gen.FieldType("state", "UrlState")))
	g.ApplyInterface(func(Querier) {}, g.GenerateModelAs("chatbot_pipelines", "Pipeline",
		gen.FieldType("values", "JSON"),
		gen.FieldType("state", "PipelineState")))

	// tinode table
	g.ApplyBasic(g.GenerateModel("users",
		gen.FieldType("access", "JSON"),
		gen.FieldType("public", "JSON"),
		gen.FieldType("trusted", "JSON"),
		gen.FieldNew("Fn", "string", map[string]string{
			"json": "fn,omitempty",
		}),
		gen.FieldNew("Verified", "bool", map[string]string{
			"json": "verified,omitempty",
		})))
	g.ApplyBasic(g.GenerateModel("topics",
		gen.FieldType("access", "JSON"),
		gen.FieldType("public", "JSON"),
		gen.FieldType("trusted", "JSON"),
		gen.FieldNew("Fn", "string", map[string]string{
			"json": "fn,omitempty",
		}),
		gen.FieldNew("Verified", "bool", map[string]string{
			"json": "verified,omitempty",
		})))
	g.ApplyBasic(g.GenerateModel("messages",
		gen.FieldType("head", "JSON"),
		gen.FieldType("content", "JSON"),
		gen.FieldNew("Txt", "string", map[string]string{
			"json": "txt,omitempty",
		}),
		gen.FieldNew("Raw", "json.RawMessage", map[string]string{
			"json": "raw,omitempty",
		})))
	g.ApplyBasic(g.GenerateModel("credentials"))
	g.ApplyBasic(g.GenerateModel("auth"))
	g.ApplyBasic(g.GenerateModel("dellog"))
	g.ApplyBasic(g.GenerateModel("devices"))
	g.ApplyBasic(g.GenerateModel("filemsglinks"))
	g.ApplyBasic(g.GenerateModel("fileuploads"))
	g.ApplyBasic(g.GenerateModel("kvmeta"))
	g.ApplyBasic(g.GenerateModel("subscriptions"))
	g.ApplyBasic(g.GenerateModel("topictags"))
	g.ApplyBasic(g.GenerateModel("usertags"))

	g.ApplyBasic(g.GenerateModel("schema_migrations"))

	// execute
	g.Execute()

	return nil
}

type configType struct {
	StoreConfig struct {
		UseAdapter string `json:"use_adapter"`
		Adapters   struct {
			Mysql struct {
				DSN string `json:"dsn"`
			} `json:"mysql"`
		} `json:"adapters"`
	} `json:"store_config"`
}

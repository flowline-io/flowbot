package workflow

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/flc1125/go-cron/v4"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	fbtrace "github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

// Catalog loads and mutates workflow definitions stored in the database.
type Catalog interface {
	DefinitionStore
	ApplyDefinition(ctx context.Context, meta *types.WorkflowMetadata) (*gen.Workflow, error)
	ListDefinitions(ctx context.Context) ([]*gen.Workflow, error)
	DeleteDefinitionByName(ctx context.Context, name string) error
	ListRunsByName(ctx context.Context, name string) ([]*gen.WorkflowRun, error)
}

// WebhookEndpoint describes a registered workflow webhook trigger.
type WebhookEndpoint struct {
	WorkflowName string
	Config       *pipeline.WebhookConfig
}

// Service orchestrates DB-backed workflow apply/run and trigger registration.
type Service struct {
	catalog Catalog
	runs    WorkflowRunStore
	auditor audit.Auditor
	metrics *metrics.WorkflowCollector

	mu       sync.RWMutex
	cron     *cron.Cron
	webhooks map[string]*WebhookEndpoint // path -> endpoint
}

// NewService creates a workflow Service.
func NewService(catalog Catalog, runs WorkflowRunStore, auditor audit.Auditor, wc *metrics.WorkflowCollector) *Service {
	return &Service{
		catalog:  catalog,
		runs:     runs,
		auditor:  auditor,
		metrics:  wc,
		webhooks: make(map[string]*WebhookEndpoint),
	}
}

// StartRunAsync validates inputs, creates a run record, and executes the workflow in a goroutine.
// It returns the new run ID immediately.
func (s *Service) StartRunAsync(ctx context.Context, name, triggerType string, input types.KV) (int64, error) {
	if s == nil || s.catalog == nil {
		return 0, types.Errorf(types.ErrUnavailable, "workflow service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, types.Errorf(types.ErrInvalidArgument, "workflow name is required")
	}
	if triggerType == "" {
		triggerType = "manual"
	}
	if input == nil {
		input = types.KV{}
	}

	meta, err := s.catalog.GetMetadata(ctx, name)
	if err != nil {
		return 0, err
	}
	if meta == nil {
		return 0, types.Errorf(types.ErrNotFound, "workflow %s", name)
	}
	input = ApplyInputDefaults(meta.Inputs, input)
	if err := ValidateInputs(meta.Inputs, input); err != nil {
		return 0, types.WrapError(types.ErrInvalidArgument, "input validation failed", err)
	}

	workflowID, err := s.lookupWorkflowID(ctx, name)
	if err != nil {
		return 0, err
	}

	if s.runs == nil {
		return 0, types.Errorf(types.ErrUnavailable, "workflow run store not ready")
	}
	run, err := s.runs.CreateRun(ctx, workflowID, name, "db", triggerType, nil, map[string]any(input))
	if err != nil {
		return 0, fmt.Errorf("create workflow run: %w", err)
	}
	if run == nil {
		return 0, types.Errorf(types.ErrInternal, "create workflow run returned nil")
	}

	go s.executeRun(meta, run, triggerType, input)
	return run.ID, nil
}

func (s *Service) executeRun(meta *types.WorkflowMetadata, run *gen.WorkflowRun, triggerType string, input types.KV) {
	asyncCtx, asyncSpan := fbtrace.StartSpan(context.Background(), "workflow.run.async")
	defer asyncSpan.End()
	ctx, cancel := fbtrace.DetachWithTimeout(asyncCtx, 10*time.Minute)
	defer cancel()

	runner := NewRunnerWithStore(s.runs, s.auditor, s.metrics, "db", triggerType).
		WithDefinitionStore(s.catalog).
		WithWorkflowID(runWorkflowID(run)).
		WithExistingRun(run)
	defer func() {
		if cerr := runner.Close(); cerr != nil {
			flog.Error(fmt.Errorf("workflow %s: close runner: %w", meta.Name, cerr))
		}
	}()

	if err := runner.Execute(ctx, *meta, input, "db"); err != nil {
		flog.Error(fmt.Errorf("workflow %s run %d: %w", meta.Name, run.ID, err))
	}
}

func runWorkflowID(run *gen.WorkflowRun) int64 {
	if run == nil || run.WorkflowID == nil {
		return 0
	}
	return *run.WorkflowID
}

func (s *Service) lookupWorkflowID(ctx context.Context, name string) (int64, error) {
	defs, err := s.catalog.ListDefinitions(ctx)
	if err != nil {
		return 0, fmt.Errorf("list workflows: %w", err)
	}
	for _, d := range defs {
		if d != nil && d.Name == name {
			return d.ID, nil
		}
	}
	return 0, types.Errorf(types.ErrNotFound, "workflow %s", name)
}

// ApplyYAML parses YAML, upserts the definition, and reloads triggers.
func (s *Service) ApplyYAML(ctx context.Context, data []byte) (*gen.Workflow, error) {
	meta, err := ParseYAML(data)
	if err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "invalid workflow YAML", err)
	}
	row, err := s.catalog.ApplyDefinition(ctx, meta)
	if err != nil {
		return nil, err
	}
	if err := s.ReloadTriggers(ctx); err != nil {
		flog.Error(fmt.Errorf("workflow reload triggers after apply: %w", err))
	}
	return row, nil
}

// List returns workflow definition rows.
func (s *Service) List(ctx context.Context) ([]*gen.Workflow, error) {
	return s.catalog.ListDefinitions(ctx)
}

// Get returns workflow metadata by name.
func (s *Service) Get(ctx context.Context, name string) (*types.WorkflowMetadata, error) {
	return s.catalog.GetMetadata(ctx, name)
}

// Export returns YAML for a stored workflow.
func (s *Service) Export(ctx context.Context, name string) ([]byte, error) {
	meta, err := s.catalog.GetMetadata(ctx, name)
	if err != nil {
		return nil, err
	}
	return ExportYAML(meta)
}

// Delete removes a workflow definition and reloads triggers.
func (s *Service) Delete(ctx context.Context, name string) error {
	if err := s.catalog.DeleteDefinitionByName(ctx, name); err != nil {
		return err
	}
	if err := s.ReloadTriggers(ctx); err != nil {
		flog.Error(fmt.Errorf("workflow reload triggers after delete: %w", err))
	}
	return nil
}

// ListRuns returns recent runs for a workflow name.
func (s *Service) ListRuns(ctx context.Context, name string) ([]*gen.WorkflowRun, error) {
	return s.catalog.ListRunsByName(ctx, name)
}

// WebhookConfigs returns a copy of the current webhook path map.
func (s *Service) WebhookConfigs() map[string]*WebhookEndpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*WebhookEndpoint, len(s.webhooks))
	maps.Copy(out, s.webhooks)
	return out
}

// LookupWebhook returns the webhook endpoint for a path (without /webhook/workflow/ prefix).
func (s *Service) LookupWebhook(path string) (*WebhookEndpoint, bool) {
	path = strings.TrimPrefix(path, "/")
	s.mu.RLock()
	defer s.mu.RUnlock()
	ep, ok := s.webhooks[path]
	return ep, ok
}

// ReloadTriggers rebuilds cron jobs and webhook configs from enabled workflow definitions.
func (s *Service) ReloadTriggers(ctx context.Context) error {
	if s == nil || s.catalog == nil {
		return nil
	}
	defs, err := s.catalog.ListDefinitions(ctx)
	if err != nil {
		return fmt.Errorf("list workflows for trigger reload: %w", err)
	}

	webhooks, jobs := s.collectTriggers(ctx, defs)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		for _, entry := range s.cron.Entries() {
			s.cron.Remove(entry.ID())
		}
	} else {
		s.cron = cron.New(
			cron.WithSeconds(),
			cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor)),
		)
		s.cron.Start()
	}

	for _, job := range jobs {
		name := job.name
		spec := job.spec
		_, err := s.cron.AddFunc(spec, func(_ context.Context) error {
			s.runCronWorkflow(name)
			return nil
		})
		if err != nil {
			flog.Error(fmt.Errorf("workflow %s: register cron %q: %w", name, spec, err))
			continue
		}
		flog.Info("workflow %s: registered cron trigger %q", name, spec)
	}

	s.webhooks = webhooks
	flog.Info("workflow service reloaded triggers: %d cron, %d webhook", len(jobs), len(webhooks))
	return nil
}

type cronJobSpec struct {
	name string
	spec string
}

func (s *Service) collectTriggers(ctx context.Context, defs []*gen.Workflow) (map[string]*WebhookEndpoint, []cronJobSpec) {
	webhooks := make(map[string]*WebhookEndpoint)
	var jobs []cronJobSpec
	for _, row := range defs {
		if row == nil || !row.Enabled {
			continue
		}
		meta, err := s.catalog.GetMetadata(ctx, row.Name)
		if err != nil {
			flog.Error(fmt.Errorf("workflow %s: load metadata for triggers: %w", row.Name, err))
			continue
		}
		appendTriggerJobs(row.Name, meta.Triggers, webhooks, &jobs)
	}
	return webhooks, jobs
}

func appendTriggerJobs(name string, triggers []types.WorkflowTriggerDef, webhooks map[string]*WebhookEndpoint, jobs *[]cronJobSpec) {
	for _, tr := range triggers {
		if !tr.Enabled {
			continue
		}
		switch strings.ToLower(tr.Type) {
		case "cron":
			if job, ok := cronJobFromTrigger(name, tr); ok {
				*jobs = append(*jobs, job)
			}
		case "webhook":
			registerWebhookTrigger(name, tr, webhooks)
		}
	}
}

func cronJobFromTrigger(name string, tr types.WorkflowTriggerDef) (cronJobSpec, bool) {
	spec := stringFromRule(tr.Rule, "cron")
	if spec == "" {
		spec = stringFromRule(tr.Rule, "expression")
	}
	if spec == "" {
		flog.Warn("workflow %s: cron trigger missing rule.cron", name)
		return cronJobSpec{}, false
	}
	if err := validateCronExpr(spec); err != nil {
		flog.Error(fmt.Errorf("workflow %s: invalid cron %q: %w", name, spec, err))
		return cronJobSpec{}, false
	}
	return cronJobSpec{name: name, spec: spec}, true
}

func registerWebhookTrigger(name string, tr types.WorkflowTriggerDef, webhooks map[string]*WebhookEndpoint) {
	wcfg, err := webhookConfigFromRule(name, tr.Rule)
	if err != nil {
		flog.Error(fmt.Errorf("workflow %s: %w", name, err))
		return
	}
	path := strings.TrimPrefix(wcfg.Path, "/")
	if _, exists := webhooks[path]; exists {
		flog.Error(fmt.Errorf("workflow %s: duplicate webhook path %q", name, path))
		return
	}
	webhooks[path] = &WebhookEndpoint{WorkflowName: name, Config: wcfg}
}

func (s *Service) runCronWorkflow(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	runID, err := s.StartRunAsync(ctx, name, "cron", types.KV{})
	if err != nil {
		flog.Error(fmt.Errorf("workflow %s: cron start: %w", name, err))
		return
	}
	flog.Info("workflow %s: cron started run %d", name, runID)
}

// Stop shuts down the cron scheduler.
func (s *Service) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron == nil {
		return
	}
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(30 * time.Second):
		flog.Warn("workflow cron stop timed out after 30s")
	}
	s.cron = nil
}

func stringFromRule(rule types.KV, key string) string {
	if rule == nil {
		return ""
	}
	v, ok := rule[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func validateCronExpr(spec string) error {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := p.Parse(spec)
	return err
}

var allowedWebhookMethods = map[string]bool{
	"GET":  true,
	"POST": true,
	"PUT":  true,
}

// webhookConfigFromRule converts a workflow trigger rule into a pipeline.WebhookConfig
// (same auth/path/method/payload fields as pipeline webhooks).
func webhookConfigFromRule(workflowName string, rule types.KV) (*pipeline.WebhookConfig, error) {
	if rule == nil {
		return nil, fmt.Errorf("webhook trigger rule is empty")
	}
	path := stringFromRule(rule, "path")
	if path == "" {
		return nil, fmt.Errorf("webhook trigger path must not be empty")
	}
	method, err := webhookMethodFromRule(rule)
	if err != nil {
		return nil, err
	}
	auth, err := webhookAuthFromRule(rule)
	if err != nil {
		return nil, err
	}
	payloadMode, err := webhookPayloadFromRule(rule)
	if err != nil {
		return nil, err
	}
	eventType := stringFromRule(rule, "event_type")
	if eventType == "" {
		if strings.TrimSpace(workflowName) != "" {
			eventType = "workflow.webhook." + workflowName
		} else {
			eventType = "workflow.webhook." + strings.TrimPrefix(path, "/")
		}
	}
	return &pipeline.WebhookConfig{
		Path:      path,
		Method:    method,
		Auth:      auth,
		Payload:   payloadMode,
		EventType: eventType,
	}, nil
}

func webhookMethodFromRule(rule types.KV) (string, error) {
	method := strings.ToUpper(stringFromRule(rule, "method"))
	if method == "" {
		method = "POST"
	}
	if !allowedWebhookMethods[method] {
		return "", fmt.Errorf("unsupported webhook method %q", method)
	}
	return method, nil
}

func webhookAuthFromRule(rule types.KV) (pipeline.WebhookAuthConfig, error) {
	authMap, ok := rule["auth"].(map[string]any)
	if !ok || authMap == nil {
		if kv, ok := rule["auth"].(types.KV); ok {
			authMap = map[string]any(kv)
		}
	}
	token := stringFromMap(authMap, "token")
	hmacSecret := stringFromMap(authMap, "hmac_secret")
	if token == "" && hmacSecret == "" {
		return pipeline.WebhookAuthConfig{}, fmt.Errorf("webhook trigger requires at least one of auth.token or auth.hmac_secret")
	}
	tokenHeader := stringFromMap(authMap, "token_header")
	if tokenHeader == "" {
		tokenHeader = "X-Webhook-Token"
	}
	hmacHeader := stringFromMap(authMap, "hmac_header")
	if hmacHeader == "" {
		hmacHeader = "X-Hub-Signature-256"
	}
	return pipeline.WebhookAuthConfig{
		Token:       token,
		HMACSecret:  hmacSecret,
		TokenHeader: tokenHeader,
		HMACHeader:  hmacHeader,
	}, nil
}

func webhookPayloadFromRule(rule types.KV) (config.WebhookPayloadMode, error) {
	payloadMode := config.WebhookPayloadMode(stringFromRule(rule, "payload"))
	if payloadMode == "" {
		payloadMode = config.WebhookPayloadRaw
	}
	if payloadMode != config.WebhookPayloadRaw && payloadMode != config.WebhookPayloadMapped {
		return "", fmt.Errorf("invalid webhook payload mode %q", payloadMode)
	}
	return payloadMode, nil
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

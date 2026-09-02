package functions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/config"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/goccy/go-yaml"
)

// ApplyResult is returned after applying a function directory or bundle.
type ApplyResult struct {
	Name    string `json:"name"`
	ID      int64  `json:"id"`
	Version int    `json:"version"`
	Status  string `json:"status"`
}

// ListInfo is a published function summary for list APIs.
type ListInfo struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Status  string `json:"status"`
}

// ListAllInfo is a function summary for management list UIs (draft + published).
type ListAllInfo struct {
	Name                  string `json:"name"`
	Status                string `json:"status"`
	Version               int    `json:"version"`
	PublishedVersion      *int   `json:"published_version,omitempty"`
	HasUnpublishedChanges bool   `json:"has_unpublished_changes"`
}

// DraftView is a redacted draft snapshot for the management editor.
type DraftView struct {
	Name                  string            `json:"name"`
	Status                string            `json:"status"`
	Version               int               `json:"version"`
	Entrypoint            string            `json:"entrypoint"`
	Source                string            `json:"source"`
	Env                   map[string]string `json:"env,omitempty"`
	Token                 string            `json:"token"`
	TokenSet              bool              `json:"token_set"`
	HMACSecret            string            `json:"hmac_secret"`
	HMACSet               bool              `json:"hmac_set"`
	PublishedVersion      *int              `json:"published_version,omitempty"`
	HasUnpublishedChanges bool              `json:"has_unpublished_changes"`
}

// ExportBundle is a published function snapshot including secrets.
type ExportBundle struct {
	Metadata   string `json:"metadata"`
	Entrypoint string `json:"entrypoint"`
	Source     string `json:"source"`
	Version    int    `json:"version"`
	Name       string `json:"name"`
}

// InvokeRequest configures one function invocation.
type InvokeRequest struct {
	Name           string
	Version        *int
	Event          any
	IdempotencyKey string
	// RequireVersion rejects calls that omit Version (pipeline / capability path).
	RequireVersion bool
}

// InvokeResult is the outcome of a successful or replayed invoke.
type InvokeResult struct {
	Name     string `json:"name"`
	Version  int    `json:"version"`
	RunID    int64  `json:"run_id"`
	Status   string `json:"status"`
	Result   any    `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Replayed bool   `json:"replayed,omitempty"`
}

// Service orchestrates DB-backed function apply/invoke.
type Service struct {
	catalog       Catalog
	execProvider  ExecProvider
	checker       dcg.Checker
	maxConcurrent int
	globalSem     chan struct{}
	fnLocks       sync.Map // map[string]chan struct{}
}

// NewService creates a function Service with default concurrency.
func NewService(catalog Catalog, execProvider ExecProvider) *Service {
	return NewServiceWithLimits(catalog, execProvider, DefaultMaxConcurrency)
}

// NewServiceWithLimits creates a function Service with a custom global concurrency limit.
func NewServiceWithLimits(catalog Catalog, execProvider ExecProvider, maxConcurrent int) *Service {
	if maxConcurrent <= 0 {
		maxConcurrent = DefaultMaxConcurrency
	}
	return &Service{
		catalog:       catalog,
		execProvider:  execProvider,
		maxConcurrent: maxConcurrent,
		globalSem:     make(chan struct{}, maxConcurrent),
	}
}

// SetChecker overrides the dcg checker (nil uses dcg.DefaultChecker).
func (s *Service) SetChecker(c dcg.Checker) {
	if s == nil {
		return
	}
	s.checker = c
}

// Ready reports whether the service has a catalog and exec provider configured.
func (s *Service) Ready() bool {
	return s != nil && s.catalog != nil && s.execProvider != nil
}

// Create creates a draft-only function with a generated HTTP token and stub source.
func (s *Service) Create(ctx context.Context, name, entrypoint, createdBy string) (*DraftView, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "invalid function name", err)
	}
	entrypoint = filepath.Base(strings.TrimSpace(entrypoint))
	if !isAllowedEntrypoint(entrypoint) {
		return nil, types.Errorf(types.ErrInvalidArgument, "entrypoint must be main.py, main.sh, or main.go")
	}
	token, err := randomToken()
	if err != nil {
		return nil, types.WrapError(types.ErrInternal, "generate function token", err)
	}
	meta := &Metadata{
		Name: name,
		HTTP: HTTPConfig{Auth: HTTPAuth{Token: token}},
		Env:  map[string]string{},
	}
	metaYAML, err := marshalMetadataYAML(meta)
	if err != nil {
		return nil, err
	}
	source := stubSource(entrypoint)
	if err := s.catalog.Create(ctx, name, metaYAML, entrypoint, source, strings.TrimSpace(createdBy)); err != nil {
		return nil, err
	}
	return s.GetDraft(ctx, name)
}

// ListAll returns draft and published function summaries for management UIs.
func (s *Service) ListAll(ctx context.Context) ([]ListAllInfo, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	defs, err := s.catalog.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ListAllInfo, 0, len(defs))
	for _, def := range defs {
		if def == nil {
			continue
		}
		info := ListAllInfo{
			Name:                  def.Name,
			Status:                def.Status,
			Version:               def.Version,
			HasUnpublishedChanges: hasUnpublishedChanges(def),
		}
		if ver, verr := s.catalog.GetLatestPublished(ctx, def.Name); verr == nil && ver != nil {
			v := ver.Version
			info.PublishedVersion = &v
		} else if verr != nil && !errors.Is(verr, types.ErrNotFound) {
			return nil, verr
		}
		items = append(items, info)
	}
	return items, nil
}

// GetDraft returns a redacted draft for the management editor.
func (s *Service) GetDraft(ctx context.Context, name string) (*DraftView, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	def, err := s.catalog.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.draftViewFromDef(ctx, def)
}

// SaveDraft merges secrets, validates, and updates draft with optimistic locking.
func (s *Service) SaveDraft(ctx context.Context, name, metadata, entrypoint, source string, expectedVersion int) (*DraftView, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	def, err := s.catalog.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.Version != expectedVersion {
		return nil, types.ErrConflict
	}
	mergedMeta, err := mergeDraftMetadata(def.MetadataDraft, metadata)
	if err != nil {
		return nil, err
	}
	if mergedMeta.Name != name {
		return nil, types.Errorf(types.ErrInvalidArgument, "metadata name must match function name %q", name)
	}
	metaYAML, err := marshalMetadataYAML(mergedMeta)
	if err != nil {
		return nil, err
	}
	entrypoint = filepath.Base(strings.TrimSpace(entrypoint))
	if !isAllowedEntrypoint(entrypoint) {
		return nil, types.Errorf(types.ErrInvalidArgument, "entrypoint must be main.py, main.sh, or main.go")
	}
	if strings.TrimSpace(source) == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "source is required")
	}
	if len(source) > MaxSourceBytes {
		return nil, types.Errorf(types.ErrInvalidArgument, "source exceeds %d bytes", MaxSourceBytes)
	}
	updated, err := s.catalog.UpdateDraft(ctx, name, metaYAML, entrypoint, source, expectedVersion)
	if err != nil {
		return nil, err
	}
	return s.draftViewFromDef(ctx, updated)
}

// LatestPublishedVersion returns the version number of the latest published snapshot.
func (s *Service) LatestPublishedVersion(ctx context.Context, name string) (int, error) {
	ver, _, err := s.publishedVersion(ctx, name, nil)
	if err != nil {
		return 0, err
	}
	return ver.Version, nil
}

// Publish publishes the current draft with optimistic locking.
func (s *Service) Publish(ctx context.Context, name string, expectedVersion int) (*ApplyResult, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	def, err := s.catalog.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.Version != expectedVersion {
		return nil, types.ErrConflict
	}
	if _, err := ParseMetadataYAML(def.MetadataDraft); err != nil {
		return nil, err
	}
	entrypoint := filepath.Base(strings.TrimSpace(def.EntrypointDraft))
	if !isAllowedEntrypoint(entrypoint) {
		return nil, types.Errorf(types.ErrInvalidArgument, "entrypoint must be main.py, main.sh, or main.go")
	}
	if strings.TrimSpace(def.SourceDraft) == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "source is required")
	}
	published, err := s.catalog.Publish(ctx, name, expectedVersion)
	if err != nil {
		return nil, err
	}
	ver, err := s.catalog.GetLatestPublished(ctx, name)
	if err != nil {
		return nil, err
	}
	return &ApplyResult{
		Name:    published.Name,
		ID:      published.ID,
		Version: ver.Version,
		Status:  published.Status,
	}, nil
}

// ApplyDir loads a function directory, writes draft, and publishes immediately.
func (s *Service) ApplyDir(ctx context.Context, dir, createdBy string) (*ApplyResult, error) {
	metaYAML, entrypoint, source, err := LoadDir(dir)
	if err != nil {
		return nil, err
	}
	return s.ApplyBundle(ctx, metaYAML, entrypoint, source, createdBy)
}

// ApplyBundle validates metadata + entrypoint source, writes draft, and publishes.
func (s *Service) ApplyBundle(ctx context.Context, metadata, entrypoint, source, createdBy string) (*ApplyResult, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	meta, err := ParseMetadataYAML(metadata)
	if err != nil {
		return nil, err
	}
	entrypoint = filepath.Base(strings.TrimSpace(entrypoint))
	if !isAllowedEntrypoint(entrypoint) {
		return nil, types.Errorf(types.ErrInvalidArgument, "entrypoint must be main.py, main.sh, or main.go")
	}
	if strings.TrimSpace(source) == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "source is required")
	}
	if len(source) > MaxSourceBytes {
		return nil, types.Errorf(types.ErrInvalidArgument, "source exceeds %d bytes", MaxSourceBytes)
	}
	name := meta.Name

	def, err := s.catalog.GetByName(ctx, name)
	if err != nil {
		if !errors.Is(err, types.ErrNotFound) {
			return nil, err
		}
		if createErr := s.catalog.Create(ctx, name, metadata, entrypoint, source, strings.TrimSpace(createdBy)); createErr != nil {
			return nil, createErr
		}
		def, err = s.catalog.GetByName(ctx, name)
		if err != nil {
			return nil, err
		}
	} else {
		updated, updErr := s.catalog.UpdateDraft(ctx, name, metadata, entrypoint, source, def.Version)
		if updErr != nil {
			return nil, updErr
		}
		def = updated
	}

	if _, err := s.catalog.Publish(ctx, name, def.Version); err != nil {
		return nil, err
	}
	ver, err := s.catalog.GetLatestPublished(ctx, name)
	if err != nil {
		return nil, err
	}
	def, err = s.catalog.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &ApplyResult{
		Name:    def.Name,
		ID:      def.ID,
		Version: ver.Version,
		Status:  def.Status,
	}, nil
}

// List returns published function summaries.
func (s *Service) List(ctx context.Context) ([]ListInfo, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	defs, err := s.catalog.ListPublished(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ListInfo, 0, len(defs))
	for _, def := range defs {
		if def == nil {
			continue
		}
		items = append(items, ListInfo{
			Name:    def.Name,
			Version: def.Version,
			Status:  def.Status,
		})
	}
	return items, nil
}

// Get returns published function metadata (including secrets) for management APIs.
func (s *Service) Get(ctx context.Context, name string) (map[string]any, error) {
	ver, def, err := s.publishedVersion(ctx, name, nil)
	if err != nil {
		return nil, err
	}
	meta, err := ParseMetadataYAML(ver.Metadata)
	if err != nil {
		return nil, types.WrapError(types.ErrInternal, "parse published metadata", err)
	}
	return map[string]any{
		"id":         def.ID,
		"name":       meta.Name,
		"version":    ver.Version,
		"status":     def.Status,
		"entrypoint": ver.Entrypoint,
		"env":        meta.Env,
	}, nil
}

// GetPublic returns published function metadata without auth secrets or source.
func (s *Service) GetPublic(ctx context.Context, name string, version *int) (map[string]any, error) {
	ver, _, err := s.publishedVersion(ctx, name, version)
	if err != nil {
		return nil, err
	}
	meta, err := ParseMetadataYAML(ver.Metadata)
	if err != nil {
		return nil, types.WrapError(types.ErrInternal, "parse published metadata", err)
	}
	return map[string]any{
		"name":       meta.Name,
		"version":    ver.Version,
		"entrypoint": ver.Entrypoint,
		"runtime":    runtimeFromEntrypoint(ver.Entrypoint),
		"env":        meta.Env,
	}, nil
}

// Export returns published metadata+entrypoint+source including secrets.
func (s *Service) Export(ctx context.Context, name string) (*ExportBundle, error) {
	ver, _, err := s.publishedVersion(ctx, name, nil)
	if err != nil {
		return nil, err
	}
	return &ExportBundle{
		Name:       ver.FunctionName,
		Version:    ver.Version,
		Metadata:   ver.Metadata,
		Entrypoint: ver.Entrypoint,
		Source:     ver.Source,
	}, nil
}

// ExportVersion returns a specific published version snapshot including secrets.
func (s *Service) ExportVersion(ctx context.Context, name string, version int) (*ExportBundle, error) {
	ver, _, err := s.publishedVersion(ctx, name, &version)
	if err != nil {
		return nil, err
	}
	return &ExportBundle{
		Name:       ver.FunctionName,
		Version:    ver.Version,
		Metadata:   ver.Metadata,
		Entrypoint: ver.Entrypoint,
		Source:     ver.Source,
	}, nil
}

// Delete removes a function definition.
func (s *Service) Delete(ctx context.Context, name string) error {
	if s == nil || s.catalog == nil {
		return types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	if _, err := s.catalog.GetByName(ctx, name); err != nil {
		return err
	}
	_, err := s.catalog.Delete(ctx, name)
	return err
}

// ListRuns returns recent runs for a function name.
func (s *Service) ListRuns(ctx context.Context, name string) ([]*model.FunctionRun, error) {
	if s == nil || s.catalog == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	return s.catalog.ListRuns(ctx, name)
}

// Invoke runs a published function version with concurrency and idempotency controls.
func (s *Service) Invoke(ctx context.Context, req InvokeRequest) (*InvokeResult, error) {
	name, err := s.validateInvokeRequest(req)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	if key != "" {
		if existing, getErr := s.catalog.GetRunByIdempotency(ctx, name, key); getErr == nil {
			return replayRun(existing)
		} else if !errors.Is(getErr, types.ErrNotFound) {
			return nil, getErr
		}
	}

	release, err := s.acquireInvokeSlots(name)
	if err != nil {
		if key != "" && errors.Is(err, types.ErrRateLimited) {
			if existing, getErr := s.catalog.GetRunByIdempotency(ctx, name, key); getErr == nil {
				return replayRun(existing)
			} else if getErr != nil && !errors.Is(getErr, types.ErrNotFound) {
				return nil, getErr
			}
		}
		return nil, err
	}
	defer release()

	ver, _, err := s.publishedVersion(ctx, name, req.Version)
	if err != nil {
		return nil, err
	}

	run, replayed, err := s.beginRun(ctx, name, ver.Version, key)
	if replayed != nil {
		return replayed, err
	}
	if err != nil {
		return nil, err
	}
	return s.finishInvoke(ctx, name, ver, run, req.Event)
}

// PublishedMetadata loads published function metadata for HTTP call authentication.
func (s *Service) PublishedMetadata(ctx context.Context, name string, version *int) (*Metadata, error) {
	ver, _, err := s.publishedVersion(ctx, name, version)
	if err != nil {
		return nil, err
	}
	meta, err := ParseMetadataYAML(ver.Metadata)
	if err != nil {
		return nil, types.WrapError(types.ErrInternal, "parse published metadata", err)
	}
	return meta, nil
}

func (s *Service) validateInvokeRequest(req InvokeRequest) (string, error) {
	if s == nil || s.catalog == nil {
		return "", types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	if s.execProvider == nil {
		return "", types.Errorf(types.ErrUnavailable, "function exec provider not ready")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	if req.RequireVersion && req.Version == nil {
		return "", types.Errorf(types.ErrInvalidArgument, "version is required")
	}
	return name, nil
}

func (s *Service) acquireInvokeSlots(name string) (func(), error) {
	if !s.tryAcquireGlobal() {
		return nil, types.Errorf(types.ErrRateLimited, "function invoke concurrency saturated")
	}
	fnLock := s.functionLock(name)
	if !tryAcquireSlot(fnLock) {
		s.releaseGlobal()
		return nil, types.Errorf(types.ErrRateLimited, "function %s is already running", name)
	}
	return func() {
		releaseSlot(fnLock)
		s.releaseGlobal()
	}, nil
}

func (s *Service) beginRun(ctx context.Context, name string, version int, idempotencyKey string) (*model.FunctionRun, *InvokeResult, error) {
	var idemKey *string
	key := strings.TrimSpace(idempotencyKey)
	if key != "" {
		idemKey = &key
	}
	run, err := s.catalog.CreateRun(ctx, name, version, idemKey)
	if err != nil {
		if key != "" && (errors.Is(err, types.ErrAlreadyExists) || errors.Is(err, types.ErrConflict)) {
			existing, getErr := s.catalog.GetRunByIdempotency(ctx, name, key)
			if getErr != nil {
				return nil, nil, getErr
			}
			replayed, replayErr := replayRun(existing)
			return nil, replayed, replayErr
		}
		return nil, nil, err
	}
	return run, nil, nil
}

func (s *Service) finishInvoke(ctx context.Context, name string, ver *model.FunctionDefinitionVersion, run *model.FunctionRun, event any) (*InvokeResult, error) {
	started := time.Now()
	result, invokeErr := s.executeRun(ctx, ver, event)
	durationMs := time.Since(started).Milliseconds()
	if invokeErr != nil {
		errMsg, exitCode := runFailureDetails(invokeErr)
		if _, cerr := s.catalog.CompleteRun(ctx, run.ID, string(types.FunctionRunFailed), durationMs, exitCode, errMsg, nil); cerr != nil {
			return nil, types.WrapError(types.ErrInternal, "complete failed run", errors.Join(invokeErr, cerr))
		}
		return nil, invokeErr
	}
	resultBytes, err := sonic.Marshal(result)
	if err != nil {
		errMsg := "marshal result JSON"
		if _, cerr := s.catalog.CompleteRun(ctx, run.ID, string(types.FunctionRunFailed), durationMs, nil, errMsg, nil); cerr != nil {
			return nil, types.WrapError(types.ErrInternal, "complete failed run after marshal error", errors.Join(err, cerr))
		}
		return nil, types.WrapError(types.ErrInternal, errMsg, err)
	}
	if len(resultBytes) > MaxJSONBytes {
		errMsg := fmt.Sprintf("result exceeds %d bytes", MaxJSONBytes)
		if _, cerr := s.catalog.CompleteRun(ctx, run.ID, string(types.FunctionRunFailed), durationMs, nil, errMsg, nil); cerr != nil {
			return nil, types.WrapError(types.ErrInternal, "complete failed run after oversized result", cerr)
		}
		return nil, types.Errorf(types.ErrInvalidArgument, "%s", errMsg)
	}
	resultStr := string(resultBytes)
	exitZero := 0
	completed, err := s.catalog.CompleteRun(ctx, run.ID, string(types.FunctionRunSucceeded), durationMs, &exitZero, "", &resultStr)
	if err != nil {
		return nil, err
	}
	return &InvokeResult{
		Name:     name,
		Version:  ver.Version,
		RunID:    completed.ID,
		Status:   completed.Status,
		Result:   result,
		ExitCode: completed.ExitCode,
	}, nil
}

func runFailureDetails(err error) (string, *int) {
	errMsg := err.Error()
	var exitCode *int
	var fre *functionRunError
	if errors.As(err, &fre) {
		exitCode = fre.exitCode
		errMsg = fre.msg
	}
	return errMsg, exitCode
}

type functionRunError struct {
	msg      string
	exitCode *int
	kind     error
}

func (e *functionRunError) Error() string {
	if e == nil {
		return "function run error"
	}
	return e.msg
}

func (e *functionRunError) Is(target error) bool {
	if e == nil {
		return false
	}
	return errors.Is(e.kind, target)
}

func (s *Service) executeRun(ctx context.Context, ver *model.FunctionDefinitionVersion, event any) (any, error) {
	if err := s.guardSource(ctx, ver); err != nil {
		return nil, err
	}
	stdin, err := buildInvokeStdin(ver, event)
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "flowbot-fn-*")
	if err != nil {
		return nil, types.WrapError(types.ErrInternal, "create temp workspace", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	cfg, err := s.execConfigForWorkspace(ctx, tmp)
	if err != nil {
		return nil, err
	}
	res, err := pkgexec.RunEntrypoint(ctx, cfg, ver.Entrypoint, ver.Source, stdin, nil)
	if err != nil {
		return nil, classifyEntrypointError(err)
	}
	return parseEntrypointResult(res)
}

func classifyEntrypointError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if sandboxDaemonUnavailable(msg) {
		return types.Errorf(types.ErrUnavailable, "function sandbox is unavailable (Docker is not running)")
	}
	return types.WrapError(types.ErrInternal, "run entrypoint", err)
}

func sandboxDaemonUnavailable(msg string) bool {
	return strings.Contains(msg, "docker API") ||
		strings.Contains(msg, "docker.sock") ||
		strings.Contains(msg, "Docker daemon") ||
		strings.Contains(msg, "pipe/docker_engine")
}

func (s *Service) guardSource(ctx context.Context, ver *model.FunctionDefinitionVersion) error {
	language, err := languageFromEntrypoint(ver.Entrypoint)
	if err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid entrypoint", err)
	}
	synth, err := dcg.SynthCommand(language, ver.Source)
	if err != nil {
		return types.WrapError(types.ErrInvalidArgument, "dcg synthesize", err)
	}
	checker := s.checker
	if checker == nil {
		checker = dcg.DefaultChecker()
	}
	decision, err := checker.Check(ctx, synth)
	if err != nil {
		return types.Errorf(types.ErrForbidden, "dcg check failed: %v", err)
	}
	if !decision.Allow {
		reason := decision.Reason
		if reason == "" {
			reason = dcg.ReasonBlocked
		}
		return types.Errorf(types.ErrForbidden, "%s", reason)
	}
	return nil
}

func buildInvokeStdin(ver *model.FunctionDefinitionVersion, event any) ([]byte, error) {
	meta, err := ParseMetadataYAML(ver.Metadata)
	if err != nil {
		return nil, err
	}
	envMap := meta.Env
	if envMap == nil {
		envMap = map[string]string{}
	}
	stdin, err := sonic.Marshal(map[string]any{
		"name":    ver.FunctionName,
		"version": ver.Version,
		"event":   event,
		"env":     envMap,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "marshal stdin envelope", err)
	}
	if len(stdin) > MaxJSONBytes {
		return nil, types.Errorf(types.ErrInvalidArgument, "stdin exceeds %d bytes", MaxJSONBytes)
	}
	return stdin, nil
}

func (s *Service) execConfigForWorkspace(ctx context.Context, workspace string) (pkgexec.Config, error) {
	cfg, err := s.execProvider.ExecConfig(ctx)
	if err != nil {
		return pkgexec.Config{}, err
	}
	cfg.Workspace = workspace
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.MaxOutput <= 0 || cfg.MaxOutput < MaxJSONBytes {
		cfg.MaxOutput = MaxJSONBytes
	}
	return cfg, nil
}

func parseEntrypointResult(res pkgexec.Result) (any, error) {
	if res.ExitCode != 0 {
		code := res.ExitCode
		return nil, &functionRunError{
			msg:      fmt.Sprintf("function exited with code %d", res.ExitCode),
			exitCode: &code,
			kind:     types.ErrInvalidArgument,
		}
	}
	parsed, err := ParseStdoutJSON(res.Stdout)
	if err != nil {
		code := res.ExitCode
		return nil, &functionRunError{
			msg:      err.Error(),
			exitCode: &code,
			kind:     types.ErrInvalidArgument,
		}
	}
	return parsed, nil
}

func replayRun(run *model.FunctionRun) (*InvokeResult, error) {
	if run == nil {
		return nil, types.Errorf(types.ErrNotFound, "idempotent run not found")
	}
	switch types.FunctionRunState(run.Status) {
	case types.FunctionRunRunning:
		return nil, types.Errorf(types.ErrConflict, "function run with this idempotency key is still running")
	case types.FunctionRunSucceeded, types.FunctionRunFailed:
		out := &InvokeResult{
			Name:     run.FunctionName,
			Version:  run.Version,
			RunID:    run.ID,
			Status:   run.Status,
			Error:    run.Error,
			ExitCode: run.ExitCode,
			Replayed: true,
		}
		if run.ResultJSON != nil && strings.TrimSpace(*run.ResultJSON) != "" {
			var v any
			if err := sonic.Unmarshal([]byte(*run.ResultJSON), &v); err != nil {
				return nil, types.WrapError(types.ErrInternal, "parse stored result JSON", err)
			}
			out.Result = v
		}
		if types.FunctionRunState(run.Status) == types.FunctionRunFailed {
			msg := run.Error
			if msg == "" {
				msg = "function run failed"
			}
			return out, types.Errorf(types.ErrInvalidArgument, "%s", msg)
		}
		return out, nil
	default:
		return nil, types.Errorf(types.ErrConflict, "unexpected run status %q", run.Status)
	}
}

func (s *Service) publishedVersion(ctx context.Context, name string, version *int) (*model.FunctionDefinitionVersion, *model.FunctionDefinition, error) {
	if s == nil || s.catalog == nil {
		return nil, nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil, types.Errorf(types.ErrInvalidArgument, "function name is required")
	}
	def, err := s.catalog.GetByName(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	var ver *model.FunctionDefinitionVersion
	if version != nil {
		ver, err = s.catalog.GetVersion(ctx, name, *version)
	} else {
		ver, err = s.catalog.GetLatestPublished(ctx, name)
	}
	if err != nil {
		return nil, nil, err
	}
	return ver, def, nil
}

func (s *Service) tryAcquireGlobal() bool {
	select {
	case s.globalSem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Service) releaseGlobal() {
	select {
	case <-s.globalSem:
	default:
	}
}

func (s *Service) functionLock(name string) chan struct{} {
	ch := make(chan struct{}, 1)
	actual, _ := s.fnLocks.LoadOrStore(name, ch)
	typed, ok := actual.(chan struct{})
	if !ok {
		return ch
	}
	return typed
}

func tryAcquireSlot(ch chan struct{}) bool {
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func releaseSlot(ch chan struct{}) {
	select {
	case <-ch:
	default:
	}
}

func (s *Service) draftViewFromDef(ctx context.Context, def *model.FunctionDefinition) (*DraftView, error) {
	if def == nil {
		return nil, types.ErrNotFound
	}
	meta, err := ParseMetadataYAML(def.MetadataDraft)
	if err != nil {
		return nil, err
	}
	view := &DraftView{
		Name:                  def.Name,
		Status:                def.Status,
		Version:               def.Version,
		Entrypoint:            def.EntrypointDraft,
		Source:                def.SourceDraft,
		Env:                   meta.Env,
		TokenSet:              meta.HTTP.Auth.Token != "",
		HMACSet:               meta.HTTP.Auth.HMACSecret != "",
		HasUnpublishedChanges: hasUnpublishedChanges(def),
	}
	if view.TokenSet {
		view.Token = config.MaskedSecret
	}
	if view.HMACSet {
		view.HMACSecret = config.MaskedSecret
	}
	if ver, verr := s.catalog.GetLatestPublished(ctx, def.Name); verr == nil && ver != nil {
		v := ver.Version
		view.PublishedVersion = &v
	} else if verr != nil && !errors.Is(verr, types.ErrNotFound) {
		return nil, verr
	}
	return view, nil
}

func hasUnpublishedChanges(def *model.FunctionDefinition) bool {
	if def == nil || def.MetadataPublished == nil || def.EntrypointPublished == nil || def.SourcePublished == nil {
		return false
	}
	return def.MetadataDraft != *def.MetadataPublished ||
		def.EntrypointDraft != *def.EntrypointPublished ||
		def.SourceDraft != *def.SourcePublished
}

func mergeDraftMetadata(existingYAML, incomingYAML string) (*Metadata, error) {
	existing, err := ParseMetadataYAML(existingYAML)
	if err != nil {
		return nil, types.WrapError(types.ErrInternal, "parse existing draft metadata", err)
	}
	incoming, err := parseMetadataYAMLLoose(incomingYAML)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(incoming.Name)
	if name == "" {
		name = existing.Name
	}
	incoming.Name = name
	incoming.HTTP.Auth.Token = mergeSecret(existing.HTTP.Auth.Token, incoming.HTTP.Auth.Token)
	incoming.HTTP.Auth.HMACSecret = mergeSecret(existing.HTTP.Auth.HMACSecret, incoming.HTTP.Auth.HMACSecret)
	if err := ValidateMetadata(incoming); err != nil {
		return nil, err
	}
	return incoming, nil
}

func parseMetadataYAMLLoose(data string) (*Metadata, error) {
	var meta Metadata
	if err := yaml.Unmarshal([]byte(data), &meta); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "invalid metadata YAML", err)
	}
	return &meta, nil
}

func mergeSecret(existing, incoming string) string {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" || incoming == config.MaskedSecret {
		return existing
	}
	return incoming
}

func marshalMetadataYAML(meta *Metadata) (string, error) {
	if meta == nil {
		return "", types.Errorf(types.ErrInvalidArgument, "metadata is required")
	}
	if err := ValidateMetadata(meta); err != nil {
		return "", err
	}
	b, err := yaml.Marshal(meta)
	if err != nil {
		return "", types.WrapError(types.ErrInternal, "marshal metadata YAML", err)
	}
	return string(b), nil
}

func randomToken() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func stubSource(entrypoint string) string {
	switch filepath.Base(entrypoint) {
	case "main.sh":
		return "echo '{}'\n"
	case "main.go":
		return "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"{}\")\n}\n"
	default:
		return "print(\"{}\")\n"
	}
}

// Command flowbot-gateway is the local CLI sidecar that claims CapGateway jobs.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/flowline-io/flowbot/cmd/gateway/client"
	"github.com/flowline-io/flowbot/cmd/gateway/config"
	"github.com/flowline-io/flowbot/cmd/gateway/runner"
	"github.com/flowline-io/flowbot/cmd/gateway/worker"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

func main() {
	cfgPath := flag.String("config", "gateway.yaml", "path to gateway.yaml")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn, error")
	flag.Parse()

	flog.Init(flog.Config{Level: *logLevel})

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		flog.Error(fmt.Errorf("load config %s: %w", *cfgPath, err))
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		flog.Error(fmt.Errorf("invalid config: %w", err))
		os.Exit(1)
	}

	if path, lookErr := exec.LookPath(cfg.CursorBinary); lookErr != nil {
		flog.Warn("cursor binary not found in PATH; jobs will fail until cursor_binary is fixed (cursor_binary=%s): %v",
			cfg.CursorBinary, lookErr)
	} else {
		flog.Info("cursor binary resolved cursor_binary=%s path=%s", cfg.CursorBinary, path)
	}

	api := client.New(cfg.FlowbotURL, cfg.AccessToken)
	cursorRunner := runner.NewCursor(cfg.CursorBinary, cfg.CursorAPIKey, cfg.JobTimeout).
		WithFlowbotAgent(cfg.FlowbotURL, cfg.AgentAccessToken)
	runners := map[types.GatewayCLI]runner.Runner{
		types.GatewayCLICursor:   cursorRunner,
		types.GatewayCLIOpenCode: runner.Unsupported{Name: "opencode"},
	}
	w := worker.New(api, cfg, runners)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Listen != "" {
		go serveHealthz(ctx, cfg.Listen)
	}

	flog.Info("flowbot-gateway starting worker_id=%s flowbot_url=%s default_workspace=%s max_concurrent=%d claim_interval=%s heartbeat_interval=%s job_timeout=%s cursor_binary=%s",
		cfg.WorkerID, cfg.FlowbotURL, cfg.DefaultWorkspace, cfg.MaxConcurrent,
		cfg.ClaimInterval, cfg.HeartbeatInterval, cfg.JobTimeout, cfg.CursorBinary)
	if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		flog.Error(fmt.Errorf("worker stopped: %w", err))
		os.Exit(1)
	}
	flog.Info("flowbot-gateway stopped")
}

func serveHealthz(ctx context.Context, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			flog.Warn("healthz write failed: %v", err)
		}
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			flog.Warn("healthz shutdown: %v", err)
		}
	}()
	flog.Info("healthz listening addr=%s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		flog.Error(fmt.Errorf("healthz server failed: %w", err))
	}
}

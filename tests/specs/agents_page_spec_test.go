//go:build integration
// +build integration

package specs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsession"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsessionentry"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type agentsWebAdapter struct {
	store.Adapter
	ent *gen.Client
	uid string
}

func (a *agentsWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *agentsWebAdapter) Close() error                     { return nil }
func (a *agentsWebAdapter) IsOpen() bool                     { return true }
func (a *agentsWebAdapter) GetName() string                  { return "bdd-agents" }
func (a *agentsWebAdapter) Stats() any                       { return nil }
func (a *agentsWebAdapter) GetDB() any                       { return a.ent }

func (a *agentsWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": []string{"admin:*"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *agentsWebAdapter) ListChatSessions(ctx context.Context, opts store.ListChatSessionsOptions) ([]*gen.ChatSession, string, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	order := []chatsession.OrderOption{
		gen.Desc(chatsession.FieldUpdatedAt),
		gen.Desc(chatsession.FieldID),
	}
	if opts.PinnedFirst {
		order = []chatsession.OrderOption{
			gen.Desc(chatsession.FieldPinned),
			gen.Desc(chatsession.FieldUpdatedAt),
			gen.Desc(chatsession.FieldID),
		}
	}
	q := a.ent.ChatSession.Query().
		Order(order...).
		Limit(opts.Limit + 1)
	if opts.Cursor != "" {
		if id, err := strconv.ParseInt(opts.Cursor, 10, 64); err == nil {
			q = q.Where(chatsession.IDLT(id))
		}
	}
	if opts.UID != "" {
		q = q.Where(chatsession.UIDEQ(opts.UID))
	}
	if opts.State != nil {
		q = q.Where(chatsession.StateEQ(*opts.State))
	}
	if opts.Archived != nil {
		q = q.Where(chatsession.ArchivedEQ(*opts.Archived))
	}
	if len(opts.Flags) > 0 {
		q = q.Where(chatsession.FlagIn(opts.Flags...))
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, "", err
	}
	var nextCursor string
	if len(rows) > opts.Limit {
		nextCursor = strconv.FormatInt(rows[opts.Limit-1].ID, 10)
		rows = rows[:opts.Limit]
	}
	return rows, nextCursor, nil
}

func (a *agentsWebAdapter) GetChatSession(ctx context.Context, flag string) (*gen.ChatSession, error) {
	row, err := a.ent.ChatSession.Query().Where(chatsession.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return row, nil
}

func (a *agentsWebAdapter) CreateChatSession(ctx context.Context, session *gen.ChatSession) error {
	_, err := a.ent.ChatSession.Create().
		SetFlag(session.Flag).
		SetUID(session.UID).
		SetLeafID(session.LeafID).
		SetState(session.State).
		SetMode(session.Mode).
		SetTitle(session.Title).
		SetCreatedAt(session.CreatedAt).
		SetUpdatedAt(session.UpdatedAt).
		Save(ctx)
	return err
}

func (a *agentsWebAdapter) UpdateChatSessionPreview(ctx context.Context, flag, preview string) error {
	n, err := a.ent.ChatSession.Update().Where(chatsession.FlagEQ(flag)).SetPreview(preview).Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *agentsWebAdapter) UpdateChatSessionPinned(ctx context.Context, flag string, pinned bool) error {
	n, err := a.ent.ChatSession.Update().Where(chatsession.FlagEQ(flag)).SetPinned(pinned).Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *agentsWebAdapter) UpdateChatSessionArchived(ctx context.Context, flag string, archived bool) error {
	n, err := a.ent.ChatSession.Update().Where(chatsession.FlagEQ(flag)).SetArchived(archived).Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func (a *agentsWebAdapter) ListChatSessionEntries(ctx context.Context, sessionID string) ([]*gen.ChatSessionEntry, error) {
	return a.ent.ChatSessionEntry.Query().
		Where(chatsessionentry.SessionIDEQ(sessionID)).
		Order(gen.Asc(chatsessionentry.FieldCreatedAt)).
		All(ctx)
}

func (a *agentsWebAdapter) ListChatSessionEntriesBySessions(ctx context.Context, sessionIDs []string) ([]*gen.ChatSessionEntry, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	return a.ent.ChatSessionEntry.Query().
		Where(chatsessionentry.SessionIDIn(sessionIDs...)).
		Order(gen.Asc(chatsessionentry.FieldCreatedAt)).
		All(ctx)
}

var _ = Describe("Agents UI", Label("module", "web"), func() {
	var (
		origDB          store.Adapter
		adapter         *agentsWebAdapter
		sessionID       string
		origModel       string
		origWorkspace   string
		workspaceDir    string
	)

	BeforeEach(func() {
		origDB = store.Database
		origModel = pkgconfig.App.ChatAgent.ChatModel
		origWorkspace = pkgconfig.App.ChatAgent.Workspace
		pkgconfig.App.ChatAgent.ChatModel = "bdd-test-model"
		sessionID = "bdd-agent-" + types.Id()

		var err error
		workspaceDir, err = os.MkdirTemp("", "agents-page-bdd-*")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(workspaceDir, "AGENTS.md"), []byte("# rules"), 0o644)).To(Succeed())
		pkgconfig.App.ChatAgent.Workspace = workspaceDir

		adapter = &agentsWebAdapter{
			Adapter: origDB,
			ent:     EntClient,
			uid:     "bdd-agents-" + types.Id(),
		}
		store.Database = adapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)

		ctx := context.Background()
		EntClient.ChatSession.Create().
			SetFlag(sessionID).
			SetUID(adapter.uid).
			SetTitle("BDD Agent Task").
			SetState(1).
			SetCreatedAt(time.Now().Add(-time.Hour)).
			SetUpdatedAt(time.Now()).
			SaveX(ctx)
	})

	AfterEach(func() {
		ctx := context.Background()
		EntClient.ChatSession.Delete().Where(chatsession.FlagEQ(sessionID)).ExecX(ctx)
		store.Database = origDB
		pkgconfig.App.ChatAgent.ChatModel = origModel
		pkgconfig.App.ChatAgent.Workspace = origWorkspace
		if workspaceDir != "" {
			_ = os.RemoveAll(workspaceDir)
			workspaceDir = ""
		}
	})

	Describe("GET /service/web/agents", func() {
		It("renders the agents page with composer and sessions", func() {
			req := MakeRequest(http.MethodGet, "/service/web/agents", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agents-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring("Agents"))
			Expect(string(body)).To(ContainSubstring("Your sessions"))
			Expect(string(body)).To(ContainSubstring("BDD Agent Task"))
			Expect(string(body)).To(ContainSubstring("chatagent-composer"))
		})

		It("redirects unauthenticated users to login", func() {
			req := MakeRequest(http.MethodGet, "/service/web/agents", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
		})
	})

	Describe("POST /service/web/agents", func() {
		It("creates a session and chat page is reachable", func() {
			req := MakeRequest(http.MethodPost, "/service/web/agents", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agents-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			var payload map[string]string
			Expect(json.Unmarshal(ReadBody(resp), &payload)).To(Succeed())
			newID := payload["session_id"]
			Expect(newID).NotTo(BeEmpty())

			detailReq := MakeRequest(http.MethodGet, fmt.Sprintf("/service/web/agents/%s", newID), nil)
			detailReq.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agents-token"})
			webmod.AttachCSRFForTest(detailReq)
			detailResp, err := App.Test(detailReq)
			Expect(err).NotTo(HaveOccurred())
			defer detailResp.Body.Close()

			Expect(detailResp.StatusCode).To(Equal(http.StatusOK))
			detailBody := ReadBody(detailResp)
			Expect(string(detailBody)).To(ContainSubstring("chatagent-thread"))
			Expect(string(detailBody)).To(ContainSubstring(newID))
			Expect(string(detailBody)).To(ContainSubstring("chatagent-context-ring"))
			Expect(string(detailBody)).To(ContainSubstring("Show context usage"))
			Expect(string(detailBody)).To(ContainSubstring("data-context-url=\"/service/web/agents/" + newID + "/context\""))
		})
	})

	Describe("GET /service/web/agents/:id/context", func() {
		It("returns context usage breakdown json", func() {
			req := MakeRequest(http.MethodGet, fmt.Sprintf("/service/web/agents/%s/context", sessionID), nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agents-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var report map[string]any
			Expect(json.Unmarshal(ReadBody(resp), &report)).To(Succeed())
			Expect(report).To(HaveKey("categories"))
			Expect(report).To(HaveKey("context_window"))
			Expect(report).To(HaveKey("model"))
		})
	})
})

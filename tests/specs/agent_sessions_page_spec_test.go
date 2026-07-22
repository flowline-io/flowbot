//go:build integration
// +build integration

package specs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsession"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsessionentry"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type agentSessionsWebAdapter struct {
	store.Adapter
	ent *gen.Client
	uid string
}

func (a *agentSessionsWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *agentSessionsWebAdapter) Close() error                     { return nil }
func (a *agentSessionsWebAdapter) IsOpen() bool                     { return true }
func (a *agentSessionsWebAdapter) GetName() string                  { return "bdd-agent-sessions" }
func (a *agentSessionsWebAdapter) Stats() any                       { return nil }
func (a *agentSessionsWebAdapter) GetDB() any                       { return a.ent }

func (a *agentSessionsWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
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

func (a *agentSessionsWebAdapter) ListChatSessions(ctx context.Context, opts store.ListChatSessionsOptions) ([]*gen.ChatSession, string, error) {
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

func (a *agentSessionsWebAdapter) GetChatSession(ctx context.Context, flag string) (*gen.ChatSession, error) {
	row, err := a.ent.ChatSession.Query().Where(chatsession.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return row, nil
}

func (a *agentSessionsWebAdapter) ListChatSessionEntries(ctx context.Context, sessionID string) ([]*gen.ChatSessionEntry, error) {
	return a.ent.ChatSessionEntry.Query().
		Where(chatsessionentry.SessionIDEQ(sessionID)).
		Order(gen.Asc(chatsessionentry.FieldCreatedAt)).
		All(ctx)
}

func (a *agentSessionsWebAdapter) ListChatSessionEntriesBySessions(ctx context.Context, sessionIDs []string) ([]*gen.ChatSessionEntry, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	return a.ent.ChatSessionEntry.Query().
		Where(chatsessionentry.SessionIDIn(sessionIDs...)).
		Order(gen.Asc(chatsessionentry.FieldCreatedAt)).
		All(ctx)
}

func (a *agentSessionsWebAdapter) ListAgentPlansBySession(_ context.Context, _ string) ([]*gen.AgentPlan, error) {
	return nil, nil
}

func (a *agentSessionsWebAdapter) ListAgentTodosBySession(_ context.Context, _ string) ([]*gen.AgentTodo, error) {
	return nil, nil
}

func (a *agentSessionsWebAdapter) ListAgentTodosBySessions(_ context.Context, _ []string) ([]*gen.AgentTodo, error) {
	return nil, nil
}

func (a *agentSessionsWebAdapter) ReplaceAgentTodosForSession(_ context.Context, _ string, _ []*gen.AgentTodo) error {
	return nil
}

func (a *agentSessionsWebAdapter) MergeAgentTodosForSession(_ context.Context, _ string, _ []*gen.AgentTodo) error {
	return nil
}

func (a *agentSessionsWebAdapter) GetChatSessionEntryInSession(ctx context.Context, sessionID, flag string) (*gen.ChatSessionEntry, error) {
	row, err := a.ent.ChatSessionEntry.Query().
		Where(
			chatsessionentry.SessionIDEQ(sessionID),
			chatsessionentry.FlagEQ(flag),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return row, nil
}

var _ = Describe("Agent Sessions UI", Label("module", "web"), func() {
	var (
		origDB     store.Adapter
		adapter    *agentSessionsWebAdapter
		sessionID  string
		entryID    string
	)

	BeforeEach(func() {
		origDB = store.Database
		sessionID = "bdd-session-" + types.Id()
		entryID = "bdd-entry-" + types.Id()

		adapter = &agentSessionsWebAdapter{
			ent: EntClient,
			uid: "bdd-agent-sessions-" + types.Id(),
		}
		store.Database = adapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)

		ctx := context.Background()
		EntClient.ChatSession.Create().
			SetFlag(sessionID).
			SetUID(adapter.uid).
			SetState(int(schema.ChatSessionActive)).
			SetCreatedAt(time.Now().Add(-time.Hour)).
			SetUpdatedAt(time.Now()).
			SaveX(ctx)

		EntClient.ChatSessionEntry.Create().
			SetFlag(entryID).
			SetSessionID(sessionID).
			SetEntryType("message").
			SetPayload(map[string]any{"role": "user", "content": "hello"}).
			SetCreatedAt(time.Now()).
			SaveX(ctx)
	})

	AfterEach(func() {
		ctx := context.Background()
		EntClient.ChatSessionEntry.Delete().Where(chatsessionentry.SessionIDEQ(sessionID)).ExecX(ctx)
		EntClient.ChatSession.Delete().Where(chatsession.FlagEQ(sessionID)).ExecX(ctx)
		store.Database = origDB
	})

	Describe("GET /service/web/agent-sessions", func() {
		It("renders the sessions list page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/agent-sessions", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agent-sessions-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring("Agent Sessions"))
			Expect(string(body)).To(ContainSubstring(sessionID))
		})

		It("redirects unauthenticated users to login", func() {
			req := MakeRequest(http.MethodGet, "/service/web/agent-sessions", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
		})
	})

	Describe("GET /service/web/agent-sessions/:id", func() {
		It("renders session detail with entries", func() {
			req := MakeRequest(http.MethodGet, fmt.Sprintf("/service/web/agent-sessions/%s", sessionID), nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agent-sessions-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring(sessionID))
			Expect(string(body)).To(ContainSubstring("message"))
			Expect(string(body)).To(ContainSubstring(entryID))
		})

		It("returns 404 for unknown session", func() {
			req := MakeRequest(http.MethodGet, "/service/web/agent-sessions/bdd-no-such-session", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agent-sessions-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})

//go:build integration

package specs

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/gomega"
)

// mountChatAgentRoutes registers Chat Agent HTTP routes on the shared test app.
func mountChatAgentRoutes(app *fiber.App) {
	server.RegisterChatAgentRoutes(app)
}

// createChatAgentAccessToken stores a scoped access token for Chat Agent BDD tests.
func createChatAgentAccessToken(ctx context.Context, uid types.Uid) string {
	token, err := auth.NewToken()
	Expect(err).NotTo(HaveOccurred())

	params := types.KV{
		"uid":    string(uid),
		"topic":  "chat-agent-bdd",
		"scopes": []string{auth.ScopeChatAgentChat},
	}
	expiredAt := time.Now().Add(24 * time.Hour)
	Expect(store.Database.ParameterSet(ctx, token, params, expiredAt)).To(Succeed())
	return token
}

func chatAgentRequest(method, path, token string, body []byte) *http.Request {
	req := JSONRequest(method, path, body)
	req.Header.Set("X-AccessToken", token)
	return req
}

func parseSSEBody(body []byte) []client.ChatStreamEvent {
	reader := bufio.NewReader(strings.NewReader(string(body)))
	var events []client.ChatStreamEvent
	var dataLines []string

	flush := func() {
		if len(dataLines) == 0 {
			return
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = nil
		if payload == "" {
			return
		}
		var event client.ChatStreamEvent
		Expect(sonic.UnmarshalString(payload, &event)).To(Succeed())
		events = append(events, event)
	}

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			flush()
			break
		}
		Expect(err).NotTo(HaveOccurred())
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return events
}

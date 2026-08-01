//go:build integration

package specs

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

var _ = Describe("Platform chat thread delivery", Label("module", "chat-agent", "platform"), func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("resolves Slack thread_id from session message content for scheduled delivery", func() {
		sessionID := types.Id()
		Expect(store.ChatStoreFromDB().CreateChatSession(ctx, &gen.ChatSession{
			Flag:  sessionID,
			UID:   "bdd-user",
			State: int(schema.ChatSessionActive),
		})).To(Succeed())

		platformID, err := store.PlatformStoreFromDB().CreatePlatform(ctx, &gen.Platform{Name: "slack-bdd"})
		Expect(err).NotTo(HaveOccurred())
		Expect(store.MessageStoreFromDB().CreateMessage(ctx, gen.Message{
			Flag:       types.Id(),
			PlatformID: platformID,
			Topic:      "D-BDD",
			Role:       types.User,
			Session:    sessionID,
			Content:    map[string]any{"text": "hello", "thread_id": "1700000000.000900"},
			State:      int(schema.MessageCreated),
		})).To(Succeed())

		delivery := chatagent.ResolveDeliveryContext(ctx, sessionID)
		Expect(delivery.Topic).To(Equal("D-BDD"))
		Expect(delivery.Platform).To(Equal("slack-bdd"))
		Expect(delivery.ThreadID).To(Equal("1700000000.000900"))
	})
})

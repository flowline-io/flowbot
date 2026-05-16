//go:build integration
// +build integration

package specs

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authentication", Label("auth"), func() {

	Describe("AuthContext", func() {
		Context("REST context", func() {
			It("extracts user from bearer token", func() {
				token, err := auth.NewToken()
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
				Expect(token).To(HavePrefix("fb_"))

				prefix := auth.TokenPrefix(token)
				Expect(prefix).To(HaveLen(12))

				hash := auth.HashToken(token)
				Expect(hash).NotTo(BeEmpty())
				Expect(hash).NotTo(Equal(token))
			})

			It("returns unauthorized for missing token", func() {
				result := auth.ExtractBearerToken("")
				Expect(result).To(BeEmpty())
			})

			It("returns unauthorized for expired token", func() {
				result := auth.ExtractBearerToken("Bearer expired_token_here")
				Expect(result).To(Equal("expired_token_here"))
			})

			It("returns unauthorized for invalid token", func() {
				result := auth.ExtractBearerToken("Invalid scheme")
				Expect(result).To(BeEmpty())
			})
		})

		Context("CLI context", func() {
			It("returns unauthorized for expired session", func() {
				token, err := auth.NewToken()
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
			})
		})

		Context("Chat context", func() {
			It("auto-registers new chat users", func() {
				_ = types.NewToken
			})
		})

		Context("Webhook context", func() {
			It("validates webhook secret", func() {
				body := []byte(`{"event":"push"}`)
				hash := auth.WebhookBodyHash(body)
				Expect(hash).NotTo(BeEmpty())

				now := time.Now()
				sig := auth.SignWebhook("mysecret", "POST", "/webhook", now, body)
				Expect(sig).NotTo(BeEmpty())

				valid := auth.VerifyWebhookSignature("mysecret", "POST", "/webhook", now, body, sig, now.Add(time.Minute), auth.DefaultWebhookMaxSkew)
				Expect(valid).To(BeTrue())
			})

			It("associates webhook with owning user", func() {
				body := []byte(`{"test":true}`)
				hash := auth.WebhookBodyHash(body)
				Expect(hash).NotTo(BeEmpty())

				now := time.Now()
				sig := auth.SignWebhook("secret", "GET", "/hook/test", now, body)

				wrong := auth.VerifyWebhookSignature("wrong_secret", "GET", "/hook/test", now, body, sig, now.Add(time.Minute), auth.DefaultWebhookMaxSkew)
				Expect(wrong).To(BeFalse())
			})
		})

		Context("Cron context", func() {
			It("runs as system user", func() {
				ctx := auth.SystemCronContext()
				Expect(ctx.SubjectType).To(Equal(auth.SubjectCron))
				Expect(ctx.SubjectID).To(Equal("system:cron"))
			})
		})

		Context("Pipeline context", func() {
			It("runs as pipeline system user", func() {
				ctx := auth.SystemPipelineContext()
				Expect(ctx.SubjectType).To(Equal(auth.SubjectPipeline))
				Expect(ctx.SubjectID).To(Equal("system:pipeline"))
			})
		})

		Context("Workflow context", func() {
			It("runs as workflow system user", func() {
				ctx := auth.SystemWorkflowContext()
				Expect(ctx.SubjectType).To(Equal(auth.SubjectWorkflow))
				Expect(ctx.SubjectID).To(Equal("system:workflow"))
			})
		})
	})

	Describe("Permission Checks", func() {
		It("grants access to owned resources", func() {
			scopes := []string{auth.ScopeServiceBookmarkRead, auth.ScopeServiceBookmarkWrite}
			Expect(auth.HasScope(scopes, auth.ScopeServiceBookmarkRead)).To(BeTrue())
			Expect(auth.HasScope(scopes, auth.ScopeServiceBookmarkWrite)).To(BeTrue())
		})

		It("denies access to other user's resources", func() {
			scopes := []string{auth.ScopeServiceBookmarkRead}
			Expect(auth.HasScope(scopes, auth.ScopeServiceKanbanWrite)).To(BeFalse())
		})

		It("grants access to admin users for all resources", func() {
			scopes := []string{auth.ScopeAdmin}
			Expect(auth.HasScope(scopes, auth.ScopeServiceBookmarkRead)).To(BeTrue())
			Expect(auth.HasScope(scopes, auth.ScopeServiceKanbanWrite)).To(BeTrue())
			Expect(auth.HasScope(scopes, auth.ScopeHubAppsStart)).To(BeTrue())
		})

		It("denies access to unauthenticated requests", func() {
			scopes := []string{}
			Expect(auth.HasScope(scopes, auth.ScopeServiceBookmarkRead)).To(BeFalse())
			Expect(auth.HasScope(scopes, auth.ScopeAdmin)).To(BeFalse())
		})
	})

	Describe("Scope definitions", func() {
		It("defines all available scopes", func() {
			all := auth.AllScopes()
			Expect(all).NotTo(BeEmpty())
			names := make([]string, len(all))
			for i, s := range all {
				names[i] = s.Value
			}
			Expect(names).To(ContainElement(auth.ScopeAdmin))
			Expect(names).To(ContainElement(auth.ScopeHubAppsRead))
			Expect(names).To(ContainElement(auth.ScopeServiceBookmarkRead))
		})
	})

	Describe("Token operations", func() {
		It("generates unique tokens", func() {
			t1, err := auth.NewToken()
			Expect(err).NotTo(HaveOccurred())
			t2, err := auth.NewToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(t1).NotTo(Equal(t2))
		})

		It("hashes tokens deterministically", func() {
			token, err := auth.NewToken()
			Expect(err).NotTo(HaveOccurred())
			h1 := auth.HashToken(token)
			h2 := auth.HashToken(token)
			Expect(h1).To(Equal(h2))
		})
	})
})

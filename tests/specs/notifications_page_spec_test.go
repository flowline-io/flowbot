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
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notificationrecord"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type notifyWebAdapter struct {
	store.Adapter
	ent    *gen.Client
	uid    string
	scopes []string
}

func (a *notifyWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *notifyWebAdapter) Close() error                     { return nil }
func (a *notifyWebAdapter) IsOpen() bool                     { return true }
func (a *notifyWebAdapter) GetName() string                  { return "bdd-notify-page" }
func (a *notifyWebAdapter) Stats() any                       { return nil }
func (a *notifyWebAdapter) GetDB() any                       { return a.ent }

func (a *notifyWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:    1,
		Flag:  flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": a.scopes,
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

var _ = Describe("Notifications Pages", Label("module", "web"), func() {
	var (
		origDB            store.Adapter
		notifyAdapter     *notifyWebAdapter
		otherUserAdapter  *notifyWebAdapter
		testUID           string
		otherUID          string
		seedRecords       []*gen.NotificationRecord
		sentRecordID      int64
		otherFailedRecID  int64
	)

	BeforeEach(func() {
		origDB = store.Database
		testUID = "bdd-notify-uid-" + types.Id()
		otherUID = "bdd-notify-other-" + types.Id()

		notifyAdapter = &notifyWebAdapter{
			ent:    EntClient,
			uid:    testUID,
			scopes: []string{"read", "write"},
		}
		otherUserAdapter = &notifyWebAdapter{
			ent:    EntClient,
			uid:    otherUID,
			scopes: []string{"read", "write"},
		}
		store.Database = notifyAdapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"admin"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)

		ctx := context.Background()

		// Seed sent record for test user
		rec1 := EntClient.NotificationRecord.Create().
			SetUID(testUID).
			SetChannel("slack").
			SetTemplateID("test-template").
			SetSummary("Test notification sent").
			SetStatus("success").
			SetErrorMsg("").
			SetPayloadSnapshot(map[string]any{"key": "value"}).
			SetCreatedAt(time.Now().Add(-30 * time.Minute)).
			SaveX(ctx)
		sentRecordID = rec1.ID

		// Seed failed record for test user
		rec2 := EntClient.NotificationRecord.Create().
			SetUID(testUID).
			SetChannel("ntfy").
			SetTemplateID("alert-template").
			SetSummary("Alert delivery failed").
			SetStatus("failed").
			SetErrorMsg("connection refused").
			SetPayloadSnapshot(map[string]any{"alert": "disk full"}).
			SetCreatedAt(time.Now().Add(-10 * time.Minute)).
			SaveX(ctx)
		_ = rec2.ID

		// Seed another sent record for test user (for pagination)
		rec3 := EntClient.NotificationRecord.Create().
			SetUID(testUID).
			SetChannel("pushover").
			SetTemplateID("daily-digest").
			SetSummary("Daily digest sent").
			SetStatus("success").
			SetErrorMsg("").
			SetPayloadSnapshot(map[string]any{}).
			SetCreatedAt(time.Now().Add(-5 * time.Minute)).
			SaveX(ctx)

		// Seed failed record for other user
		rec4 := EntClient.NotificationRecord.Create().
			SetUID(otherUID).
			SetChannel("slack").
			SetTemplateID("other-template").
			SetSummary("Other user notification").
			SetStatus("failed").
			SetErrorMsg("timeout").
			SetPayloadSnapshot(map[string]any{}).
			SetCreatedAt(time.Now()).
			SaveX(ctx)
		otherFailedRecID = rec4.ID

		seedRecords = []*gen.NotificationRecord{rec1, rec2, rec3, rec4}
	})

	AfterEach(func() {
		for _, r := range seedRecords {
			EntClient.NotificationRecord.Delete().Where(
				notificationrecord.ID(r.ID),
			).ExecX(context.Background())
		}
		store.Database = origDB
	})

	Describe("GET /notifications", func() {
		Context("with valid auth token", func() {
			It("returns the notifications page with records", func() {
				req := MakeRequest(http.MethodGet, "/service/web/notifications", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: notifyAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("Test notification sent"))
			})
		})

		Context("without authentication", func() {
			It("redirects to login page", func() {
				req := MakeRequest(http.MethodGet, "/service/web/notifications", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
				Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
			})
		})
	})

	Describe("GET /notifications/list", func() {
		Context("with valid auth token", func() {
			It("returns the notification table fragment with user records", func() {
				req := MakeRequest(http.MethodGet, "/service/web/notifications/list", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: notifyAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("Test notification sent"))
				Expect(body).To(ContainSubstring("Alert delivery failed"))
			})

			It("supports cursor-based pagination", func() {
				// Seed extra records to trigger pagination (limit is 20)
				extraRecords := make([]*gen.NotificationRecord, 0)
				for i := range 20 {
					rec := EntClient.NotificationRecord.Create().
						SetUID(testUID).
						SetChannel("slack").
						SetTemplateID(fmt.Sprintf("paginated-%d", i)).
						SetSummary(fmt.Sprintf("Paginated record %d", i)).
						SetStatus("success").
						SetErrorMsg("").
						SetPayloadSnapshot(map[string]any{}).
						SetCreatedAt(time.Now().Add(time.Duration(i) * time.Second)).
						SaveX(context.Background())
					extraRecords = append(extraRecords, rec)
				}
				DeferCleanup(func() {
					for _, r := range extraRecords {
						EntClient.NotificationRecord.Delete().Where(
							notificationrecord.ID(r.ID),
						).ExecX(context.Background())
					}
				})

				req := MakeRequest(http.MethodGet, "/service/web/notifications/list", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: notifyAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring(`hx-get="/service/web/notifications/list`))
			})

			It("shows empty table for a user with no records", func() {
				store.Database = otherUserAdapter

				ctx := context.Background()
				EntClient.NotificationRecord.Delete().Where(
					notificationrecord.UID(otherUID),
				).ExecX(ctx)

				req := MakeRequest(http.MethodGet, "/service/web/notifications/list", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: otherUserAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(Or(
					ContainSubstring("No notifications"),
					ContainSubstring("empty"),
				))
			})
		})
	})

	Describe("POST /notifications/:id/retry", func() {
		Context("with valid auth token", func() {
			It("rejects invalid ID format", func() {
				req := MakeRequest(http.MethodPost, "/service/web/notifications/abc/retry", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: notifyAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(string(ReadBody(resp))).To(Equal("Invalid ID"))
			})

			It("returns not-found for non-existent record", func() {
				req := MakeRequest(http.MethodPost, "/service/web/notifications/99999/retry", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: notifyAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(string(ReadBody(resp))).To(ContainSubstring("Record not found"))
			})

			It("rejects non-failed record", func() {
				req := MakeRequest(http.MethodPost, "/service/web/notifications/"+strconv.FormatInt(sentRecordID, 10)+"/retry", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: notifyAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(string(ReadBody(resp))).To(ContainSubstring("Only failed notifications can be retried"))
			})

			It("rejects wrong user record", func() {
				store.Database = notifyAdapter
				req := MakeRequest(http.MethodPost, "/service/web/notifications/"+strconv.FormatInt(otherFailedRecID, 10)+"/retry", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: notifyAdapter.uid})
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
				Expect(string(ReadBody(resp))).To(Equal("Not your notification"))
			})
		})
	})
})

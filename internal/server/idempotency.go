package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/idempotency"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

func init() {
	route.SetIdempotencyWrap(serveIdempotency)
}

func serveIdempotency(c fiber.Ctx, operation string, next fiber.Handler) error {
	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if key == "" {
		return next(c)
	}
	if err := idempotency.Validate(key); err != nil {
		return err
	}
	tokenID, err := idempotencyTokenID(c)
	if err != nil {
		return err
	}
	paramHash, err := idempotency.HashJSON(c.Body())
	if err != nil {
		return &types.Error{
			Kind:    types.ErrInvalidArgument,
			Message: "idempotency request body must be JSON",
			Field:   "body",
		}
	}
	st := store.IdempotencyStoreFromDB()
	if !st.Ready() {
		return types.Errorf(types.ErrUnavailable, "idempotency store unavailable")
	}
	correlation := types.NewCorrelationID()
	c.SetContext(types.WithLineage(c.Context(), correlation, ""))
	claim, err := claimIdempotency(c, st, store.IdempotencyRef{
		TokenID:       tokenID,
		Operation:     operation,
		Key:           key,
		ParamHash:     paramHash,
		CorrelationID: correlation,
	})
	if err != nil {
		return err
	}
	if claim.Outcome != idempotency.OutcomeRun {
		return writeIdempotencyOutcome(c, operation, key, claim)
	}
	return finishIdempotency(c, st, claim, next)
}

func idempotencyTokenID(c fiber.Ctx) (int64, error) {
	raw := strings.TrimSpace(c.Get("X-AccessToken"))
	if raw == "" {
		header := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		raw = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "bearer "))
	}
	if raw == "" {
		return 0, types.Errorf(types.ErrUnauthorized, "idempotency key requires an access token")
	}
	tok, err := route.LookupAccessToken(c.Context(), raw)
	if err != nil {
		return 0, err
	}
	if tok.ID <= 0 {
		return 0, types.Errorf(types.ErrUnauthorized, "idempotency key requires an access token")
	}
	return tok.ID, nil
}

func claimIdempotency(c fiber.Ctx, st *store.IdempotencyStore, ref store.IdempotencyRef) (idempotency.Claim, error) {
	row, found, err := st.Lookup(c.Context(), ref)
	if err != nil {
		return idempotency.Claim{}, err
	}
	if found {
		return store.IdempotencyView(row, ref.ParamHash, false), nil
	}
	row, created, err := st.Begin(c.Context(), ref)
	if err != nil {
		return idempotency.Claim{}, err
	}
	return store.IdempotencyView(row, ref.ParamHash, created), nil
}

func finishIdempotency(c fiber.Ctx, st *store.IdempotencyStore, claim idempotency.Claim, next fiber.Handler) error {
	callErr := next(c)
	if callErr != nil {
		settleHTTPClaim(c, st, claim.ID, callErr)
		return callErr
	}
	status := c.Response().StatusCode()
	if status == http.StatusBadRequest {
		settleHTTPClaim(c, st, claim.ID, types.ErrInvalidArgument)
		return nil
	}
	if status >= 400 {
		settleHTTPClaim(c, st, claim.ID, types.ErrUnavailable)
		return nil
	}
	body := append([]byte(nil), c.Response().Body()...)
	if err := st.Finish(c.Context(), claim.ID, body, claim.CorrelationID, idempotency.MaxResultBytes()); err != nil {
		flog.Error(err)
	}
	return nil
}

func settleHTTPClaim(c fiber.Ctx, st *store.IdempotencyStore, id int64, callErr error) {
	var err error
	if errors.Is(callErr, types.ErrInvalidArgument) {
		err = st.Release(c.Context(), id)
	} else {
		err = st.Fail(c.Context(), id)
	}
	if err != nil {
		flog.Error(err)
	}
}

func writeIdempotencyOutcome(c fiber.Ctx, operation, key string, claim idempotency.Claim) error {
	if claim.Outcome == idempotency.OutcomeReplay {
		c.Set("Idempotency-Replayed", "true")
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		auditHTTPReplay(c, operation, key, claim.CorrelationID)
		return c.SendString(claim.ResultJSON)
	}
	return idempotency.ErrorFor(claim.Outcome)
}

func auditHTTPReplay(c fiber.Ctx, operation, key, correlationID string) {
	auditor := route.Auditor()
	if auditor == nil {
		return
	}
	subject := &audit.Subject{SubjectType: "token"}
	if rc := route.GetRequestContext(c); rc != nil {
		subject.UID = string(rc.UID)
		subject.SubjectID = string(rc.UID)
	}
	if err := auditor.RecordSuccess(c.Context(), audit.Entry{
		Subject:       subject,
		Action:        "http.idempotency.replay",
		Target:        audit.Target{Type: "route", ID: operation},
		CorrelationID: correlationID,
		Request: map[string]any{
			"idempotency_key": key,
			"replayed":        true,
		},
	}); err != nil {
		flog.Error(err)
	}
}

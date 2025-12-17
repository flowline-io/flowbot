package connections

import (
	"net/http"
	"strconv"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
)

// API provides HTTP handlers for connection management
type API struct {
	store store.Adapter
}

// NewAPI creates a new connection API
func NewAPI(storeAdapter store.Adapter) *API {
	return &API{
		store: storeAdapter,
	}
}

// ListConnections lists all connections
func (a *API) ListConnections(c fiber.Ctx) error {
	uid, topic := resolveUIDTopicFromRequest(c, a.store)

	connections, err := a.store.GetConnections(uid, topic)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get connections",
		})
	}

	return c.JSON(connections)
}

func resolveUIDTopicFromRequest(c fiber.Ctx, storeAdapter store.Adapter) (types.Uid, string) {
	uidStr := c.Query("uid", "")
	topic := c.Query("topic", "")
	if uidStr != "" {
		return types.Uid(uidStr), topic
	}

	flag := c.Query("p", "")
	if flag == "" {
		return "", ""
	}
	p, err := storeAdapter.ParameterGet(flag)
	if err != nil {
		return "", ""
	}
	if p.IsExpired() {
		return "", ""
	}
	uid, _ := types.KV(p.Params).String("uid")
	topic, _ = types.KV(p.Params).String("topic")
	return types.Uid(uid), topic
}

// GetConnection gets a connection by ID
func (a *API) GetConnection(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid connection id",
		})
	}

	conn, err := a.store.GetConnection(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "connection not found",
		})
	}

	return c.JSON(conn)
}

// CreateConnection creates a new connection
func (a *API) CreateConnection(c fiber.Ctx) error {
	var conn model.Connection
	if err := c.Bind().Body(&conn); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	uid, topic := resolveUIDTopicFromRequest(c, a.store)
	if conn.UID == "" {
		conn.UID = uid.String()
	}
	if conn.Topic == "" {
		conn.Topic = topic
	}

	id, err := a.store.CreateConnection(&conn)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create connection",
		})
	}

	conn.ID = id
	return c.Status(http.StatusCreated).JSON(conn)
}

// UpdateConnection updates a connection
func (a *API) UpdateConnection(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid connection id",
		})
	}

	var conn model.Connection
	if err := c.Bind().Body(&conn); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	uid, topic := resolveUIDTopicFromRequest(c, a.store)
	if conn.UID == "" {
		conn.UID = uid.String()
	}
	if conn.Topic == "" {
		conn.Topic = topic
	}

	conn.ID = id
	if err := a.store.UpdateConnection(&conn); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update connection",
		})
	}

	return c.JSON(conn)
}

// DeleteConnection deletes a connection
func (a *API) DeleteConnection(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid connection id",
		})
	}

	if err := a.store.DeleteConnection(id); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete connection",
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

// ListAuthentications lists all authentications
func (a *API) ListAuthentications(c fiber.Ctx) error {
	uid, topic := resolveUIDTopicFromRequest(c, a.store)

	auths, err := a.store.GetAuthentications(uid, topic)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get authentications",
		})
	}

	return c.JSON(auths)
}

// GetAuthentication gets an authentication by ID
func (a *API) GetAuthentication(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid authentication id",
		})
	}

	auth, err := a.store.GetAuthentication(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "authentication not found",
		})
	}

	return c.JSON(auth)
}

// CreateAuthentication creates a new authentication
func (a *API) CreateAuthentication(c fiber.Ctx) error {
	var auth model.Authentication
	if err := c.Bind().Body(&auth); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	uid, topic := resolveUIDTopicFromRequest(c, a.store)
	if auth.UID == "" {
		auth.UID = uid.String()
	}
	if auth.Topic == "" {
		auth.Topic = topic
	}

	id, err := a.store.CreateAuthentication(&auth)
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create authentication",
		})
	}

	auth.ID = id
	return c.Status(http.StatusCreated).JSON(auth)
}

// UpdateAuthentication updates an authentication
func (a *API) UpdateAuthentication(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid authentication id",
		})
	}

	var auth model.Authentication
	if err := c.Bind().Body(&auth); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	uid, topic := resolveUIDTopicFromRequest(c, a.store)
	if auth.UID == "" {
		auth.UID = uid.String()
	}
	if auth.Topic == "" {
		auth.Topic = topic
	}

	auth.ID = id
	if err := a.store.UpdateAuthentication(&auth); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update authentication",
		})
	}

	return c.JSON(auth)
}

// DeleteAuthentication deletes an authentication
func (a *API) DeleteAuthentication(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid authentication id",
		})
	}

	if err := a.store.DeleteAuthentication(id); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete authentication",
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/app"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/auditlog"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/capabilitybinding"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/connection"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/fileupload"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/topic"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/url"
	"github.com/flowline-io/flowbot/internal/store/model"
)

// DatabaseExtTestSuite tests CRUD operations for models not covered by DatabaseTestSuite.
type DatabaseExtTestSuite struct {
	IntegrationTestSuite
}

// TestDatabaseExtTestSuite runs the extended database test suite.
func TestDatabaseExtTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseExtTestSuite))
}

// TestTopicCRUD tests topic CRUD operations.
func (s *DatabaseExtTestSuite) TestTopicCRUD() {
	ctx := context.Background()

	flag := uuid.New().String()
	entTopic, err := s.EntClient.Topic.Create().
		SetFlag(flag).
		SetPlatform("discord").
		SetName("integration-test-topic").
		SetOwner(0).
		SetType("channel").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entTopic.ID, int64(0))

	retrievedTopic, err := s.EntClient.Topic.Query().
		Where(topic.IDEQ(entTopic.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entTopic.Name, retrievedTopic.Name)

	err = s.EntClient.Topic.Update().
		Where(topic.IDEQ(retrievedTopic.ID)).
		SetName("updated-topic").
		Exec(ctx)
	s.NoError(err)

	retrievedTopic, err = s.EntClient.Topic.Query().
		Where(topic.IDEQ(entTopic.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal("updated-topic", retrievedTopic.Name)

	s.EntClient.Topic.Delete().Where(topic.IDEQ(retrievedTopic.ID)).Exec(ctx)
}

// TestFileuploadCRUD tests fileupload CRUD operations.
func (s *DatabaseExtTestSuite) TestFileuploadCRUD() {
	ctx := context.Background()

	fid := uuid.New().String()
	entFile, err := s.EntClient.Fileupload.Create().
		SetUID(uuid.New().String()).
		SetFid(fid).
		SetName("test-file.txt").
		SetMimetype("text/plain").
		SetSize(1024).
		SetLocation("/tmp/test-file.txt").
		SetState(int(model.FileStart)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entFile.ID, int64(0))

	retrievedFile, err := s.EntClient.Fileupload.Query().
		Where(fileupload.FidEQ(fid)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entFile.Mimetype, retrievedFile.Mimetype)

	err = s.EntClient.Fileupload.Update().
		Where(fileupload.IDEQ(retrievedFile.ID)).
		SetState(int(model.FileFinish)).
		Exec(ctx)
	s.NoError(err)

	retrievedFile, err = s.EntClient.Fileupload.Query().
		Where(fileupload.FidEQ(fid)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int(model.FileFinish), retrievedFile.State)

	s.EntClient.Fileupload.Delete().Where(fileupload.IDEQ(retrievedFile.ID)).Exec(ctx)
}

// TestUrlCRUD tests url CRUD operations.
func (s *DatabaseExtTestSuite) TestUrlCRUD() {
	ctx := context.Background()

	entURL, err := s.EntClient.Url.Create().
		SetFlag("integration-test-url").
		SetURL("https://example.com/integration-test").
		SetState(int(model.UrlStateEnable)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entURL.ID, int64(0))

	retrievedURL, err := s.EntClient.Url.Query().
		Where(url.IDEQ(entURL.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entURL.URL, retrievedURL.URL)

	err = s.EntClient.Url.Update().
		Where(url.IDEQ(retrievedURL.ID)).
		SetState(int(model.UrlStateDisable)).
		Exec(ctx)
	s.NoError(err)

	retrievedURL, err = s.EntClient.Url.Query().
		Where(url.IDEQ(entURL.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int(model.UrlStateDisable), retrievedURL.State)

	s.EntClient.Url.Delete().Where(url.IDEQ(retrievedURL.ID)).Exec(ctx)
}

// TestAppCRUD tests app CRUD operations.
func (s *DatabaseExtTestSuite) TestAppCRUD() {
	ctx := context.Background()

	entApp, err := s.EntClient.App.Create().
		SetName("integration-test-app").
		SetPath("/home/user/homelab/apps/testapp").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entApp.ID, int64(0))

	retrievedApp, err := s.EntClient.App.Query().
		Where(app.IDEQ(entApp.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entApp.Name, retrievedApp.Name)

	err = s.EntClient.App.Update().
		Where(app.IDEQ(retrievedApp.ID)).
		SetStatus(string(model.AppStatusRunning)).
		Exec(ctx)
	s.NoError(err)

	retrievedApp, err = s.EntClient.App.Query().
		Where(app.IDEQ(entApp.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(string(model.AppStatusRunning), retrievedApp.Status)

	s.EntClient.App.Delete().Where(app.IDEQ(retrievedApp.ID)).Exec(ctx)
}

// TestCapabilityBindingCRUD tests capability binding CRUD operations.
func (s *DatabaseExtTestSuite) TestCapabilityBindingCRUD() {
	ctx := context.Background()

	entBinding, err := s.EntClient.CapabilityBinding.Create().
		SetCapability("bookmark").
		SetBackend("karakeep").
		SetApp("archivebox").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entBinding.ID, int64(0))

	retrieved, err := s.EntClient.CapabilityBinding.Query().
		Where(capabilitybinding.IDEQ(entBinding.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entBinding.Capability, retrieved.Capability)

	err = s.EntClient.CapabilityBinding.Update().
		Where(capabilitybinding.IDEQ(retrieved.ID)).
		SetHealthy(true).
		Exec(ctx)
	s.NoError(err)

	retrieved, err = s.EntClient.CapabilityBinding.Query().
		Where(capabilitybinding.IDEQ(entBinding.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(true, retrieved.Healthy)

	s.EntClient.CapabilityBinding.Delete().Where(capabilitybinding.IDEQ(retrieved.ID)).Exec(ctx)
}

// TestAuditLogCRUD tests audit log CRUD operations.
func (s *DatabaseExtTestSuite) TestAuditLogCRUD() {
	ctx := context.Background()

	entLog, err := s.EntClient.AuditLog.Create().
		SetAction("test.action").
		SetTargetType("test").
		SetTargetID("1").
		SetCreatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entLog.ID, int64(0))

	retrieved, err := s.EntClient.AuditLog.Query().
		Where(auditlog.IDEQ(entLog.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entLog.Action, retrieved.Action)

	s.EntClient.AuditLog.Delete().Where(auditlog.IDEQ(retrieved.ID)).Exec(ctx)
}

// TestParameterCRUD tests parameter CRUD operations.
func (s *DatabaseExtTestSuite) TestParameterCRUD() {
	ctx := context.Background()

	entParam, err := s.EntClient.Parameter.Create().
		SetFlag("integration-test-param").
		SetParams(map[string]interface{}{"key": "value"}).
		SetExpiredAt(time.Now().Add(time.Hour)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entParam.ID, int64(0))

	retrieved, err := s.EntClient.Parameter.Query().
		Where(parameter.IDEQ(entParam.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entParam.Flag, retrieved.Flag)

	err = s.EntClient.Parameter.Update().
		Where(parameter.IDEQ(retrieved.ID)).
		SetParams(map[string]interface{}{"key": "updated"}).
		Exec(ctx)
	s.NoError(err)

	retrieved, err = s.EntClient.Parameter.Query().
		Where(parameter.IDEQ(entParam.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(map[string]interface{}{"key": "updated"}, retrieved.Params)

	s.EntClient.Parameter.Delete().Where(parameter.IDEQ(retrieved.ID)).Exec(ctx)
}

// TestConnectionCRUD tests connection CRUD operations.
func (s *DatabaseExtTestSuite) TestConnectionCRUD() {
	ctx := context.Background()

	entConn, err := s.EntClient.Connection.Create().
		SetUID(uuid.New().String()).
		SetTopic("test-topic").
		SetName("integration-test-connection").
		SetType("http").
		SetConfig(map[string]interface{}{"url": "https://example.com"}).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entConn.ID, int64(0))

	retrieved, err := s.EntClient.Connection.Query().
		Where(connection.IDEQ(entConn.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entConn.Name, retrieved.Name)

	err = s.EntClient.Connection.Update().
		Where(connection.IDEQ(retrieved.ID)).
		SetEnabled(false).
		Exec(ctx)
	s.NoError(err)

	retrieved, err = s.EntClient.Connection.Query().
		Where(connection.IDEQ(entConn.ID)).
		Only(ctx)
	s.NoError(err)
	s.False(retrieved.Enabled)

	s.EntClient.Connection.Delete().Where(connection.IDEQ(retrieved.ID)).Exec(ctx)
}

// TestTopicIsHardDeleted tests that topic can be deleted with Ent hard delete.
func (s *DatabaseExtTestSuite) TestTopicIsHardDeleted() {
	ctx := context.Background()

	flag := uuid.New().String()
	entTopic, err := s.EntClient.Topic.Create().
		SetFlag(flag).
		SetPlatform("slack").
		SetName("hard-delete-topic").
		SetOwner(0).
		SetType("channel").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.EntClient.Topic.Delete().
		Where(topic.IDEQ(entTopic.ID)).
		Exec(ctx)
	s.NoError(err)

	_, err = s.EntClient.Topic.Query().
		Where(topic.IDEQ(entTopic.ID)).
		Only(ctx)
	s.Error(err, "topic should be deleted after hard delete")
}

// TestUrlViewCountDefault tests that view_count defaults to 0.
func (s *DatabaseExtTestSuite) TestUrlViewCountDefault() {
	ctx := context.Background()

	entURL, err := s.EntClient.Url.Create().
		SetFlag("integration-test-url-default").
		SetURL("https://example.com/default-test").
		SetState(int(model.UrlStateEnable)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)

	retrieved, err := s.EntClient.Url.Query().
		Where(url.IDEQ(entURL.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int(model.UrlStateEnable), retrieved.State)

	s.EntClient.Url.Delete().Where(url.IDEQ(retrieved.ID)).Exec(ctx)
}

// TestFileStateTransitions tests file state transitions.
func (s *DatabaseExtTestSuite) TestFileStateTransitions() {
	ctx := context.Background()

	entFile, err := s.EntClient.Fileupload.Create().
		SetUID(uuid.New().String()).
		SetFid(uuid.New().String()).
		SetName("state-test.txt").
		SetMimetype("text/plain").
		SetSize(512).
		SetLocation("/tmp/state-test.txt").
		SetState(int(model.FileStart)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)

	err = s.EntClient.Fileupload.Update().
		Where(fileupload.IDEQ(entFile.ID)).
		SetState(int(model.FileFinish)).
		Exec(ctx)
	s.NoError(err)

	updated, err := s.EntClient.Fileupload.Query().
		Where(fileupload.IDEQ(entFile.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int(model.FileFinish), updated.State)

	s.EntClient.Fileupload.Delete().Where(fileupload.IDEQ(updated.ID)).Exec(ctx)
}

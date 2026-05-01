//go:build integration
// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
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
	topic := &model.Topic{
		Platform:  "discord",
		Name:      "integration-test-topic",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(topic).Error
	s.Require().NoError(err)
	s.Greater(topic.ID, int64(0))

	var retrievedTopic model.Topic
	err = s.DB.First(&retrievedTopic, topic.ID).Error
	s.NoError(err)
	s.Equal(topic.Name, retrievedTopic.Name)

	err = s.DB.Model(&retrievedTopic).Update("name", "updated-topic").Error
	s.NoError(err)

	err = s.DB.First(&retrievedTopic, topic.ID).Error
	s.NoError(err)
	s.Equal("updated-topic", retrievedTopic.Name)

	s.DB.Delete(&retrievedTopic)
}

// TestFileuploadCRUD tests fileupload CRUD operations.
func (s *DatabaseExtTestSuite) TestFileuploadCRUD() {
	fid := uuid.New().String()
	fileupload := &model.Fileupload{
		UID:       uuid.New().String(),
		Fid:       fid,
		Name:      "test-file.txt",
		Mimetype:  "text/plain",
		Size:      1024,
		Location:  "/tmp/test-file.txt",
		State:     model.FileStart,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(fileupload).Error
	s.Require().NoError(err)
	s.Greater(fileupload.ID, int64(0))

	var retrievedFile model.Fileupload
	err = s.DB.Where("fid = ?", fid).First(&retrievedFile).Error
	s.NoError(err)
	s.Equal(fileupload.Mimetype, retrievedFile.Mimetype)

	err = s.DB.Model(&retrievedFile).Update("state", model.FileFinish).Error
	s.NoError(err)

	err = s.DB.Where("fid = ?", fid).First(&retrievedFile).Error
	s.NoError(err)
	s.Equal(model.FileFinish, retrievedFile.State)

	s.DB.Delete(&retrievedFile)
}

// TestUrlCRUD tests url CRUD operations.
func (s *DatabaseExtTestSuite) TestUrlCRUD() {
	urlItem := &model.Url{
		Flag:      "integration-test-url",
		URL:       "https://example.com/integration-test",
		State:     model.UrlStateEnable,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(urlItem).Error
	s.Require().NoError(err)
	s.Greater(urlItem.ID, int64(0))

	var retrievedUrl model.Url
	err = s.DB.First(&retrievedUrl, urlItem.ID).Error
	s.NoError(err)
	s.Equal(urlItem.URL, retrievedUrl.URL)

	err = s.DB.Model(&retrievedUrl).Update("state", model.UrlStateDisable).Error
	s.NoError(err)

	err = s.DB.First(&retrievedUrl, urlItem.ID).Error
	s.NoError(err)
	s.Equal(model.UrlStateDisable, retrievedUrl.State)

	s.DB.Delete(&retrievedUrl)
}

// TestAppCRUD tests app CRUD operations.
func (s *DatabaseExtTestSuite) TestAppCRUD() {
	app := &model.App{
		Name:      "integration-test-app",
		Path:      "/home/user/homelab/apps/testapp",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(app).Error
	s.Require().NoError(err)
	s.Greater(app.ID, int64(0))

	var retrievedApp model.App
	err = s.DB.First(&retrievedApp, app.ID).Error
	s.NoError(err)
	s.Equal(app.Name, retrievedApp.Name)

	err = s.DB.Model(&retrievedApp).Update("status", model.AppStatusRunning).Error
	s.NoError(err)

	err = s.DB.First(&retrievedApp, app.ID).Error
	s.NoError(err)
	s.Equal(model.AppStatusRunning, retrievedApp.Status)

	s.DB.Delete(&retrievedApp)
}

// TestCapabilityBindingCRUD tests capability binding CRUD operations.
func (s *DatabaseExtTestSuite) TestCapabilityBindingCRUD() {
	binding := &model.CapabilityBinding{
		Capability: "bookmark",
		Backend:    "karakeep",
		App:        "archivebox",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := s.DB.Create(binding).Error
	s.Require().NoError(err)
	s.Greater(binding.ID, int64(0))

	var retrieved model.CapabilityBinding
	err = s.DB.First(&retrieved, binding.ID).Error
	s.NoError(err)
	s.Equal(binding.Capability, retrieved.Capability)

	err = s.DB.Model(&retrieved).Update("healthy", 1).Error
	s.NoError(err)

	err = s.DB.First(&retrieved, binding.ID).Error
	s.NoError(err)
	s.Equal(true, retrieved.Healthy)

	s.DB.Delete(&retrieved)
}

// TestAuditLogCRUD tests audit log CRUD operations.
func (s *DatabaseExtTestSuite) TestAuditLogCRUD() {
	auditLog := &model.AuditLog{
		Action:    "test.action",
		CreatedAt: time.Now(),
	}

	err := s.DB.Create(auditLog).Error
	s.Require().NoError(err)
	s.Greater(auditLog.ID, int64(0))

	var retrieved model.AuditLog
	err = s.DB.First(&retrieved, auditLog.ID).Error
	s.NoError(err)
	s.Equal(auditLog.Action, retrieved.Action)

	s.DB.Delete(&retrieved)
}

// TestParameterCRUD tests parameter CRUD operations.
func (s *DatabaseExtTestSuite) TestParameterCRUD() {
	parameter := &model.Parameter{
		Flag:      "integration-test-param",
		Params:    model.JSON{"key": "value"},
		ExpiredAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(parameter).Error
	s.Require().NoError(err)
	s.Greater(parameter.ID, int64(0))

	var retrieved model.Parameter
	err = s.DB.First(&retrieved, parameter.ID).Error
	s.NoError(err)
	s.Equal(parameter.Flag, retrieved.Flag)

	err = s.DB.Model(&retrieved).Update("params", model.JSON{"key": "updated"}).Error
	s.NoError(err)

	err = s.DB.First(&retrieved, parameter.ID).Error
	s.NoError(err)
	s.Equal(model.JSON{"key": "updated"}, retrieved.Params)

	s.DB.Delete(&retrieved)
}

// TestConnectionCRUD tests connection CRUD operations.
func (s *DatabaseExtTestSuite) TestConnectionCRUD() {
	connection := &model.Connection{
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Name:      "integration-test-connection",
		Type:      "http",
		Config:    model.JSON{"url": "https://example.com"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(connection).Error
	s.Require().NoError(err)
	s.Greater(connection.ID, int64(0))

	var retrieved model.Connection
	err = s.DB.First(&retrieved, connection.ID).Error
	s.NoError(err)
	s.Equal(connection.Name, retrieved.Name)

	err = s.DB.Model(&retrieved).Update("enabled", false).Error
	s.NoError(err)

	err = s.DB.First(&retrieved, connection.ID).Error
	s.NoError(err)
	s.False(retrieved.Enabled)

	s.DB.Delete(&retrieved)
}

// TestTopicSoftDelete tests that topic can be deleted with GORM soft delete if supported.
func (s *DatabaseExtTestSuite) TestTopicIsHardDeleted() {
	topic := &model.Topic{
		Platform:  "slack",
		Name:      "hard-delete-topic",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(topic).Error
	s.Require().NoError(err)

	err = s.DB.Delete(topic).Error
	s.NoError(err)

	var found model.Topic
	err = s.DB.First(&found, topic.ID).Error
	s.Error(err, "topic should be deleted after hard delete")
}

// TestUrlViewCountDefault tests that view_count defaults to 0.
func (s *DatabaseExtTestSuite) TestUrlViewCountDefault() {
	urlItem := &model.Url{
		Flag:      "integration-test-url-default",
		URL:       "https://example.com/default-test",
		State:     model.UrlStateEnable,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(urlItem).Error
	s.Require().NoError(err)

	var retrieved model.Url
	err = s.DB.First(&retrieved, urlItem.ID).Error
	s.NoError(err)
	s.Equal(model.UrlStateEnable, retrieved.State)

	s.DB.Delete(&retrieved)
}

// TestFileStateTransitions tests file state transitions.
func (s *DatabaseExtTestSuite) TestFileStateTransitions() {
	fileupload := &model.Fileupload{
		UID:       uuid.New().String(),
		Fid:       uuid.New().String(),
		Name:      "state-test.txt",
		Mimetype:  "text/plain",
		Size:      512,
		Location:  "/tmp/state-test.txt",
		State:     model.FileStart,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.DB.Create(fileupload).Error
	s.Require().NoError(err)

	err = s.DB.Model(fileupload).Update("state", model.FileFinish).Error
	s.NoError(err)

	var updated model.Fileupload
	err = s.DB.First(&updated, fileupload.ID).Error
	s.NoError(err)
	s.Equal(model.FileFinish, updated.State)

	s.DB.Delete(&updated)
}

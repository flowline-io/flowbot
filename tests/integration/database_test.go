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

// DatabaseTestSuite tests database operations with Testcontainers.
type DatabaseTestSuite struct {
	IntegrationTestSuite
}

// TestUserCRUD tests user CRUD operations.
func (s *DatabaseTestSuite) TestUserCRUD() {
	// Create a test user
	user := &model.User{
		Flag:      uuid.New().String(),
		Name:      "Integration Test User",
		Tags:      `{"role": "test"}`,
		State:     model.UserActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert using GORM
	err := s.DB.Create(user).Error
	s.Require().NoError(err)
	s.Greater(user.ID, int64(0), "user ID should be set after creation")

	// Query the user
	var retrievedUser model.User
	err = s.DB.First(&retrievedUser, user.ID).Error
	s.NoError(err)
	s.Equal(user.Flag, retrievedUser.Flag)
	s.Equal(user.Name, retrievedUser.Name)

	// Update the user
	err = s.DB.Model(&retrievedUser).Update("name", "Updated Test User").Error
	s.NoError(err)

	// Verify update
	err = s.DB.First(&retrievedUser, user.ID).Error
	s.NoError(err)
	s.Equal("Updated Test User", retrievedUser.Name)

	// Delete the user (soft delete)
	err = s.DB.Delete(&retrievedUser).Error
	s.NoError(err)

	// Verify soft delete
	err = s.DB.First(&retrievedUser, user.ID).Error
	s.Error(err, "user should be soft deleted")
}

// TestBotCRUD tests bot CRUD operations.
func (s *DatabaseTestSuite) TestBotCRUD() {
	bot := &model.Bot{
		Name:      "integration-test-bot",
		State:     model.BotActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create bot
	err := s.DB.Create(bot).Error
	s.Require().NoError(err)
	s.Greater(bot.ID, int64(0))

	// Query bot
	var retrievedBot model.Bot
	err = s.DB.First(&retrievedBot, bot.ID).Error
	s.NoError(err)
	s.Equal(bot.Name, retrievedBot.Name)

	// Update bot
	err = s.DB.Model(&retrievedBot).Update("state", model.BotInactive).Error
	s.NoError(err)

	// Verify update
	err = s.DB.First(&retrievedBot, bot.ID).Error
	s.NoError(err)
	s.Equal(model.BotInactive, retrievedBot.State)

	// Cleanup
	s.DB.Delete(&retrievedBot)
}

// TestPlatformCRUD tests platform CRUD operations.
func (s *DatabaseTestSuite) TestPlatformCRUD() {
	platform := &model.Platform{
		Name:      "integration-test-platform",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create platform
	err := s.DB.Create(platform).Error
	s.Require().NoError(err)
	s.Greater(platform.ID, int64(0))

	// Query platform
	var retrievedPlatform model.Platform
	err = s.DB.First(&retrievedPlatform, platform.ID).Error
	s.NoError(err)
	s.Equal(platform.Name, retrievedPlatform.Name)

	// Cleanup
	s.DB.Delete(&retrievedPlatform)
}

// TestChannelCRUD tests channel CRUD operations.
func (s *DatabaseTestSuite) TestChannelCRUD() {
	channel := &model.Channel{
		Name:      "integration-test-channel",
		Flag:      "integration-test-channel-flag",
		State:     model.ChannelActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create channel
	err := s.DB.Create(channel).Error
	s.Require().NoError(err)
	s.Greater(channel.ID, int64(0))

	// Query channel
	var retrievedChannel model.Channel
	err = s.DB.First(&retrievedChannel, channel.ID).Error
	s.NoError(err)
	s.Equal(channel.Name, retrievedChannel.Name)

	// Cleanup
	s.DB.Delete(&retrievedChannel)
}

// TestMessageCRUD tests message CRUD operations.
func (s *DatabaseTestSuite) TestMessageCRUD() {
	message := &model.Message{
		Flag:          uuid.New().String(),
		PlatformID:    1,
		PlatformMsgID: "test-msg-id",
		Topic:         "test-topic",
		Session:       "test-session",
		State:         model.MessageCreated,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Create message
	err := s.DB.Create(message).Error
	s.Require().NoError(err)
	s.Greater(message.ID, int64(0))

	// Query message
	var retrievedMessage model.Message
	err = s.DB.First(&retrievedMessage, message.ID).Error
	s.NoError(err)
	s.Equal(message.Flag, retrievedMessage.Flag)
	s.Equal(message.Session, retrievedMessage.Session)

	// Cleanup
	s.DB.Delete(&retrievedMessage)
}

// TestWebhookCRUD tests webhook CRUD operations.
func (s *DatabaseTestSuite) TestWebhookCRUD() {
	webhook := &model.Webhook{
		UID:       uuid.New().String(),
		Flag:      "integration-test-webhook",
		Topic:     "test-topic",
		Secret:    uuid.New().String(),
		State:     model.WebhookActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create webhook
	err := s.DB.Create(webhook).Error
	s.Require().NoError(err)
	s.Greater(webhook.ID, int64(0))

	// Query webhook
	var retrievedWebhook model.Webhook
	err = s.DB.First(&retrievedWebhook, webhook.ID).Error
	s.NoError(err)
	s.Equal(webhook.Flag, retrievedWebhook.Flag)
	s.Equal(webhook.Secret, retrievedWebhook.Secret)

	// Cleanup
	s.DB.Delete(&retrievedWebhook)
}

// TestCounterCRUD tests counter CRUD operations.
func (s *DatabaseTestSuite) TestCounterCRUD() {
	counter := &model.Counter{
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Flag:      "integration-test-counter",
		Digit:     100,
		Status:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create counter
	err := s.DB.Create(counter).Error
	s.Require().NoError(err)
	s.Greater(counter.ID, int64(0))

	// Query counter
	var retrievedCounter model.Counter
	err = s.DB.First(&retrievedCounter, counter.ID).Error
	s.NoError(err)
	s.Equal(counter.Digit, retrievedCounter.Digit)

	// Update counter
	err = s.DB.Model(&retrievedCounter).Update("digit", 200).Error
	s.NoError(err)

	// Verify update
	err = s.DB.First(&retrievedCounter, counter.ID).Error
	s.NoError(err)
	s.Equal(int64(200), retrievedCounter.Digit)

	// Cleanup
	s.DB.Delete(&retrievedCounter)
}

// TestDataCRUD tests data CRUD operations.
func (s *DatabaseTestSuite) TestDataCRUD() {
	data := &model.Data{
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Key:       "test-key",
		Value:     model.JSON{"test": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create data
	err := s.DB.Create(data).Error
	s.Require().NoError(err)
	s.Greater(data.ID, int64(0))

	// Query data
	var retrievedData model.Data
	err = s.DB.First(&retrievedData, data.ID).Error
	s.NoError(err)
	s.Equal(data.Key, retrievedData.Key)

	// Cleanup
	s.DB.Delete(&retrievedData)
}

// TestConfigCRUD tests config CRUD operations.
func (s *DatabaseTestSuite) TestConfigCRUD() {
	config := &model.Config{
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Key:       "test-config-key",
		Value:     model.JSON{"setting": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create config
	err := s.DB.Create(config).Error
	s.Require().NoError(err)
	s.Greater(config.ID, int64(0))

	// Query config
	var retrievedConfig model.Config
	err = s.DB.First(&retrievedConfig, config.ID).Error
	s.NoError(err)
	s.Equal(config.Key, retrievedConfig.Key)

	// Cleanup
	s.DB.Delete(&retrievedConfig)
}

// TestFormCRUD tests form CRUD operations.
func (s *DatabaseTestSuite) TestFormCRUD() {
	formID := uuid.New().String()
	form := &model.Form{
		FormID:    formID,
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Schema:    model.JSON{"type": "object"},
		Values:    model.JSON{"field": "value"},
		State:     model.FormStateCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create form
	err := s.DB.Create(form).Error
	s.Require().NoError(err)
	s.Greater(form.ID, int64(0))

	// Query form
	var retrievedForm model.Form
	err = s.DB.Where("form_id = ?", formID).First(&retrievedForm).Error
	s.NoError(err)
	s.Equal(form.FormID, retrievedForm.FormID)

	// Cleanup
	s.DB.Delete(&retrievedForm)
}

// TestPageCRUD tests page CRUD operations.
func (s *DatabaseTestSuite) TestPageCRUD() {
	pageID := uuid.New().String()
	page := &model.Page{
		PageID:    pageID,
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Type:      model.PageForm,
		Schema:    model.JSON{"title": "Test Page"},
		State:     model.PageStateCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create page
	err := s.DB.Create(page).Error
	s.Require().NoError(err)
	s.Greater(page.ID, int64(0))

	// Query page
	var retrievedPage model.Page
	err = s.DB.Where("page_id = ?", pageID).First(&retrievedPage).Error
	s.NoError(err)
	s.Equal(page.PageID, retrievedPage.PageID)

	// Cleanup
	s.DB.Delete(&retrievedPage)
}

// TestBehaviorCRUD tests behavior CRUD operations.
func (s *DatabaseTestSuite) TestBehaviorCRUD() {
	uid := uuid.New().String()
	behavior := &model.Behavior{
		UID:       uid,
		Flag:      "integration-test-behavior",
		Count_:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create behavior
	err := s.DB.Create(behavior).Error
	s.Require().NoError(err)
	s.Greater(behavior.ID, int64(0))

	// Query behavior
	var retrievedBehavior model.Behavior
	err = s.DB.Where("uid = ? AND flag = ?", uid, behavior.Flag).First(&retrievedBehavior).Error
	s.NoError(err)
	s.Equal(behavior.Count_, retrievedBehavior.Count_)

	// Update behavior
	err = s.DB.Model(&retrievedBehavior).Update("count", 10).Error
	s.NoError(err)

	// Verify update
	err = s.DB.Where("uid = ? AND flag = ?", uid, behavior.Flag).First(&retrievedBehavior).Error
	s.NoError(err)
	s.Equal(int32(10), retrievedBehavior.Count_)

	// Cleanup
	s.DB.Delete(&retrievedBehavior)
}

// TestInstructCRUD tests instruct CRUD operations.
func (s *DatabaseTestSuite) TestInstructCRUD() {
	instruct := &model.Instruct{
		No:        uuid.New().String()[:25],
		UID:       uuid.New().String(),
		Object:    model.InstructObjectAgent,
		Bot:       "test-bot",
		Flag:      "test-flag",
		Content:   model.JSON{"action": "test"},
		Priority:  model.InstructPriorityDefault,
		State:     model.InstructCreate,
		ExpireAt:  time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create instruct
	err := s.DB.Create(instruct).Error
	s.Require().NoError(err)
	s.Greater(instruct.ID, int64(0))

	// Query instruct
	var retrievedInstruct model.Instruct
	err = s.DB.First(&retrievedInstruct, instruct.ID).Error
	s.NoError(err)
	s.Equal(instruct.No, retrievedInstruct.No)

	// Update instruct
	err = s.DB.Model(&retrievedInstruct).Update("state", model.InstructDone).Error
	s.NoError(err)

	// Verify update
	err = s.DB.First(&retrievedInstruct, instruct.ID).Error
	s.NoError(err)
	s.Equal(model.InstructDone, retrievedInstruct.State)

	// Cleanup
	s.DB.Delete(&retrievedInstruct)
}

// TestAgentCRUD tests agent CRUD operations.
func (s *DatabaseTestSuite) TestAgentCRUD() {
	agent := &model.Agent{
		UID:            uuid.New().String(),
		Topic:          "test-topic",
		Hostid:         "test-host-id",
		Hostname:       "test-host",
		OnlineDuration: 0,
		LastOnlineAt:   time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create agent
	err := s.DB.Create(agent).Error
	s.Require().NoError(err)
	s.Greater(agent.ID, int64(0))

	// Query agent
	var retrievedAgent model.Agent
	err = s.DB.First(&retrievedAgent, agent.ID).Error
	s.NoError(err)
	s.Equal(agent.Hostid, retrievedAgent.Hostid)
	s.Equal(agent.Hostname, retrievedAgent.Hostname)

	// Cleanup
	s.DB.Delete(&retrievedAgent)
}

// TestTransactionSupport tests database transaction support.
func (s *DatabaseTestSuite) TestTransactionSupport() {
	// Start a transaction
	tx := s.DB.Begin()
	s.Require().NoError(tx.Error)

	// Create user within transaction
	user := &model.User{
		Flag:      uuid.New().String(),
		Name:      "Transaction Test User",
		Tags:      `{}`,
		State:     model.UserActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := tx.Create(user).Error
	s.Require().NoError(err)

	// Commit transaction
	err = tx.Commit().Error
	s.Require().NoError(err)

	// Verify user exists
	var retrievedUser model.User
	err = s.DB.First(&retrievedUser, user.ID).Error
	s.NoError(err)
	s.Equal(user.Flag, retrievedUser.Flag)

	// Cleanup
	s.DB.Delete(&retrievedUser)
}

// TestTransactionRollback tests database transaction rollback.
func (s *DatabaseTestSuite) TestTransactionRollback() {
	flag := uuid.New().String()

	// Start a transaction
	tx := s.DB.Begin()
	s.Require().NoError(tx.Error)

	// Create user within transaction
	user := &model.User{
		Flag:      flag,
		Name:      "Rollback Test User",
		Tags:      `{}`,
		State:     model.UserActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := tx.Create(user).Error
	s.Require().NoError(err)

	// Rollback transaction
	err = tx.Rollback().Error
	s.Require().NoError(err)

	// Verify user does not exist
	var count int64
	s.DB.Model(&model.User{}).Where("flag = ?", flag).Count(&count)
	s.Equal(int64(0), count, "user should not exist after rollback")
}

// DatabaseTestSuite entry point.
func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

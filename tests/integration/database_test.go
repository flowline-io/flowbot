//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/agent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/behavior"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/bot"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/channel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/configdata"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/counter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/data"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/form"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/instruct"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/message"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/page"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platform"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/user"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/webhook"
	"github.com/flowline-io/flowbot/internal/store/model"
)

// DatabaseTestSuite tests database operations with Testcontainers.
type DatabaseTestSuite struct {
	IntegrationTestSuite
}

// TestUserCRUD tests user CRUD operations.
func (s *DatabaseTestSuite) TestUserCRUD() {
	ctx := context.Background()

	flag := uuid.New().String()
	entUser, err := s.EntClient.User.Create().
		SetFlag(flag).
		SetName("Integration Test User").
		SetTags(`{"role": "test"}`).
		SetState(int(model.UserActive)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entUser.ID, int64(0), "user ID should be set after creation")

	// Query the user
	retrievedUser, err := s.EntClient.User.Query().
		Where(user.IDEQ(entUser.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entUser.Flag, retrievedUser.Flag)
	s.Equal(entUser.Name, retrievedUser.Name)

	// Update the user
	err = s.EntClient.User.Update().
		Where(user.IDEQ(retrievedUser.ID)).
		SetName("Updated Test User").
		Exec(ctx)
	s.NoError(err)

	// Verify update
	retrievedUser, err = s.EntClient.User.Query().
		Where(user.IDEQ(entUser.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal("Updated Test User", retrievedUser.Name)

	// Delete the user (Ent hard deletes by default)
	_, err = s.EntClient.User.Delete().
		Where(user.IDEQ(retrievedUser.ID)).
		Exec(ctx)
	s.NoError(err)

	// Verify delete
	_, err = s.EntClient.User.Query().
		Where(user.IDEQ(entUser.ID)).
		Only(ctx)
	s.Error(err, "user should be deleted")
}

// TestBotCRUD tests bot CRUD operations.
func (s *DatabaseTestSuite) TestBotCRUD() {
	ctx := context.Background()

	entBot, err := s.EntClient.Bot.Create().
		SetName("integration-test-bot").
		SetState(int(model.BotActive)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entBot.ID, int64(0))

	// Query bot
	retrievedBot, err := s.EntClient.Bot.Query().
		Where(bot.IDEQ(entBot.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entBot.Name, retrievedBot.Name)

	// Update bot
	err = s.EntClient.Bot.Update().
		Where(bot.IDEQ(retrievedBot.ID)).
		SetState(int(model.BotInactive)).
		Exec(ctx)
	s.NoError(err)

	// Verify update
	retrievedBot, err = s.EntClient.Bot.Query().
		Where(bot.IDEQ(entBot.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int(model.BotInactive), retrievedBot.State)

	// Cleanup
	s.EntClient.Bot.Delete().Where(bot.IDEQ(retrievedBot.ID)).Exec(ctx)
}

// TestPlatformCRUD tests platform CRUD operations.
func (s *DatabaseTestSuite) TestPlatformCRUD() {
	ctx := context.Background()

	entPlatform, err := s.EntClient.Platform.Create().
		SetName("integration-test-platform").
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entPlatform.ID, int64(0))

	// Query platform
	retrievedPlatform, err := s.EntClient.Platform.Query().
		Where(platform.IDEQ(entPlatform.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entPlatform.Name, retrievedPlatform.Name)

	// Cleanup
	s.EntClient.Platform.Delete().Where(platform.IDEQ(retrievedPlatform.ID)).Exec(ctx)
}

// TestChannelCRUD tests channel CRUD operations.
func (s *DatabaseTestSuite) TestChannelCRUD() {
	ctx := context.Background()

	entChannel, err := s.EntClient.Channel.Create().
		SetName("integration-test-channel").
		SetFlag("integration-test-channel-flag").
		SetState(int(model.ChannelActive)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entChannel.ID, int64(0))

	// Query channel
	retrievedChannel, err := s.EntClient.Channel.Query().
		Where(channel.IDEQ(entChannel.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entChannel.Name, retrievedChannel.Name)

	// Cleanup
	s.EntClient.Channel.Delete().Where(channel.IDEQ(retrievedChannel.ID)).Exec(ctx)
}

// TestMessageCRUD tests message CRUD operations.
func (s *DatabaseTestSuite) TestMessageCRUD() {
	ctx := context.Background()

	entMessage, err := s.EntClient.Message.Create().
		SetFlag(uuid.New().String()).
		SetPlatformID(1).
		SetPlatformMsgID("test-msg-id").
		SetTopic("test-topic").
		SetSession("test-session").
		SetState(int(model.MessageCreated)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entMessage.ID, int64(0))

	// Query message
	retrievedMessage, err := s.EntClient.Message.Query().
		Where(message.IDEQ(entMessage.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entMessage.Flag, retrievedMessage.Flag)
	s.Equal(entMessage.Session, retrievedMessage.Session)

	// Cleanup
	s.EntClient.Message.Delete().Where(message.IDEQ(retrievedMessage.ID)).Exec(ctx)
}

// TestWebhookCRUD tests webhook CRUD operations.
func (s *DatabaseTestSuite) TestWebhookCRUD() {
	ctx := context.Background()

	entWebhook, err := s.EntClient.Webhook.Create().
		SetUID(uuid.New().String()).
		SetFlag("integration-test-webhook").
		SetTopic("test-topic").
		SetSecret(uuid.New().String()).
		SetState(int(model.WebhookActive)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entWebhook.ID, int64(0))

	// Query webhook
	retrievedWebhook, err := s.EntClient.Webhook.Query().
		Where(webhook.IDEQ(entWebhook.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entWebhook.Flag, retrievedWebhook.Flag)
	s.Equal(entWebhook.Secret, retrievedWebhook.Secret)

	// Cleanup
	s.EntClient.Webhook.Delete().Where(webhook.IDEQ(retrievedWebhook.ID)).Exec(ctx)
}

// TestCounterCRUD tests counter CRUD operations.
func (s *DatabaseTestSuite) TestCounterCRUD() {
	ctx := context.Background()

	entCounter, err := s.EntClient.Counter.Create().
		SetUID(uuid.New().String()).
		SetTopic("test-topic").
		SetFlag("integration-test-counter").
		SetDigit(100).
		SetStatus(1).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entCounter.ID, int64(0))

	// Query counter
	retrievedCounter, err := s.EntClient.Counter.Query().
		Where(counter.IDEQ(entCounter.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entCounter.Digit, retrievedCounter.Digit)

	// Update counter
	err = s.EntClient.Counter.Update().
		Where(counter.IDEQ(retrievedCounter.ID)).
		SetDigit(200).
		Exec(ctx)
	s.NoError(err)

	// Verify update
	retrievedCounter, err = s.EntClient.Counter.Query().
		Where(counter.IDEQ(entCounter.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int64(200), retrievedCounter.Digit)

	// Cleanup
	s.EntClient.Counter.Delete().Where(counter.IDEQ(retrievedCounter.ID)).Exec(ctx)
}

// TestDataCRUD tests data CRUD operations.
func (s *DatabaseTestSuite) TestDataCRUD() {
	ctx := context.Background()

	entData, err := s.EntClient.Data.Create().
		SetUID(uuid.New().String()).
		SetTopic("test-topic").
		SetKey("test-key").
		SetValue(model.JSON{"test": "value"}).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entData.ID, int64(0))

	// Query data
	retrievedData, err := s.EntClient.Data.Query().
		Where(data.IDEQ(entData.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entData.Key, retrievedData.Key)

	// Cleanup
	s.EntClient.Data.Delete().Where(data.IDEQ(retrievedData.ID)).Exec(ctx)
}

// TestConfigCRUD tests config CRUD operations.
func (s *DatabaseTestSuite) TestConfigCRUD() {
	ctx := context.Background()

	entConfig, err := s.EntClient.ConfigData.Create().
		SetUID(uuid.New().String()).
		SetTopic("test-topic").
		SetKey("test-config-key").
		SetValue(model.JSON{"setting": "value"}).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entConfig.ID, int64(0))

	// Query config
	retrievedConfig, err := s.EntClient.ConfigData.Query().
		Where(configdata.IDEQ(entConfig.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entConfig.Key, retrievedConfig.Key)

	// Cleanup
	s.EntClient.ConfigData.Delete().Where(configdata.IDEQ(retrievedConfig.ID)).Exec(ctx)
}

// TestFormCRUD tests form CRUD operations.
func (s *DatabaseTestSuite) TestFormCRUD() {
	ctx := context.Background()

	formID := uuid.New().String()
	entForm, err := s.EntClient.Form.Create().
		SetFormID(formID).
		SetUID(uuid.New().String()).
		SetTopic("test-topic").
		SetSchema(model.JSON{"type": "object"}).
		SetValues(model.JSON{"field": "value"}).
		SetState(int(model.FormStateCreated)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entForm.ID, int64(0))

	// Query form
	retrievedForm, err := s.EntClient.Form.Query().
		Where(form.FormIDEQ(formID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entForm.FormID, retrievedForm.FormID)

	// Cleanup
	s.EntClient.Form.Delete().Where(form.IDEQ(retrievedForm.ID)).Exec(ctx)
}

// TestPageCRUD tests page CRUD operations.
func (s *DatabaseTestSuite) TestPageCRUD() {
	ctx := context.Background()

	pageID := uuid.New().String()
	entPage, err := s.EntClient.Page.Create().
		SetPageID(pageID).
		SetUID(uuid.New().String()).
		SetTopic("test-topic").
		SetType(string(model.PageForm)).
		SetSchema(model.JSON{"title": "Test Page"}).
		SetState(int(model.PageStateCreated)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entPage.ID, int64(0))

	// Query page
	retrievedPage, err := s.EntClient.Page.Query().
		Where(page.PageIDEQ(pageID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entPage.PageID, retrievedPage.PageID)

	// Cleanup
	s.EntClient.Page.Delete().Where(page.IDEQ(retrievedPage.ID)).Exec(ctx)
}

// TestBehaviorCRUD tests behavior CRUD operations.
func (s *DatabaseTestSuite) TestBehaviorCRUD() {
	ctx := context.Background()

	uid := uuid.New().String()
	entBehavior, err := s.EntClient.Behavior.Create().
		SetUID(uid).
		SetFlag("integration-test-behavior").
		SetCount(int32(1)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entBehavior.ID, int64(0))

	// Query behavior
	retrievedBehavior, err := s.EntClient.Behavior.Query().
		Where(behavior.UIDEQ(uid), behavior.FlagEQ(entBehavior.Flag)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entBehavior.Count, retrievedBehavior.Count)

	// Update behavior
	err = s.EntClient.Behavior.Update().
		Where(behavior.UIDEQ(uid), behavior.FlagEQ(entBehavior.Flag)).
		SetCount(int32(10)).
		Exec(ctx)
	s.NoError(err)

	// Verify update
	retrievedBehavior, err = s.EntClient.Behavior.Query().
		Where(behavior.UIDEQ(uid), behavior.FlagEQ(entBehavior.Flag)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int32(10), retrievedBehavior.Count)

	// Cleanup
	s.EntClient.Behavior.Delete().Where(behavior.IDEQ(retrievedBehavior.ID)).Exec(ctx)
}

// TestInstructCRUD tests instruct CRUD operations.
func (s *DatabaseTestSuite) TestInstructCRUD() {
	ctx := context.Background()

	entInstruct, err := s.EntClient.Instruct.Create().
		SetNo(uuid.New().String()[:25]).
		SetUID(uuid.New().String()).
		SetObject(string(model.InstructObjectAgent)).
		SetBot("test-bot").
		SetFlag("test-flag").
		SetContent(model.JSON{"action": "test"}).
		SetPriority(int(model.InstructPriorityDefault)).
		SetState(int(model.InstructCreate)).
		SetExpireAt(time.Now().Add(time.Hour)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entInstruct.ID, int64(0))

	// Query instruct
	retrievedInstruct, err := s.EntClient.Instruct.Query().
		Where(instruct.IDEQ(entInstruct.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entInstruct.No, retrievedInstruct.No)

	// Update instruct
	err = s.EntClient.Instruct.Update().
		Where(instruct.IDEQ(retrievedInstruct.ID)).
		SetState(int(model.InstructDone)).
		Exec(ctx)
	s.NoError(err)

	// Verify update
	retrievedInstruct, err = s.EntClient.Instruct.Query().
		Where(instruct.IDEQ(entInstruct.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(int(model.InstructDone), retrievedInstruct.State)

	// Cleanup
	s.EntClient.Instruct.Delete().Where(instruct.IDEQ(retrievedInstruct.ID)).Exec(ctx)
}

// TestAgentCRUD tests agent CRUD operations.
func (s *DatabaseTestSuite) TestAgentCRUD() {
	ctx := context.Background()

	entAgent, err := s.EntClient.Agent.Create().
		SetUID(uuid.New().String()).
		SetTopic("test-topic").
		SetHostid("test-host-id").
		SetHostname("test-host").
		SetOnlineDuration(0).
		SetLastOnlineAt(time.Now()).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)
	s.Greater(entAgent.ID, int64(0))

	// Query agent
	retrievedAgent, err := s.EntClient.Agent.Query().
		Where(agent.IDEQ(entAgent.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entAgent.Hostid, retrievedAgent.Hostid)
	s.Equal(entAgent.Hostname, retrievedAgent.Hostname)

	// Cleanup
	s.EntClient.Agent.Delete().Where(agent.IDEQ(retrievedAgent.ID)).Exec(ctx)
}

// TestTransactionSupport tests database transaction support.
func (s *DatabaseTestSuite) TestTransactionSupport() {
	ctx := context.Background()

	tx, err := s.EntClient.Tx(ctx)
	s.Require().NoError(err)

	flag := uuid.New().String()
	entUser, err := tx.User.Create().
		SetFlag(flag).
		SetName("Transaction Test User").
		SetTags(`{}`).
		SetState(int(model.UserActive)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)

	err = tx.Commit()
	s.Require().NoError(err)

	// Verify user exists
	retrievedUser, err := s.EntClient.User.Query().
		Where(user.IDEQ(entUser.ID)).
		Only(ctx)
	s.NoError(err)
	s.Equal(entUser.Flag, retrievedUser.Flag)

	// Cleanup
	s.EntClient.User.Delete().Where(user.IDEQ(retrievedUser.ID)).Exec(ctx)
}

// TestTransactionRollback tests database transaction rollback.
func (s *DatabaseTestSuite) TestTransactionRollback() {
	ctx := context.Background()

	flag := uuid.New().String()

	tx, err := s.EntClient.Tx(ctx)
	s.Require().NoError(err)

	_, err = tx.User.Create().
		SetFlag(flag).
		SetName("Rollback Test User").
		SetTags(`{}`).
		SetState(int(model.UserActive)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	s.Require().NoError(err)

	err = tx.Rollback()
	s.Require().NoError(err)

	// Verify user does not exist
	count, err := s.EntClient.User.Query().
		Where(user.FlagEQ(flag)).
		Count(ctx)
	s.NoError(err)
	s.Equal(0, count, "user should not exist after rollback")
}

// DatabaseTestSuite entry point.
func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

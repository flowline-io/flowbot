package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	storeMigrate "github.com/flowline-io/flowbot/pkg/migrate"
	"github.com/flowline-io/flowbot/pkg/types"
)

// AdapterTestSuite is a test suite for the MySQL adapter using testcontainers
type AdapterTestSuite struct {
	suite.Suite
	ctx       context.Context
	container *tcmysql.MySQLContainer
	adapter   *adapter
}

// SetupSuite initializes the test suite with a MySQL container
func (s *AdapterTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Start MySQL container
	container, err := tcmysql.Run(s.ctx,
		"mysql:8.0",
		tcmysql.WithDatabase("flowbot"),
		tcmysql.WithUsername("root"),
		tcmysql.WithPassword("root"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").
				WithStartupTimeout(60*time.Second),
		),
	)
	s.Require().NoError(err, "failed to start MySQL container")
	s.container = container

	// Get connection string
	connStr, err := container.ConnectionString(s.ctx)
	s.Require().NoError(err, "failed to get connection string")

	// Create adapter
	s.adapter = &adapter{}

	// Configure and open adapter
	storeConfig := config.StoreType{
		UseAdapter: "mysql",
		Adapters: map[string]any{
			"mysql": map[string]any{
				"dsn":               connStr + "?charset=utf8mb4&parseTime=True&loc=Local",
				"max_open_conns":    10,
				"max_idle_conns":    5,
				"conn_max_lifetime": 3600,
				"sql_timeout":       30,
			},
		},
	}

	err = s.adapter.Open(storeConfig)
	s.Require().NoError(err, "failed to open adapter")

	// Run migrations
	err = s.runMigrations("")
	s.Require().NoError(err, "failed to run migrations")
}

// TearDownSuite cleans up the test suite
func (s *AdapterTestSuite) TearDownSuite() {
	if s.adapter != nil && s.adapter.IsOpen() {
		err := s.adapter.Close()
		s.Require().NoError(err, "failed to close adapter")
	}

	if s.container != nil {
		err := s.container.Terminate(s.ctx)
		s.Require().NoError(err, "failed to terminate container")
	}
}

// runMigrations runs the database migrations
func (s *AdapterTestSuite) runMigrations(_ string) error {
	db, err := s.adapter.GetDB().DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB: %w", err)
	}

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	d, err := iofs.New(storeMigrate.Fs, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "mysql", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// TestAdapter_Open tests opening the adapter
func (s *AdapterTestSuite) TestAdapter_Open() {
	s.True(s.adapter.IsOpen(), "adapter should be open")
	s.Equal("mysql", s.adapter.GetName(), "adapter name should be mysql")
	s.NotNil(s.adapter.GetDB(), "DB should not be nil")
}

// TestAdapter_UserOperations tests user CRUD operations
func (s *AdapterTestSuite) TestAdapter_UserOperations() {
	// Create user
	user := &model.User{
		Flag:      uuid.New().String(),
		Name:      "Test User",
		Tags:      `{"role": "admin"}`,
		State:     model.UserActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.adapter.UserCreate(user)
	s.Require().NoError(err, "failed to create user")
	s.Positive(user.ID, "user ID should be set")

	// Get user by UID
	uid := types.Uid(user.Flag)
	retrievedUser, err := s.adapter.UserGet(uid)
	s.Require().NoError(err, "failed to get user")
	s.Equal(user.Flag, retrievedUser.Flag, "flag should match")
	s.Equal(user.Name, retrievedUser.Name, "name should match")

	// Get user by ID
	userByID, err := s.adapter.GetUserById(user.ID)
	s.Require().NoError(err, "failed to get user by ID")
	s.Equal(user.Flag, userByID.Flag, "flag should match")

	// Get user by flag
	userByFlag, err := s.adapter.GetUserByFlag(user.Flag)
	s.Require().NoError(err, "failed to get user by flag")
	s.Equal(user.Name, userByFlag.Name, "name should match")

	// Update user
	update := types.KV{
		"name": "Updated User",
	}
	err = s.adapter.UserUpdate(uid, update)
	s.Require().NoError(err, "failed to update user")

	// Verify update
	updatedUser, err := s.adapter.UserGet(uid)
	s.Require().NoError(err, "failed to get updated user")
	s.Equal("Updated User", updatedUser.Name, "name should be updated")

	// Get all users
	users, err := s.adapter.GetUsers()
	s.Require().NoError(err, "failed to get users")
	s.GreaterOrEqual(len(users), 1, "should have at least one user")

	// Get first user
	firstUser, err := s.adapter.FirstUser()
	s.Require().NoError(err, "failed to get first user")
	s.NotNil(firstUser, "first user should not be nil")

	// Delete user
	err = s.adapter.UserDelete(uid, false)
	s.Require().NoError(err, "failed to delete user")

	// Verify deletion
	_, err = s.adapter.UserGet(uid)
	s.Error(err, "user should be deleted")
}

// TestAdapter_BotOperations tests bot CRUD operations
func (s *AdapterTestSuite) TestAdapter_BotOperations() {
	// Create bot
	bot := &model.Bot{
		Name:      "test-bot",
		State:     model.BotActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	botID, err := s.adapter.CreateBot(bot)
	s.Require().NoError(err, "failed to create bot")
	s.Positive(botID, "bot ID should be set")

	// Get bot by ID
	retrievedBot, err := s.adapter.GetBot(botID)
	s.Require().NoError(err, "failed to get bot")
	s.Equal(bot.Name, retrievedBot.Name, "name should match")

	// Get bot by name
	botByName, err := s.adapter.GetBotByName(bot.Name)
	s.Require().NoError(err, "failed to get bot by name")
	s.Equal(botID, botByName.ID, "ID should match")

	// Get all bots
	bots, err := s.adapter.GetBots()
	s.Require().NoError(err, "failed to get bots")
	s.GreaterOrEqual(len(bots), 1, "should have at least one bot")

	// Update bot
	bot.State = model.BotInactive
	err = s.adapter.UpdateBot(bot)
	s.Require().NoError(err, "failed to update bot")

	// Verify update
	updatedBot, err := s.adapter.GetBot(botID)
	s.Require().NoError(err, "failed to get updated bot")
	s.Equal(model.BotInactive, updatedBot.State, "state should be updated")

	// Delete bot
	err = s.adapter.DeleteBot(bot.Name)
	s.Require().NoError(err, "failed to delete bot")

	// Verify deletion
	_, err = s.adapter.GetBotByName(bot.Name)
	s.Error(err, "bot should be deleted")
}

// TestAdapter_PlatformOperations tests platform CRUD operations
func (s *AdapterTestSuite) TestAdapter_PlatformOperations() {
	// Create platform
	platform := &model.Platform{
		Name:      "test-platform",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	platformID, err := s.adapter.CreatePlatform(platform)
	s.Require().NoError(err, "failed to create platform")
	s.Positive(platformID, "platform ID should be set")

	// Get platform by ID
	retrievedPlatform, err := s.adapter.GetPlatform(platformID)
	s.Require().NoError(err, "failed to get platform")
	s.Equal(platform.Name, retrievedPlatform.Name, "name should match")

	// Get platform by name
	platformByName, err := s.adapter.GetPlatformByName(platform.Name)
	s.Require().NoError(err, "failed to get platform by name")
	s.Equal(platformID, platformByName.ID, "ID should match")

	// Get all platforms
	platforms, err := s.adapter.GetPlatforms()
	s.Require().NoError(err, "failed to get platforms")
	s.GreaterOrEqual(len(platforms), 1, "should have at least one platform")
}

// TestAdapter_ChannelOperations tests channel CRUD operations
func (s *AdapterTestSuite) TestAdapter_ChannelOperations() {
	// Create channel
	channel := &model.Channel{
		Name:      "test-channel",
		State:     model.ChannelActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	channelID, err := s.adapter.CreateChannel(channel)
	s.Require().NoError(err, "failed to create channel")
	s.Positive(channelID, "channel ID should be set")

	// Get channel by ID
	retrievedChannel, err := s.adapter.GetChannel(channelID)
	s.Require().NoError(err, "failed to get channel")
	s.Equal(channel.Name, retrievedChannel.Name, "name should match")

	// Get channel by name
	channelByName, err := s.adapter.GetChannelByName(channel.Name)
	s.Require().NoError(err, "failed to get channel by name")
	s.Equal(channelID, channelByName.ID, "ID should match")

	// Get all channels
	channels, err := s.adapter.GetChannels()
	s.Require().NoError(err, "failed to get channels")
	s.GreaterOrEqual(len(channels), 1, "should have at least one channel")

	// Update channel
	channel.State = model.ChannelInactive
	err = s.adapter.UpdateChannel(channel)
	s.Require().NoError(err, "failed to update channel")

	// Verify update
	updatedChannel, err := s.adapter.GetChannel(channelID)
	s.Require().NoError(err, "failed to get updated channel")
	s.Equal(model.ChannelInactive, updatedChannel.State, "state should be updated")

	// Delete channel
	err = s.adapter.DeleteChannel(channel.Name)
	s.Require().NoError(err, "failed to delete channel")

	// Verify deletion
	_, err = s.adapter.GetChannelByName(channel.Name)
	s.Error(err, "channel should be deleted")
}

// TestAdapter_PlatformUserOperations tests platform user CRUD operations
func (s *AdapterTestSuite) TestAdapter_PlatformUserOperations() {
	// First create a user
	user := &model.User{
		Flag:      uuid.New().String(),
		Name:      "Platform Test User",
		Tags:      "{}",
		State:     model.UserActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.adapter.UserCreate(user)
	s.Require().NoError(err, "failed to create user")

	// Create platform user
	platformUser := &model.PlatformUser{
		UserID:     user.ID,
		PlatformID: 1,
		Flag:       "platform-user-flag",
		Name:       "Platform User",
		Email:      "test@example.com",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	platformUserID, err := s.adapter.CreatePlatformUser(platformUser)
	s.Require().NoError(err, "failed to create platform user")
	s.Positive(platformUserID, "platform user ID should be set")

	// Get platform user by flag
	retrievedPlatformUser, err := s.adapter.GetPlatformUserByFlag(platformUser.Flag)
	s.Require().NoError(err, "failed to get platform user")
	s.Equal(platformUser.UserID, retrievedPlatformUser.UserID, "user ID should match")

	// Get platform users by user ID
	platformUsers, err := s.adapter.GetPlatformUsersByUserId(user.ID)
	s.Require().NoError(err, "failed to get platform users by user ID")
	s.Len(platformUsers, 1, "should have one platform user")

	// Update platform user
	platformUser.Name = "Updated Platform User"
	err = s.adapter.UpdatePlatformUser(platformUser)
	s.Require().NoError(err, "failed to update platform user")
}

// TestAdapter_PlatformChannelOperations tests platform channel CRUD operations
func (s *AdapterTestSuite) TestAdapter_PlatformChannelOperations() {
	// Create platform channel
	platformChannel := &model.PlatformChannel{
		Flag:       "platform-channel-flag",
		PlatformID: 1,
		ChannelID:  1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	platformChannelID, err := s.adapter.CreatePlatformChannel(platformChannel)
	s.Require().NoError(err, "failed to create platform channel")
	s.Positive(platformChannelID, "platform channel ID should be set")

	// Get platform channel by flag
	retrievedChannel, err := s.adapter.GetPlatformChannelByFlag(platformChannel.Flag)
	s.Require().NoError(err, "failed to get platform channel")
	s.Equal(platformChannel.PlatformID, retrievedChannel.PlatformID, "platform ID should match")

	// Get platform channels by platform IDs
	channels, err := s.adapter.GetPlatformChannelsByPlatformIds([]int64{1})
	s.Require().NoError(err, "failed to get platform channels by platform IDs")
	s.GreaterOrEqual(len(channels), 1, "should have at least one channel")

	// Get platform channel by channel ID
	channelByID, err := s.adapter.GetPlatformChannelsByChannelId(1)
	s.Require().NoError(err, "failed to get platform channel by channel ID")
	s.Equal(platformChannel.Flag, channelByID.Flag, "flag should match")
}

// TestAdapter_PlatformChannelUserOperations tests platform channel user CRUD operations
func (s *AdapterTestSuite) TestAdapter_PlatformChannelUserOperations() {
	// Create platform channel user
	platformChannelUser := &model.PlatformChannelUser{
		UserFlag:    "user-flag-123",
		ChannelFlag: "channel-flag-456",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	platformChannelUserID, err := s.adapter.CreatePlatformChannelUser(platformChannelUser)
	s.Require().NoError(err, "failed to create platform channel user")
	s.Positive(platformChannelUserID, "platform channel user ID should be set")

	// Get platform channel users by user flag
	users, err := s.adapter.GetPlatformChannelUsersByUserFlag(platformChannelUser.UserFlag)
	s.Require().NoError(err, "failed to get platform channel users")
	s.Len(users, 1, "should have one platform channel user")
}

// TestAdapter_MessageOperations tests message CRUD operations
func (s *AdapterTestSuite) TestAdapter_MessageOperations() {
	// Create message
	message := model.Message{
		Flag:          uuid.New().String(),
		PlatformID:    1,
		PlatformMsgID: "msg-123",
		Session:       "session-456",
		State:         model.MessageCreated,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := s.adapter.CreateMessage(message)
	s.Require().NoError(err, "failed to create message")

	// Get message by flag
	retrievedMessage, err := s.adapter.GetMessage(message.Flag)
	s.Require().NoError(err, "failed to get message")
	s.Equal(message.Session, retrievedMessage.Session, "session should match")

	// Get message by platform
	messageByPlatform, err := s.adapter.GetMessageByPlatform(message.PlatformID, message.PlatformMsgID)
	s.Require().NoError(err, "failed to get message by platform")
	s.Equal(message.Flag, messageByPlatform.Flag, "flag should match")

	// Get messages by session
	messages, err := s.adapter.GetMessagesBySession(message.Session)
	s.Require().NoError(err, "failed to get messages by session")
	s.Len(messages, 1, "should have one message")
}

// TestAdapter_DataOperations tests data CRUD operations
func (s *AdapterTestSuite) TestAdapter_DataOperations() {
	uid := types.Uid(uuid.New().String())
	topic := "test-topic"
	key := "test-key"
	value := types.KV{
		"field1": "value1",
		"field2": 42,
	}

	// Set data
	err := s.adapter.DataSet(uid, topic, key, value)
	s.Require().NoError(err, "failed to set data")

	// Get data
	retrievedValue, err := s.adapter.DataGet(uid, topic, key)
	s.Require().NoError(err, "failed to get data")
	s.Equal(value["field1"], retrievedValue["field1"], "field1 should match")
	// JSON unmarshaling converts numbers to float64
	s.InEpsilon(float64(42), retrievedValue["field2"], 0.001, "field2 should match")

	// List data
	filter := types.DataFilter{}
	dataList, err := s.adapter.DataList(uid, topic, filter)
	s.Require().NoError(err, "failed to list data")
	s.Len(dataList, 1, "should have one data entry")

	// Delete data
	err = s.adapter.DataDelete(uid, topic, key)
	s.Require().NoError(err, "failed to delete data")

	// Verify deletion
	_, err = s.adapter.DataGet(uid, topic, key)
	s.Error(err, "data should be deleted")
}

// TestAdapter_ConfigOperations tests config CRUD operations
func (s *AdapterTestSuite) TestAdapter_ConfigOperations() {
	uid := types.Uid(uuid.New().String())
	topic := "test-config-topic"
	key := "test-config-key"
	value := types.KV{
		"setting1": "value1",
		"setting2": true,
	}

	// Set config
	err := s.adapter.ConfigSet(uid, topic, key, value)
	s.Require().NoError(err, "failed to set config")

	// Get config
	retrievedValue, err := s.adapter.ConfigGet(uid, topic, key)
	s.Require().NoError(err, "failed to get config")
	s.Equal(value["setting1"], retrievedValue["setting1"], "setting1 should match")

	// List config by prefix
	configs, err := s.adapter.ListConfigByPrefix(uid, topic, "test")
	s.Require().NoError(err, "failed to list configs")
	s.GreaterOrEqual(len(configs), 1, "should have at least one config")

	// Delete config
	err = s.adapter.ConfigDelete(uid, topic, key)
	s.Require().NoError(err, "failed to delete config")

	// Verify deletion
	_, err = s.adapter.ConfigGet(uid, topic, key)
	s.Error(err, "config should be deleted")
}

// TestAdapter_OAuthOperations tests OAuth CRUD operations
func (s *AdapterTestSuite) TestAdapter_OAuthOperations() {
	uid := types.Uid(uuid.New().String())
	oauth := model.OAuth{
		UID:       string(uid),
		Topic:     "test-topic",
		Type:      "github",
		Token:     "test-token",
		Extra:     model.JSON{"scope": "read"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set OAuth
	err := s.adapter.OAuthSet(oauth)
	s.Require().NoError(err, "failed to set OAuth")

	// Get OAuth
	retrievedOAuth, err := s.adapter.OAuthGet(uid, oauth.Topic, oauth.Type)
	s.Require().NoError(err, "failed to get OAuth")
	s.Equal(oauth.Token, retrievedOAuth.Token, "token should match")

	// Get available OAuth
	availableOAuth, err := s.adapter.OAuthGetAvailable(oauth.Type)
	s.Require().NoError(err, "failed to get available OAuth")
	s.GreaterOrEqual(len(availableOAuth), 1, "should have at least one OAuth")
}

// TestAdapter_FormOperations tests form CRUD operations
func (s *AdapterTestSuite) TestAdapter_FormOperations() {
	formID := uuid.New().String()
	form := model.Form{
		FormID:    formID,
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Schema:    model.JSON{"type": "object"},
		Values:    model.JSON{"field1": "value1"},
		State:     model.FormStateCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set form
	err := s.adapter.FormSet(formID, form)
	s.Require().NoError(err, "failed to set form")

	// Get form
	retrievedForm, err := s.adapter.FormGet(formID)
	s.Require().NoError(err, "failed to get form")
	s.Equal(form.UID, retrievedForm.UID, "UID should match")
	s.Equal(form.State, retrievedForm.State, "state should match")
}

// TestAdapter_PageOperations tests page CRUD operations
func (s *AdapterTestSuite) TestAdapter_PageOperations() {
	pageID := uuid.New().String()
	page := model.Page{
		PageID:    pageID,
		UID:       uuid.New().String(),
		Topic:     "test-topic",
		Type:      model.PageForm,
		Schema:    model.JSON{"title": "Test Page"},
		State:     model.PageStateCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set page
	err := s.adapter.PageSet(pageID, page)
	s.Require().NoError(err, "failed to set page")

	// Get page
	retrievedPage, err := s.adapter.PageGet(pageID)
	s.Require().NoError(err, "failed to get page")
	s.Equal(page.UID, retrievedPage.UID, "UID should match")
	s.Equal(page.State, retrievedPage.State, "state should match")
}

// TestAdapter_BehaviorOperations tests behavior CRUD operations
func (s *AdapterTestSuite) TestAdapter_BehaviorOperations() {
	uid := types.Uid(uuid.New().String())
	flag := "test-behavior"
	behavior := model.Behavior{
		UID:       string(uid),
		Flag:      flag,
		Count_:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set behavior
	err := s.adapter.BehaviorSet(behavior)
	s.Require().NoError(err, "failed to set behavior")

	// Get behavior
	retrievedBehavior, err := s.adapter.BehaviorGet(uid, flag)
	s.Require().NoError(err, "failed to get behavior")
	s.Equal(behavior.Count_, retrievedBehavior.Count_, "count should match")

	// List behaviors
	behaviors, err := s.adapter.BehaviorList(uid)
	s.Require().NoError(err, "failed to list behaviors")
	s.Len(behaviors, 1, "should have one behavior")

	// Increase behavior count
	err = s.adapter.BehaviorIncrease(uid, flag, 5)
	s.Require().NoError(err, "failed to increase behavior count")

	// Verify increase
	updatedBehavior, err := s.adapter.BehaviorGet(uid, flag)
	s.Require().NoError(err, "failed to get updated behavior")
	s.Equal(int32(6), updatedBehavior.Count_, "count should be increased by 5")
}

// TestAdapter_ParameterOperations tests parameter CRUD operations
func (s *AdapterTestSuite) TestAdapter_ParameterOperations() {
	flag := "test-parameter"
	params := types.KV{
		"key1": "value1",
		"key2": 42,
	}
	expiredAt := time.Now().Add(time.Hour)

	// Set parameter
	err := s.adapter.ParameterSet(flag, params, expiredAt)
	s.Require().NoError(err, "failed to set parameter")

	// Get parameter
	retrievedParam, err := s.adapter.ParameterGet(flag)
	s.Require().NoError(err, "failed to get parameter")
	s.False(retrievedParam.IsExpired(), "parameter should not be expired")

	// Delete parameter
	err = s.adapter.ParameterDelete(flag)
	s.Require().NoError(err, "failed to delete parameter")

	// Verify deletion
	_, err = s.adapter.ParameterGet(flag)
	s.Error(err, "parameter should be deleted")
}

// TestAdapter_InstructOperations tests instruct CRUD operations
func (s *AdapterTestSuite) TestAdapter_InstructOperations() {
	uid := types.Uid(uuid.New().String())
	instruct := &model.Instruct{
		No:        fmt.Sprintf("NO-%d", time.Now().UnixNano()%1000000000),
		UID:       string(uid),
		Object:    model.InstructObjectAgent,
		Bot:       "test-bot",
		Flag:      "test-flag",
		Content:   model.JSON{"action": "test-action", "params": "value1"},
		Priority:  model.InstructPriorityDefault,
		State:     model.InstructCreate,
		ExpireAt:  time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create instruct
	instructID, err := s.adapter.CreateInstruct(instruct)
	s.Require().NoError(err, "failed to create instruct")
	s.Positive(instructID, "instruct ID should be set")

	// List instructs (not expired)
	instructs, err := s.adapter.ListInstruct(uid, false, 10)
	s.Require().NoError(err, "failed to list instructs")
	s.Len(instructs, 1, "should have one instruct")

	// Update instruct
	instruct.State = model.InstructDone
	err = s.adapter.UpdateInstruct(instruct)
	s.Require().NoError(err, "failed to update instruct")
}

// TestAdapter_WebhookOperations tests webhook CRUD operations
func (s *AdapterTestSuite) TestAdapter_WebhookOperations() {
	uid := types.Uid(uuid.New().String())
	webhook := &model.Webhook{
		UID:          string(uid),
		Flag:         "test-webhook",
		Topic:        "test-topic",
		Secret:       uuid.New().String(),
		State:        model.WebhookActive,
		TriggerCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create webhook
	webhookID, err := s.adapter.CreateWebhook(webhook)
	s.Require().NoError(err, "failed to create webhook")
	s.Positive(webhookID, "webhook ID should be set")

	// List webhooks
	webhooks, err := s.adapter.ListWebhook(uid)
	s.Require().NoError(err, "failed to list webhooks")
	s.Len(webhooks, 1, "should have one webhook")

	// Get webhook by secret
	webhookBySecret, err := s.adapter.GetWebhookBySecret(webhook.Secret)
	s.Require().NoError(err, "failed to get webhook by secret")
	s.Equal(webhook.Flag, webhookBySecret.Flag, "flag should match")

	// Get webhook by UID and flag
	webhookByUidAndFlag, err := s.adapter.GetWebhookByUidAndFlag(uid, webhook.Flag)
	s.Require().NoError(err, "failed to get webhook by UID and flag")
	s.Equal(webhook.Topic, webhookByUidAndFlag.Topic, "topic should match")

	// Increase webhook count
	err = s.adapter.IncreaseWebhookCount(webhookID)
	s.Require().NoError(err, "failed to increase webhook count")

	// Update webhook
	webhook.State = model.WebhookInactive
	err = s.adapter.UpdateWebhook(webhook)
	s.Require().NoError(err, "failed to update webhook")

	// Delete webhook
	err = s.adapter.DeleteWebhook(webhookID)
	s.Require().NoError(err, "failed to delete webhook")
}

// TestAdapter_CounterOperations tests counter CRUD operations
func (s *AdapterTestSuite) TestAdapter_CounterOperations() {
	uid := types.Uid(uuid.New().String())
	topic := "test-counter-topic"
	counter := &model.Counter{
		UID:       string(uid),
		Topic:     topic,
		Flag:      "test-counter",
		Digit:     100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create counter
	counterID, err := s.adapter.CreateCounter(counter)
	s.Require().NoError(err, "failed to create counter")
	s.Positive(counterID, "counter ID should be set")

	// Get counter
	retrievedCounter, err := s.adapter.GetCounter(counterID)
	s.Require().NoError(err, "failed to get counter")
	s.Equal(counter.Digit, retrievedCounter.Digit, "digit should match")

	// Get counter by flag
	counterByFlag, err := s.adapter.GetCounterByFlag(uid, topic, counter.Flag)
	s.Require().NoError(err, "failed to get counter by flag")
	s.Equal(counter.Digit, counterByFlag.Digit, "digit should match")

	// List counters
	counters, err := s.adapter.ListCounter(uid, topic)
	s.Require().NoError(err, "failed to list counters")
	s.Len(counters, 1, "should have one counter")

	// Increase counter
	err = s.adapter.IncreaseCounter(counterID, 50)
	s.Require().NoError(err, "failed to increase counter")

	// Verify increase
	updatedCounter, err := s.adapter.GetCounter(counterID)
	s.Require().NoError(err, "failed to get updated counter")
	s.Equal(int64(150), updatedCounter.Digit, "digit should be 150")

	// Decrease counter
	err = s.adapter.DecreaseCounter(counterID, 30)
	s.Require().NoError(err, "failed to decrease counter")

	// Verify decrease
	decreasedCounter, err := s.adapter.GetCounter(counterID)
	s.Require().NoError(err, "failed to get decreased counter")
	s.Equal(int64(120), decreasedCounter.Digit, "digit should be 120")
}

// TestAdapter_AgentOperations tests agent CRUD operations
func (s *AdapterTestSuite) TestAdapter_AgentOperations() {
	uid := types.Uid(uuid.New().String())
	topic := "test-agent-topic"
	hostid := "test-host-id"
	agent := &model.Agent{
		UID:            string(uid),
		Topic:          topic,
		Hostid:         hostid,
		Hostname:       "test-host",
		OnlineDuration: 0,
		LastOnlineAt:   time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create agent
	agentID, err := s.adapter.CreateAgent(agent)
	s.Require().NoError(err, "failed to create agent")
	s.Positive(agentID, "agent ID should be set")

	// Get agent by hostid
	retrievedAgent, err := s.adapter.GetAgentByHostid(uid, topic, hostid)
	s.Require().NoError(err, "failed to get agent")
	s.Equal(agent.Hostname, retrievedAgent.Hostname, "hostname should match")

	// Get all agents
	agents, err := s.adapter.GetAgents()
	s.Require().NoError(err, "failed to get agents")
	s.GreaterOrEqual(len(agents), 1, "should have at least one agent")

	// Update last online at
	lastOnline := time.Now()
	err = s.adapter.UpdateAgentLastOnlineAt(uid, topic, hostid, lastOnline)
	s.Require().NoError(err, "failed to update last online at")

	// Update online duration
	offlineTime := lastOnline.Add(time.Hour)
	err = s.adapter.UpdateAgentOnlineDuration(uid, topic, hostid, offlineTime)
	s.Require().NoError(err, "failed to update online duration")
}

// TestAdapter_FileOperations tests file CRUD operations
func (s *AdapterTestSuite) TestAdapter_FileOperations() {
	uid := types.Uid(uuid.New().String())
	fileDef := &types.FileDef{
		ObjHeader: types.ObjHeader{
			Id: uuid.New().String(),
		},
		User:     string(uid),
		Name:     "test-file.txt",
		MimeType: "text/plain",
		Location: "/tmp/test-file.txt",
	}

	// Start upload
	err := s.adapter.FileStartUpload(fileDef)
	s.Require().NoError(err, "failed to start file upload")

	// Finish upload
	finishedFile, err := s.adapter.FileFinishUpload(fileDef, true, 1024)
	s.Require().NoError(err, "failed to finish file upload")
	s.NotNil(finishedFile, "finished file should not be nil")

	// Get file
	retrievedFile, err := s.adapter.FileGet(fileDef.Id)
	s.Require().NoError(err, "failed to get file")
	s.Equal(fileDef.Name, retrievedFile.Name, "name should match")
	s.Equal(int64(1024), retrievedFile.Size, "size should match")
}

// TestAdapter_Stats tests the Stats method
func (s *AdapterTestSuite) TestAdapter_Stats() {
	stats := s.adapter.Stats()
	s.Require().NotNil(stats, "stats should not be nil")

	sqlStats, ok := stats.(sql.DBStats)
	s.Require().True(ok, "stats should be of type sql.DBStats")
	s.GreaterOrEqual(sqlStats.OpenConnections, 0, "open connections should be >= 0")
}

// TestAdapter_SetMaxResults tests the SetMaxResults method
func (s *AdapterTestSuite) TestAdapter_SetMaxResults() {
	err := s.adapter.SetMaxResults(500)
	s.Require().NoError(err, "failed to set max results")
	s.Equal(500, s.adapter.maxResults, "max results should be 500")
}

// TestAdapter_DoubleOpen tests that opening an already opened adapter fails
func (s *AdapterTestSuite) TestAdapter_DoubleOpen() {
	storeConfig := config.StoreType{
		UseAdapter: "mysql",
		Adapters: map[string]any{
			"mysql": map[string]any{
				"dsn": "test",
			},
		},
	}

	err := s.adapter.Open(storeConfig)
	s.Require().Error(err, "should fail to open already opened adapter")
	s.Contains(err.Error(), "already connected", "error should indicate adapter is already connected")
}

// TestAdapter_InvalidConfig tests opening adapter with invalid config
func (s *AdapterTestSuite) TestAdapter_InvalidConfig() {
	newAdapter := &adapter{}

	// Test missing adapter name
	storeConfig := config.StoreType{
		UseAdapter: "",
	}
	err := newAdapter.Open(storeConfig)
	s.Require().Error(err, "should fail with missing adapter name")
	s.Contains(err.Error(), "adapter name missing", "error should indicate missing adapter name")

	// Test wrong adapter name
	storeConfig.UseAdapter = "postgres"
	err = newAdapter.Open(storeConfig)
	s.Require().Error(err, "should fail with wrong adapter name")
	s.Contains(err.Error(), "adapter name must be 'mysql'", "error should indicate wrong adapter name")
}

// Run the test suite
func TestAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(AdapterTestSuite))
}

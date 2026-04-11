package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	storeMigrate "github.com/flowline-io/flowbot/pkg/migrate"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
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
	require.NoError(s.T(), err, "failed to start MySQL container")
	s.container = container

	// Get connection string
	connStr, err := container.ConnectionString(s.ctx)
	require.NoError(s.T(), err, "failed to get connection string")

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
	require.NoError(s.T(), err, "failed to open adapter")

	// Run migrations
	err = s.runMigrations("")
	require.NoError(s.T(), err, "failed to run migrations")
}

// TearDownSuite cleans up the test suite
func (s *AdapterTestSuite) TearDownSuite() {
	if s.adapter != nil && s.adapter.IsOpen() {
		err := s.adapter.Close()
		require.NoError(s.T(), err, "failed to close adapter")
	}

	if s.container != nil {
		err := s.container.Terminate(s.ctx)
		require.NoError(s.T(), err, "failed to terminate container")
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
	assert.True(s.T(), s.adapter.IsOpen(), "adapter should be open")
	assert.Equal(s.T(), "mysql", s.adapter.GetName(), "adapter name should be mysql")
	assert.NotNil(s.T(), s.adapter.GetDB(), "DB should not be nil")
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
	require.NoError(s.T(), err, "failed to create user")
	assert.Greater(s.T(), user.ID, int64(0), "user ID should be set")

	// Get user by UID
	uid := types.Uid(user.Flag)
	retrievedUser, err := s.adapter.UserGet(uid)
	require.NoError(s.T(), err, "failed to get user")
	assert.Equal(s.T(), user.Flag, retrievedUser.Flag, "flag should match")
	assert.Equal(s.T(), user.Name, retrievedUser.Name, "name should match")

	// Get user by ID
	userByID, err := s.adapter.GetUserById(user.ID)
	require.NoError(s.T(), err, "failed to get user by ID")
	assert.Equal(s.T(), user.Flag, userByID.Flag, "flag should match")

	// Get user by flag
	userByFlag, err := s.adapter.GetUserByFlag(user.Flag)
	require.NoError(s.T(), err, "failed to get user by flag")
	assert.Equal(s.T(), user.Name, userByFlag.Name, "name should match")

	// Update user
	update := types.KV{
		"name": "Updated User",
	}
	err = s.adapter.UserUpdate(uid, update)
	require.NoError(s.T(), err, "failed to update user")

	// Verify update
	updatedUser, err := s.adapter.UserGet(uid)
	require.NoError(s.T(), err, "failed to get updated user")
	assert.Equal(s.T(), "Updated User", updatedUser.Name, "name should be updated")

	// Get all users
	users, err := s.adapter.GetUsers()
	require.NoError(s.T(), err, "failed to get users")
	assert.GreaterOrEqual(s.T(), len(users), 1, "should have at least one user")

	// Get first user
	firstUser, err := s.adapter.FirstUser()
	require.NoError(s.T(), err, "failed to get first user")
	assert.NotNil(s.T(), firstUser, "first user should not be nil")

	// Delete user
	err = s.adapter.UserDelete(uid, false)
	require.NoError(s.T(), err, "failed to delete user")

	// Verify deletion
	_, err = s.adapter.UserGet(uid)
	assert.Error(s.T(), err, "user should be deleted")
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
	require.NoError(s.T(), err, "failed to create bot")
	assert.Greater(s.T(), botID, int64(0), "bot ID should be set")

	// Get bot by ID
	retrievedBot, err := s.adapter.GetBot(botID)
	require.NoError(s.T(), err, "failed to get bot")
	assert.Equal(s.T(), bot.Name, retrievedBot.Name, "name should match")

	// Get bot by name
	botByName, err := s.adapter.GetBotByName(bot.Name)
	require.NoError(s.T(), err, "failed to get bot by name")
	assert.Equal(s.T(), botID, botByName.ID, "ID should match")

	// Get all bots
	bots, err := s.adapter.GetBots()
	require.NoError(s.T(), err, "failed to get bots")
	assert.GreaterOrEqual(s.T(), len(bots), 1, "should have at least one bot")

	// Update bot
	bot.State = model.BotInactive
	err = s.adapter.UpdateBot(bot)
	require.NoError(s.T(), err, "failed to update bot")

	// Verify update
	updatedBot, err := s.adapter.GetBot(botID)
	require.NoError(s.T(), err, "failed to get updated bot")
	assert.Equal(s.T(), model.BotInactive, updatedBot.State, "state should be updated")

	// Delete bot
	err = s.adapter.DeleteBot(bot.Name)
	require.NoError(s.T(), err, "failed to delete bot")

	// Verify deletion
	_, err = s.adapter.GetBotByName(bot.Name)
	assert.Error(s.T(), err, "bot should be deleted")
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
	require.NoError(s.T(), err, "failed to create platform")
	assert.Greater(s.T(), platformID, int64(0), "platform ID should be set")

	// Get platform by ID
	retrievedPlatform, err := s.adapter.GetPlatform(platformID)
	require.NoError(s.T(), err, "failed to get platform")
	assert.Equal(s.T(), platform.Name, retrievedPlatform.Name, "name should match")

	// Get platform by name
	platformByName, err := s.adapter.GetPlatformByName(platform.Name)
	require.NoError(s.T(), err, "failed to get platform by name")
	assert.Equal(s.T(), platformID, platformByName.ID, "ID should match")

	// Get all platforms
	platforms, err := s.adapter.GetPlatforms()
	require.NoError(s.T(), err, "failed to get platforms")
	assert.GreaterOrEqual(s.T(), len(platforms), 1, "should have at least one platform")
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
	require.NoError(s.T(), err, "failed to create channel")
	assert.Greater(s.T(), channelID, int64(0), "channel ID should be set")

	// Get channel by ID
	retrievedChannel, err := s.adapter.GetChannel(channelID)
	require.NoError(s.T(), err, "failed to get channel")
	assert.Equal(s.T(), channel.Name, retrievedChannel.Name, "name should match")

	// Get channel by name
	channelByName, err := s.adapter.GetChannelByName(channel.Name)
	require.NoError(s.T(), err, "failed to get channel by name")
	assert.Equal(s.T(), channelID, channelByName.ID, "ID should match")

	// Get all channels
	channels, err := s.adapter.GetChannels()
	require.NoError(s.T(), err, "failed to get channels")
	assert.GreaterOrEqual(s.T(), len(channels), 1, "should have at least one channel")

	// Update channel
	channel.State = model.ChannelInactive
	err = s.adapter.UpdateChannel(channel)
	require.NoError(s.T(), err, "failed to update channel")

	// Verify update
	updatedChannel, err := s.adapter.GetChannel(channelID)
	require.NoError(s.T(), err, "failed to get updated channel")
	assert.Equal(s.T(), model.ChannelInactive, updatedChannel.State, "state should be updated")

	// Delete channel
	err = s.adapter.DeleteChannel(channel.Name)
	require.NoError(s.T(), err, "failed to delete channel")

	// Verify deletion
	_, err = s.adapter.GetChannelByName(channel.Name)
	assert.Error(s.T(), err, "channel should be deleted")
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
	require.NoError(s.T(), err, "failed to create user")

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
	require.NoError(s.T(), err, "failed to create platform user")
	assert.Greater(s.T(), platformUserID, int64(0), "platform user ID should be set")

	// Get platform user by flag
	retrievedPlatformUser, err := s.adapter.GetPlatformUserByFlag(platformUser.Flag)
	require.NoError(s.T(), err, "failed to get platform user")
	assert.Equal(s.T(), platformUser.UserID, retrievedPlatformUser.UserID, "user ID should match")

	// Get platform users by user ID
	platformUsers, err := s.adapter.GetPlatformUsersByUserId(user.ID)
	require.NoError(s.T(), err, "failed to get platform users by user ID")
	assert.Len(s.T(), platformUsers, 1, "should have one platform user")

	// Update platform user
	platformUser.Name = "Updated Platform User"
	err = s.adapter.UpdatePlatformUser(platformUser)
	require.NoError(s.T(), err, "failed to update platform user")
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
	require.NoError(s.T(), err, "failed to create platform channel")
	assert.Greater(s.T(), platformChannelID, int64(0), "platform channel ID should be set")

	// Get platform channel by flag
	retrievedChannel, err := s.adapter.GetPlatformChannelByFlag(platformChannel.Flag)
	require.NoError(s.T(), err, "failed to get platform channel")
	assert.Equal(s.T(), platformChannel.PlatformID, retrievedChannel.PlatformID, "platform ID should match")

	// Get platform channels by platform IDs
	channels, err := s.adapter.GetPlatformChannelsByPlatformIds([]int64{1})
	require.NoError(s.T(), err, "failed to get platform channels by platform IDs")
	assert.GreaterOrEqual(s.T(), len(channels), 1, "should have at least one channel")

	// Get platform channel by channel ID
	channelByID, err := s.adapter.GetPlatformChannelsByChannelId(1)
	require.NoError(s.T(), err, "failed to get platform channel by channel ID")
	assert.Equal(s.T(), platformChannel.Flag, channelByID.Flag, "flag should match")
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
	require.NoError(s.T(), err, "failed to create platform channel user")
	assert.Greater(s.T(), platformChannelUserID, int64(0), "platform channel user ID should be set")

	// Get platform channel users by user flag
	users, err := s.adapter.GetPlatformChannelUsersByUserFlag(platformChannelUser.UserFlag)
	require.NoError(s.T(), err, "failed to get platform channel users")
	assert.Len(s.T(), users, 1, "should have one platform channel user")
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
	require.NoError(s.T(), err, "failed to create message")

	// Get message by flag
	retrievedMessage, err := s.adapter.GetMessage(message.Flag)
	require.NoError(s.T(), err, "failed to get message")
	assert.Equal(s.T(), message.Session, retrievedMessage.Session, "session should match")

	// Get message by platform
	messageByPlatform, err := s.adapter.GetMessageByPlatform(message.PlatformID, message.PlatformMsgID)
	require.NoError(s.T(), err, "failed to get message by platform")
	assert.Equal(s.T(), message.Flag, messageByPlatform.Flag, "flag should match")

	// Get messages by session
	messages, err := s.adapter.GetMessagesBySession(message.Session)
	require.NoError(s.T(), err, "failed to get messages by session")
	assert.Len(s.T(), messages, 1, "should have one message")
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
	require.NoError(s.T(), err, "failed to set data")

	// Get data
	retrievedValue, err := s.adapter.DataGet(uid, topic, key)
	require.NoError(s.T(), err, "failed to get data")
	assert.Equal(s.T(), value["field1"], retrievedValue["field1"], "field1 should match")
	// JSON unmarshaling converts numbers to float64
	assert.Equal(s.T(), float64(42), retrievedValue["field2"], "field2 should match")

	// List data
	filter := types.DataFilter{}
	dataList, err := s.adapter.DataList(uid, topic, filter)
	require.NoError(s.T(), err, "failed to list data")
	assert.Len(s.T(), dataList, 1, "should have one data entry")

	// Delete data
	err = s.adapter.DataDelete(uid, topic, key)
	require.NoError(s.T(), err, "failed to delete data")

	// Verify deletion
	_, err = s.adapter.DataGet(uid, topic, key)
	assert.Error(s.T(), err, "data should be deleted")
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
	require.NoError(s.T(), err, "failed to set config")

	// Get config
	retrievedValue, err := s.adapter.ConfigGet(uid, topic, key)
	require.NoError(s.T(), err, "failed to get config")
	assert.Equal(s.T(), value["setting1"], retrievedValue["setting1"], "setting1 should match")

	// List config by prefix
	configs, err := s.adapter.ListConfigByPrefix(uid, topic, "test")
	require.NoError(s.T(), err, "failed to list configs")
	assert.GreaterOrEqual(s.T(), len(configs), 1, "should have at least one config")

	// Delete config
	err = s.adapter.ConfigDelete(uid, topic, key)
	require.NoError(s.T(), err, "failed to delete config")

	// Verify deletion
	_, err = s.adapter.ConfigGet(uid, topic, key)
	assert.Error(s.T(), err, "config should be deleted")
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
	require.NoError(s.T(), err, "failed to set OAuth")

	// Get OAuth
	retrievedOAuth, err := s.adapter.OAuthGet(uid, oauth.Topic, oauth.Type)
	require.NoError(s.T(), err, "failed to get OAuth")
	assert.Equal(s.T(), oauth.Token, retrievedOAuth.Token, "token should match")

	// Get available OAuth
	availableOAuth, err := s.adapter.OAuthGetAvailable(oauth.Type)
	require.NoError(s.T(), err, "failed to get available OAuth")
	assert.GreaterOrEqual(s.T(), len(availableOAuth), 1, "should have at least one OAuth")
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
	require.NoError(s.T(), err, "failed to set form")

	// Get form
	retrievedForm, err := s.adapter.FormGet(formID)
	require.NoError(s.T(), err, "failed to get form")
	assert.Equal(s.T(), form.UID, retrievedForm.UID, "UID should match")
	assert.Equal(s.T(), form.State, retrievedForm.State, "state should match")
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
	require.NoError(s.T(), err, "failed to set page")

	// Get page
	retrievedPage, err := s.adapter.PageGet(pageID)
	require.NoError(s.T(), err, "failed to get page")
	assert.Equal(s.T(), page.UID, retrievedPage.UID, "UID should match")
	assert.Equal(s.T(), page.State, retrievedPage.State, "state should match")
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
	require.NoError(s.T(), err, "failed to set behavior")

	// Get behavior
	retrievedBehavior, err := s.adapter.BehaviorGet(uid, flag)
	require.NoError(s.T(), err, "failed to get behavior")
	assert.Equal(s.T(), behavior.Count_, retrievedBehavior.Count_, "count should match")

	// List behaviors
	behaviors, err := s.adapter.BehaviorList(uid)
	require.NoError(s.T(), err, "failed to list behaviors")
	assert.Len(s.T(), behaviors, 1, "should have one behavior")

	// Increase behavior count
	err = s.adapter.BehaviorIncrease(uid, flag, 5)
	require.NoError(s.T(), err, "failed to increase behavior count")

	// Verify increase
	updatedBehavior, err := s.adapter.BehaviorGet(uid, flag)
	require.NoError(s.T(), err, "failed to get updated behavior")
	assert.Equal(s.T(), int32(6), updatedBehavior.Count_, "count should be increased by 5")
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
	require.NoError(s.T(), err, "failed to set parameter")

	// Get parameter
	retrievedParam, err := s.adapter.ParameterGet(flag)
	require.NoError(s.T(), err, "failed to get parameter")
	assert.False(s.T(), retrievedParam.IsExpired(), "parameter should not be expired")

	// Delete parameter
	err = s.adapter.ParameterDelete(flag)
	require.NoError(s.T(), err, "failed to delete parameter")

	// Verify deletion
	_, err = s.adapter.ParameterGet(flag)
	assert.Error(s.T(), err, "parameter should be deleted")
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
	require.NoError(s.T(), err, "failed to create instruct")
	assert.Greater(s.T(), instructID, int64(0), "instruct ID should be set")

	// List instructs (not expired)
	instructs, err := s.adapter.ListInstruct(uid, false, 10)
	require.NoError(s.T(), err, "failed to list instructs")
	assert.Len(s.T(), instructs, 1, "should have one instruct")

	// Update instruct
	instruct.State = model.InstructDone
	err = s.adapter.UpdateInstruct(instruct)
	require.NoError(s.T(), err, "failed to update instruct")
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
	require.NoError(s.T(), err, "failed to create webhook")
	assert.Greater(s.T(), webhookID, int64(0), "webhook ID should be set")

	// List webhooks
	webhooks, err := s.adapter.ListWebhook(uid)
	require.NoError(s.T(), err, "failed to list webhooks")
	assert.Len(s.T(), webhooks, 1, "should have one webhook")

	// Get webhook by secret
	webhookBySecret, err := s.adapter.GetWebhookBySecret(webhook.Secret)
	require.NoError(s.T(), err, "failed to get webhook by secret")
	assert.Equal(s.T(), webhook.Flag, webhookBySecret.Flag, "flag should match")

	// Get webhook by UID and flag
	webhookByUidAndFlag, err := s.adapter.GetWebhookByUidAndFlag(uid, webhook.Flag)
	require.NoError(s.T(), err, "failed to get webhook by UID and flag")
	assert.Equal(s.T(), webhook.Topic, webhookByUidAndFlag.Topic, "topic should match")

	// Increase webhook count
	err = s.adapter.IncreaseWebhookCount(webhookID)
	require.NoError(s.T(), err, "failed to increase webhook count")

	// Update webhook
	webhook.State = model.WebhookInactive
	err = s.adapter.UpdateWebhook(webhook)
	require.NoError(s.T(), err, "failed to update webhook")

	// Delete webhook
	err = s.adapter.DeleteWebhook(webhookID)
	require.NoError(s.T(), err, "failed to delete webhook")
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
	require.NoError(s.T(), err, "failed to create counter")
	assert.Greater(s.T(), counterID, int64(0), "counter ID should be set")

	// Get counter
	retrievedCounter, err := s.adapter.GetCounter(counterID)
	require.NoError(s.T(), err, "failed to get counter")
	assert.Equal(s.T(), counter.Digit, retrievedCounter.Digit, "digit should match")

	// Get counter by flag
	counterByFlag, err := s.adapter.GetCounterByFlag(uid, topic, counter.Flag)
	require.NoError(s.T(), err, "failed to get counter by flag")
	assert.Equal(s.T(), counter.Digit, counterByFlag.Digit, "digit should match")

	// List counters
	counters, err := s.adapter.ListCounter(uid, topic)
	require.NoError(s.T(), err, "failed to list counters")
	assert.Len(s.T(), counters, 1, "should have one counter")

	// Increase counter
	err = s.adapter.IncreaseCounter(counterID, 50)
	require.NoError(s.T(), err, "failed to increase counter")

	// Verify increase
	updatedCounter, err := s.adapter.GetCounter(counterID)
	require.NoError(s.T(), err, "failed to get updated counter")
	assert.Equal(s.T(), int64(150), updatedCounter.Digit, "digit should be 150")

	// Decrease counter
	err = s.adapter.DecreaseCounter(counterID, 30)
	require.NoError(s.T(), err, "failed to decrease counter")

	// Verify decrease
	decreasedCounter, err := s.adapter.GetCounter(counterID)
	require.NoError(s.T(), err, "failed to get decreased counter")
	assert.Equal(s.T(), int64(120), decreasedCounter.Digit, "digit should be 120")
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
	require.NoError(s.T(), err, "failed to create agent")
	assert.Greater(s.T(), agentID, int64(0), "agent ID should be set")

	// Get agent by hostid
	retrievedAgent, err := s.adapter.GetAgentByHostid(uid, topic, hostid)
	require.NoError(s.T(), err, "failed to get agent")
	assert.Equal(s.T(), agent.Hostname, retrievedAgent.Hostname, "hostname should match")

	// Get all agents
	agents, err := s.adapter.GetAgents()
	require.NoError(s.T(), err, "failed to get agents")
	assert.GreaterOrEqual(s.T(), len(agents), 1, "should have at least one agent")

	// Update last online at
	lastOnline := time.Now()
	err = s.adapter.UpdateAgentLastOnlineAt(uid, topic, hostid, lastOnline)
	require.NoError(s.T(), err, "failed to update last online at")

	// Update online duration
	offlineTime := lastOnline.Add(time.Hour)
	err = s.adapter.UpdateAgentOnlineDuration(uid, topic, hostid, offlineTime)
	require.NoError(s.T(), err, "failed to update online duration")
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
	require.NoError(s.T(), err, "failed to start file upload")

	// Finish upload
	finishedFile, err := s.adapter.FileFinishUpload(fileDef, true, 1024)
	require.NoError(s.T(), err, "failed to finish file upload")
	assert.NotNil(s.T(), finishedFile, "finished file should not be nil")

	// Get file
	retrievedFile, err := s.adapter.FileGet(fileDef.Id)
	require.NoError(s.T(), err, "failed to get file")
	assert.Equal(s.T(), fileDef.Name, retrievedFile.Name, "name should match")
	assert.Equal(s.T(), int64(1024), retrievedFile.Size, "size should match")
}

// TestAdapter_Stats tests the Stats method
func (s *AdapterTestSuite) TestAdapter_Stats() {
	stats := s.adapter.Stats()
	require.NotNil(s.T(), stats, "stats should not be nil")

	sqlStats, ok := stats.(sql.DBStats)
	require.True(s.T(), ok, "stats should be of type sql.DBStats")
	assert.GreaterOrEqual(s.T(), sqlStats.OpenConnections, 0, "open connections should be >= 0")
}

// TestAdapter_SetMaxResults tests the SetMaxResults method
func (s *AdapterTestSuite) TestAdapter_SetMaxResults() {
	err := s.adapter.SetMaxResults(500)
	require.NoError(s.T(), err, "failed to set max results")
	assert.Equal(s.T(), 500, s.adapter.maxResults, "max results should be 500")
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
	assert.Error(s.T(), err, "should fail to open already opened adapter")
	assert.Contains(s.T(), err.Error(), "already connected", "error should indicate adapter is already connected")
}

// TestAdapter_InvalidConfig tests opening adapter with invalid config
func (s *AdapterTestSuite) TestAdapter_InvalidConfig() {
	newAdapter := &adapter{}

	// Test missing adapter name
	storeConfig := config.StoreType{
		UseAdapter: "",
	}
	err := newAdapter.Open(storeConfig)
	assert.Error(s.T(), err, "should fail with missing adapter name")
	assert.Contains(s.T(), err.Error(), "adapter name missing", "error should indicate missing adapter name")

	// Test wrong adapter name
	storeConfig.UseAdapter = "postgres"
	err = newAdapter.Open(storeConfig)
	assert.Error(s.T(), err, "should fail with wrong adapter name")
	assert.Contains(s.T(), err.Error(), "adapter name must be 'mysql'", "error should indicate wrong adapter name")
}

// Run the test suite
func TestAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(AdapterTestSuite))
}

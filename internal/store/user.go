package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformuser"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/user"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

// UserStore persists users and platform user mappings.
type UserStore struct {
	client *gen.Client
}

// NewUserStore creates a UserStore with the given ent client.
func NewUserStore(client *gen.Client) *UserStore {
	return &UserStore{client: client}
}

// UserStoreFromDB returns a UserStore using the global database client.
func UserStoreFromDB() *UserStore {
	return NewUserStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *UserStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// UserCreate creates a new user.
func (s *UserStore) UserCreate(ctx context.Context, usr *gen.User) error {
	u, err := s.client.User.Create().
		SetFlag(usr.Flag).
		SetName(usr.Name).
		SetTags(usr.Tags).
		SetState(int(usr.State)).
		SetCreatedAt(usr.CreatedAt).
		SetUpdatedAt(usr.UpdatedAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create user: %w", err)
	}
	usr.ID = u.ID
	return nil
}

// UserGet returns the user for uid.
func (s *UserStore) UserGet(ctx context.Context, uid types.Uid) (*gen.User, error) {
	u, err := s.client.User.Query().Where(user.FlagEQ(uid.String())).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get user: %w", err)
	}
	return u, nil
}

// GetUsers returns all users up to the query limit.
func (s *UserStore) GetUsers(ctx context.Context) ([]*gen.User, error) {
	users, err := s.client.User.Query().Limit(queryMaxResults()).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get all users: %w", err)
	}
	result := make([]*gen.User, len(users))
	copy(result, users)
	return result, nil
}

// UserGetAll returns users, optionally filtered by uid flags.
func (s *UserStore) UserGetAll(ctx context.Context, ids ...types.Uid) ([]*gen.User, error) {
	if len(ids) == 0 {
		return s.GetUsers(ctx)
	}
	flags := make([]string, len(ids))
	for i, id := range ids {
		flags[i] = id.String()
	}
	users, err := s.client.User.Query().
		Where(user.FlagIn(flags...)).
		Limit(queryMaxResults()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get all users: %w", err)
	}
	result := make([]*gen.User, len(users))
	copy(result, users)
	return result, nil
}

// FirstUser returns the first user.
func (s *UserStore) FirstUser(ctx context.Context) (*gen.User, error) {
	u, err := s.client.User.Query().Order(gen.Asc(user.FieldID)).First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: first user: %w", err)
	}
	return u, nil
}

// UserDelete soft- or hard-deletes the user for uid.
func (s *UserStore) UserDelete(ctx context.Context, uid types.Uid, hard bool) error {
	if hard {
		_, err := s.client.User.Delete().Where(user.FlagEQ(uid.String())).Exec(ctx)
		if err != nil {
			return fmt.Errorf("postgres: hard delete user: %w", err)
		}
		return nil
	}
	_, err := s.client.User.Update().Where(user.FlagEQ(uid.String())).
		SetState(int(int(schema.UserInactive))).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: soft delete user: %w", err)
	}
	return nil
}

// UserUpdate applies partial updates to the user for uid.
func (s *UserStore) UserUpdate(ctx context.Context, uid types.Uid, update types.KV) error {
	u := s.client.User.Update().Where(user.FlagEQ(uid.String()))
	if v, ok := update.String("name"); ok {
		u = u.SetName(v)
	}
	if v, ok := update.String("tags"); ok {
		u = u.SetTags(v)
	}
	if v, ok := update.Int64("state"); ok {
		u = u.SetState(int(v))
	}
	u = u.SetUpdatedAt(time.Now())
	_, err := u.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update user: %w", err)
	}
	return nil
}

// GetUserById returns the user with the given id.
func (s *UserStore) GetUserById(ctx context.Context, id int64) (*gen.User, error) {
	u, err := s.client.User.Query().Where(user.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get user by id: %w", err)
	}
	return u, nil
}

// GetUserByFlag returns the user by flag.
func (s *UserStore) GetUserByFlag(ctx context.Context, flag string) (*gen.User, error) {
	return s.UserGet(ctx, types.Uid(flag))
}

// normalizePlatformUserProfile fills required profile fields when callers omit them.
func normalizePlatformUserProfile(item *gen.PlatformUser) {
	if item == nil {
		return
	}
	if item.Email == "" {
		flag := item.Flag
		if flag == "" {
			flag = "user"
		}
		item.Email = fmt.Sprintf("%s@unknown.local", flag)
	}
	if item.AvatarURL == "" {
		item.AvatarURL = "-"
	}
}

// CreatePlatformUser creates a platform user record.
func (s *UserStore) CreatePlatformUser(ctx context.Context, item *gen.PlatformUser) (int64, error) {
	normalizePlatformUserProfile(item)
	u, err := s.client.PlatformUser.Create().
		SetPlatformID(item.PlatformID).
		SetUserID(item.UserID).
		SetFlag(item.Flag).
		SetName(item.Name).
		SetEmail(item.Email).
		SetAvatarURL(item.AvatarURL).
		SetIsBot(item.IsBot).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create platform user: %w", err)
	}
	return u.ID, nil
}

// GetPlatformUsersByUserId returns the platform users by user id.
func (s *UserStore) GetPlatformUsersByUserId(ctx context.Context, userId int64) ([]*gen.PlatformUser, error) {
	users, err := s.client.PlatformUser.Query().Where(platformuser.UserIDEQ(userId)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform users by user id: %w", err)
	}
	result := make([]*gen.PlatformUser, len(users))
	copy(result, users)
	return result, nil
}

// GetPlatformUserByFlag returns the platform user by flag.
func (s *UserStore) GetPlatformUserByFlag(ctx context.Context, flag string) (*gen.PlatformUser, error) {
	u, err := s.client.PlatformUser.Query().Where(platformuser.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform user by flag: %w", err)
	}
	return u, nil
}

// UpdatePlatformUser updates the platform user.
func (s *UserStore) UpdatePlatformUser(ctx context.Context, item *gen.PlatformUser) error {
	_, err := s.client.PlatformUser.Update().Where(platformuser.IDEQ(item.ID)).
		SetPlatformID(item.PlatformID).
		SetUserID(item.UserID).
		SetFlag(item.Flag).
		SetName(item.Name).
		SetEmail(item.Email).
		SetAvatarURL(item.AvatarURL).
		SetIsBot(item.IsBot).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: update platform user: %w", err)
	}
	return nil
}


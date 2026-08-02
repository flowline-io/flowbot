// Package store provides database storage implementations.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// Client is a type alias for the Ent client.
type Client = gen.Client

var adp Adapter

var availableAdapters = make(map[string]Adapter)

func openAdapter(jsonConfig config.StoreType) error {
	if adp == nil {
		if len(availableAdapters) >= 1 {
			// Default to the only entry in availableAdapters.
			for _, v := range availableAdapters {
				adp = v
			}
		} else {
			return errors.New("store: db adapter is not specified. Please set postgres.dsn in flowbot.yaml")
		}
	}

	if adp.IsOpen() {
		return errors.New("store: connection is already opened")
	}

	return adp.Open(jsonConfig)
}

func RegisterAdapter(a Adapter) {
	if a == nil {
		flog.Fatal("store: Register adapter is nil")
	}

	name := a.GetName()
	if _, ok := availableAdapters[name]; ok {
		flog.Fatal("store: adapter %s is already registered", name)
	}
	availableAdapters[name] = a
	flog.Info("store: adapter '%s' registered", name)
}

func Migrate() error {
	if !adp.IsOpen() {
		return errors.New("store: connection is not opened")
	}
	client, ok := adp.GetDB().(*gen.Client)
	if !ok {
		return errors.New("store: failed to get Ent client from adapter")
	}
	err := client.Schema.Create(context.Background())
	if err != nil {
		return fmt.Errorf("store: schema migration: %w", err)
	}
	return nil
}

// PersistentStorageInterface defines methods used for interaction with persistent storage.
type PersistentStorageInterface interface {
	Open(jsonConfig config.StoreType) error
	Close() error
	IsOpen() bool
	GetAdapter() Adapter
	DbStats() func() any
}

// Store is the main object for interacting with persistent storage.
var Store PersistentStorageInterface

type storeObj struct{}

func (storeObj) Open(jsonConfig config.StoreType) error {
	return openAdapter(jsonConfig)
}

func (storeObj) Close() error {
	if adp.IsOpen() {
		return adp.Close()
	}

	return nil
}

func (storeObj) GetAdapter() Adapter {
	return adp
}

// IsOpen checks if persistent storage connection has been initialized.
func (storeObj) IsOpen() bool {
	if adp != nil {
		return adp.IsOpen()
	}

	return false
}

func (s storeObj) DbStats() func() any {
	if !s.IsOpen() {
		return nil
	}
	return adp.Stats
}


// Adapter is the database connection facade (open/close/ping/client).
// Domain persistence lives on *Store types (ChatStore, AgentStore, …).
type Adapter interface {
	// Open and configure the adapter
	Open(storeConfig config.StoreType) error
	// Close the adapter
	Close() error
	// IsOpen checks if the adapter is ready for use
	IsOpen() bool
	// GetName returns the name of the adapter
	GetName() string
	// Stats returns the DB connection stats object.
	Stats() any
	// Ping checks database connectivity and returns the round-trip latency.
	Ping(ctx context.Context) (time.Duration, error)
	// GetDB returns the underlying DB connection (ent client as any).
	GetDB() any
	// GetClient returns the ent client.
	GetClient() *gen.Client
}

var Database Adapter

func Init() {
	Store = storeObj{}
	pgAdapter, ok := availableAdapters["postgres"]
	if !ok {
		flog.Fatal("postgres adapter not available - check build tags")
	}
	Database = pgAdapter
}

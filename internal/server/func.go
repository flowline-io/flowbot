package server

import (
	"github.com/go-playground/validator/v10"

	"github.com/flowline-io/flowbot/pkg/cache"
)

var cacheStore *cache.RedisStore

// SetCacheStore sets the cache store for server functions.
func SetCacheStore(s *cache.RedisStore) {
	cacheStore = s
}

type structValidator struct {
	validate *validator.Validate
}

// Validator needs to implement the Validate method
func (v *structValidator) Validate(out any) error {
	return v.validate.Struct(out)
}

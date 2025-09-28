package utils

import (
	"time"

	"github.com/google/uuid"
	"resty.dev/v3"
)

func DefaultRestyClient() *resty.Client {
	c := resty.New()
	c.SetDisableWarn(true)
	c.SetTimeout(time.Minute)
	c.AddContentTypeEncoder("json", EncodeJSON)
	c.AddContentTypeDecoder("json", DecodeJSON)

	traceId := uuid.New().String()
	c.SetHeader("X-Trace-Id", traceId)

	return c
}

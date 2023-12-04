package flog

import (
	"context"
	"fmt"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"time"
)

var GormLogger = &gormLogger{}

type gormLogger struct{}

func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	switch level {
	case logger.Silent:
	case logger.Info:
		SetLevel("info")
	case logger.Warn:
		SetLevel("warn")
	case logger.Error:
		SetLevel("error")
	}

	return g
}

func (g *gormLogger) Info(_ context.Context, s string, i ...interface{}) {
	Info(s, i...)
}

func (g *gormLogger) Warn(_ context.Context, s string, i ...interface{}) {
	Warn(s, i...)
}

func (g *gormLogger) Error(_ context.Context, s string, i ...interface{}) {
	l.Error().Caller(1).Stack().Msgf(s, i...)
}

func (g *gormLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	elapsed := time.Since(begin)
	l.Debug().
		Int64("rows", rows).
		Str("elapsed", fmt.Sprintf("%fms", float64(elapsed.Nanoseconds())/1e6)).
		Msgf("%s > %s", utils.FileWithLineNum(), sql)
}

package flog

import (
	"context"
	"fmt"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"time"
)

func NewGormLogger(level string) *GormLogger {
	return &GormLogger{level: level}
}

type GormLogger struct {
	level string
}

func (g *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
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

func (g *GormLogger) Info(_ context.Context, s string, i ...interface{}) {
	Info(s, i...)
}

func (g *GormLogger) Warn(_ context.Context, s string, i ...interface{}) {
	Warn(s, i...)
}

func (g *GormLogger) Error(_ context.Context, s string, i ...interface{}) {
	l.Error().Caller(1).Stack().Msgf(s, i...)
}

func (g *GormLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if g.level == DebugLevel { // fixme
		return
	}

	if !(g.level == DebugLevel || g.level == InfoLevel) {
		return
	}

	sql, rows := fc()
	elapsed := time.Since(begin)
	elapsedMs := float64(elapsed.Nanoseconds()) / 1e6

	if g.level == InfoLevel {
		if elapsedMs <= 1000 {
			return
		}
	}

	switch g.level {
	case DebugLevel:
		l.Debug().
			Int64("rows", rows).
			Str("elapsed", fmt.Sprintf("%fms", elapsedMs)).
			Msgf("%s > %s", utils.FileWithLineNum(), sql)
	case InfoLevel:
		l.Info().
			Int64("rows", rows).
			Str("elapsed", fmt.Sprintf("%fms", elapsedMs)).
			Msgf("%s > %s", utils.FileWithLineNum(), sql)
	}
}
